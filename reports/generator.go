// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"bytes"
	"embed"
	"encoding/csv"
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

type TemplateData struct {
	Title       string
	Reports     []Report
	CurrentDate string
	CurrentTime string
}

//go:embed report_template.html
var reportTemplate embed.FS

func generateHTMLReport(data TemplateData) ([]byte, error) {
	content, err := reportTemplate.ReadFile("report_template.html")
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	tmpl := template.New("report").Funcs(template.FuncMap{
		"formatTime":  formatTime,
		"formatValue": formatValue,
	})

	tmpl, err = tmpl.Parse(string(content))
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return buf.Bytes(), nil
}

func generatePDFReport(title string, reports []Report) ([]byte, error) {
	data := TemplateData{
		Title:       title,
		Reports:     reports,
		CurrentDate: time.Now().UTC().Format("02 Jan 2006"),
		CurrentTime: time.Now().UTC().Format("15:04:05"),
	}

	htmlBytes, err := generateHTMLReport(data)
	if err != nil {
		return nil, err
	}

	// Use your preferred HTML to PDF converter here
	pdfBytes, err := ConvertHTMLToPDF(htmlBytes)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return pdfBytes, nil
}

func ConvertHTMLToPDF(html []byte) ([]byte, error) {
	// Initialize generator
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	// Configure global settings
	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.Grayscale.Set(false)

	// Add page
	page := wkhtmltopdf.NewPageReader(bytes.NewReader(html))
	page.FooterRight.Set("[page]")
	page.FooterFontSize.Set(10)
	pdfg.AddPage(page)

	// Create PDF
	err = pdfg.Create()
	if err != nil {
		return nil, err
	}

	return pdfg.Bytes(), nil
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

func generateCSVReport(title string, reports []Report) ([]byte, error) {
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
