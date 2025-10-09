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
	"log/slog"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // Embed timezone database

	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

const nanosecondThreshold = float64(10 * time.Second / time.Nanosecond)

type ReportData struct {
	Title         string
	GeneratedTime string
	GeneratedDate string
	Reports       []Report
	Timezone      string
}

func (r *report) generatePDFReport(ctx context.Context, title string, reports []Report, template ReportTemplate, timezone string) ([]byte, error) {
	for i := range reports {
		sort.Slice(reports[i].Messages, func(j, k int) bool {
			return reports[i].Messages[j].Time < reports[i].Messages[k].Time
		})
	}

	loc, err := resolveTimezone(timezone)
	if err != nil {
		r.runInfo <- pkglog.RunInfo{
			Level:   slog.LevelWarn,
			Message: fmt.Sprintf("failed to resolve timezone '%s', falling back to UTC: %s", timezone, err),
			Details: []slog.Attr{
				slog.String("report_title", title),
				slog.Time("time", time.Now().UTC()),
			},
		}
	}

	now := time.Now().In(loc)
	displayTZ := timezone
	if strings.TrimSpace(displayTZ) == "" {
		displayTZ = "UTC"
	}

	data := ReportData{
		Title:         title,
		GeneratedTime: now.Format("15:04:05"),
		GeneratedDate: now.Format("02 Jan 2006"),
		Reports:       reports,
		Timezone:      displayTZ,
	}

	templateContent := r.defaultTemplate.String()
	if template.String() != "" {
		templateContent = template.String()
	}
	return r.generate(ctx, templateContent, data)
}

func (r *report) generate(ctx context.Context, templateContent string, data ReportData) ([]byte, error) {
	tmpl := template.New("report").Funcs(template.FuncMap{
		"formatTime":  func(t float64) string { return r.formatTimeWithTimezone(t, data.Timezone) },
		"formatValue": formatValue,
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
		"iterate":     func(count int) []int { return makeRange(count) },
		"ge":          func(a, b int) bool { return a >= b },
		"lt":          func(a, b int) bool { return a < b },
		"eq":          func(a, b int) bool { return a == b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"getStartRow": getStartRow,
		"getEndRow":   getEndRow,
	})

	tmpl, err := tmpl.Parse(templateContent)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	var htmlBuf bytes.Buffer
	if err := tmpl.Execute(&htmlBuf, data); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	htmlContent := htmlBuf.String()
	pdfBytes, err := r.htmlToPDF(ctx, htmlContent)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return pdfBytes, nil
}

func (r *report) htmlToPDF(ctx context.Context, htmlContent string) ([]byte, error) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	htmlPart, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	if _, err := htmlPart.Write([]byte(htmlContent)); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if err := writer.WriteField("marginTop", "0"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	if err := writer.WriteField("marginBottom", "0"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	if err := writer.WriteField("marginLeft", "0"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	if err := writer.WriteField("marginRight", "0"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if err := writer.WriteField("printBackground", "true"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if err := writer.WriteField("preferCSSPageSize", "true"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	if err := writer.WriteField("emulatedMediaType", "print"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if err := writer.WriteField("waitForSelector", "body"); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if err := writer.Close(); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.converterURL, &requestBody)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	defer resp.Body.Close()

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return pdfBytes, nil
}

func (r *report) formatTimeWithTimezone(t float64, timezone string) string {
	loc, err := resolveTimezone(timezone)
	if err != nil {
		r.runInfo <- pkglog.RunInfo{
			Level:   slog.LevelWarn,
			Message: fmt.Sprintf("failed to resolve timezone '%s', falling back to UTC: %s", timezone, err),
			Details: []slog.Attr{slog.Time("time", time.Now().UTC())},
		}
	}

	var timeVal time.Time
	switch {
	case t > nanosecondThreshold:
		timeVal = time.Unix(0, int64(t)).In(loc)
	default:
		timeVal = time.Unix(int64(t), 0).In(loc)
	}

	return timeVal.Format("2006-01-02 15:04:05")
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

func makeRange(n int) []int {
	result := make([]int, n)
	for i := range result {
		result[i] = i
	}
	return result
}

func getStartRow(pageNum, firstPageRows, continuationPageRows int) int {
	if pageNum == 0 {
		return 0
	}
	return firstPageRows + (pageNum-1)*continuationPageRows
}

func getEndRow(pageNum, firstPageRows, continuationPageRows, totalMessages int) int {
	var end int
	if pageNum == 0 {
		end = firstPageRows
	} else {
		start := firstPageRows + (pageNum-1)*continuationPageRows
		end = start + continuationPageRows
	}

	if end > totalMessages {
		end = totalMessages
	}
	return end
}

func (r *report) generateCSVReport(_ context.Context, title string, reports []Report, timezone string) ([]byte, error) {
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
			timeStr := r.formatTimeWithTimezone(msg.Time, timezone)

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
