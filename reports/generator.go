// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

type ReportData struct {
	Title         string
	GeneratedTime string
	GeneratedDate string
	Reports       []Report
}

func (r *report) generatePDFReport(ctx context.Context, title string, reports []Report, template ReportTemplate) ([]byte, error) {
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

	templateContent := r.defaultTemplate.String()
	if template.String() != "" {
		templateContent = template.String()
	}
	return r.generate(ctx, templateContent, data)
}

func (r *report) generate(ctx context.Context, templateContent string, data ReportData) ([]byte, error) {
	tmpl := template.New("report").Funcs(template.FuncMap{
		"formatTime":  formatTime,
		"formatValue": formatValue,
		"add":         func(a, b int) int { return a + b },
		"sub":         func(a, b int) int { return a - b },
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
	payload := map[string]interface{}{
		"html": htmlContent,
		"options": map[string]interface{}{
			"printBackground": true,
			"margin": map[string]string{
				"top":    "0",
				"bottom": "0",
				"left":   "0",
				"right":  "0",
			},
			"preferCSSPageSize": true,
		},
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.browserURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
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

func (r *report) generateCSVReport(_ context.Context, title string, reports []Report) ([]byte, error) {
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
