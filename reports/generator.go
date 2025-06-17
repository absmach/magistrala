// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const defaultTemplatePath = "/report_template.html"

type ReportData struct {
	Title         string
	GeneratedTime string
	GeneratedDate string
	Reports       []Report
}

func generatePDFReportWithDefault(ctx context.Context, title string, reports []Report) ([]byte, error) {
	for i := range reports {
		sort.Slice(reports[i].Messages, func(j, k int) bool {
			return reports[i].Messages[j].Time < reports[i].Messages[k].Time
		})
	}

	now := time.Now().UTC()
	data := ReportData{
		Title:         title,
		GeneratedTime: now.Format("15:04:05"),
		GeneratedDate: now.Format("02 Jan 2006"),
		Reports:       reports,
	}

	templateContent, err := readTemplateFile(defaultTemplatePath)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("failed to read template file: %w", err))
	}

	return generator(ctx, templateContent, data)
}

func generatePDFReportWithCustom(ctx context.Context, title string, reports []Report, customTemplate ReportTemplate) ([]byte, error) {
	for i := range reports {
		sort.Slice(reports[i].Messages, func(j, k int) bool {
			return reports[i].Messages[j].Time < reports[i].Messages[k].Time
		})
	}

	now := time.Now().UTC()
	data := ReportData{
		Title:         title,
		GeneratedTime: now.Format("15:04:05"),
		GeneratedDate: now.Format("02 Jan 2006"),
		Reports:       reports,
	}

	var templateContent string

	if customTemplate.String() != "" {
		templateContent = customTemplate.String()
	} else {
		return generatePDFReportWithDefault(ctx, title, reports)
	}

	return generator(ctx, templateContent, data)
}

func generator(ctx context.Context, templateContent string, data ReportData) ([]byte, error) {
	tmpl := template.New("report").Funcs(template.FuncMap{
		"formatTime":  formatTime,
		"formatValue": formatValue,
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
	})

	tmpl, err := tmpl.Parse(templateContent)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("failed to parse template: %w", err))
	}

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, data); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("failed to execute template: %w", err))
	}

	htmlContent := htmlBuf.String()
	pdfBytes, err := htmlToPDF(ctx, htmlContent)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("failed to convert HTML to PDF: %w", err))
	}

	return pdfBytes, nil
}

func htmlToPDF(ctx context.Context, htmlContent string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/chromium-browser"),
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.NoFirstRun,
		chromedp.Headless,
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	var pdfBuffer []byte

	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.Navigate("data:text/html,"+url.PathEscape(htmlContent)),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuffer, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).  // A4 width in inches
				WithPaperHeight(11.7). // A4 height in inches
				WithMarginTop(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				WithPreferCSSPageSize(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp execution failed: %w", err)
	}

	return pdfBuffer, nil
}

func readTemplateFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open template file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	return string(content), nil
}

func formatTime(t float64) string {
	if t > 9999999999 {
		return time.Unix(0, int64(t)).Format("2006-01-02 15:04:05")
	}
	return time.Unix(int64(t), 0).Format("2006-01-02 15:04:05")
}

func formatValue(msg senml.Message) string {
	switch {
	case msg.Value != nil:
		return fmt.Sprintf("%.2f", *msg.Value)
	case msg.StringValue != nil:
		return *msg.StringValue
	case msg.BoolValue != nil:
		return fmt.Sprintf("%t", *msg.BoolValue)
	case msg.DataValue != nil:
		return *msg.DataValue
	default:
		return "N/A"
	}
}

func generateCSVReport(_ context.Context, title string, reports []Report) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{"Time", "Value", "Unit", "Protocol", "Subtopic"}

	for i, report := range reports {
		if i > 0 {
			if err := writer.Write([]string{""}); err != nil {
				return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
			}
			if err := writer.Write([]string{"=== NEW REPORT ==="}); err != nil {
				return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
			}
			if err := writer.Write([]string{""}); err != nil {
				return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
			}
		} else {
			if err := writer.Write([]string{title}); err != nil {
				return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
			}
			if err := writer.Write([]string{""}); err != nil {
				return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
			}
		}

		if err := writer.Write([]string{"Report Information:"}); err != nil {
			return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		if err := writer.Write([]string{"Name", report.Metric.Name}); err != nil {
			return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		if report.Metric.ClientID != "" {
			if err := writer.Write([]string{"Device ID", report.Metric.ClientID}); err != nil {
				return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
			}
		}
		if err := writer.Write([]string{"Channel ID", report.Metric.ChannelID}); err != nil {
			return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		if err := writer.Write([]string{""}); err != nil {
			return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		if err := writer.Write(headers); err != nil {
			return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		sort.Slice(report.Messages, func(i, j int) bool {
			return report.Messages[i].Time < report.Messages[j].Time
		})

		for _, msg := range report.Messages {
			timeStr := formatTime(msg.Time)

			var valueStr string
			if msg.Value != nil {
				valueStr = fmt.Sprintf("%.2f", *msg.Value)
			} else if msg.StringValue != nil {
				valueStr = *msg.StringValue
			} else if msg.BoolValue != nil {
				valueStr = fmt.Sprintf("%v", *msg.BoolValue)
			} else if msg.DataValue != nil {
				valueStr = *msg.DataValue
			} else {
				valueStr = "N/A"
			}

			row := []string{
				timeStr,
				valueStr,
				msg.Unit,
				msg.Protocol,
				msg.Subtopic,
			}

			if err := writer.Write(row); err != nil {
				return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return buf.Bytes(), nil
}
