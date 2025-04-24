// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

const reportTemplate = `
<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 40px;
        }
        h1 {
            color: #333;
            font-size: 24px;
        }
        h2 {
            color: #555;
            font-size: 18px;
            margin-top: 20px;
        }
        .report-info {
            margin-bottom: 20px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        th {
            background-color: #f2f2f2;
            padding: 8px;
            text-align: left;
            border: 1px solid #ddd;
        }
        td {
            padding: 8px;
            border: 1px solid #ddd;
        }
        tr:nth-child(even) {
            background-color: #f9f9f9;
        }
        .page-break {
            page-break-after: always;
        }
    </style>
</head>
<body>
    {{range $reportIndex, $report := .Reports}}
    <div {{if lt $reportIndex (subtract (len $.Reports) 1)}}class="page-break"{{end}}>
        <h1>Metrics Report</h1>
        <div class="report-info">
            <h2>Report Information</h2>
            <p><strong>Metric:</strong> {{$report.Metric.Name}}</p>
            <p><strong>Device ID:</strong> {{$report.Metric.ClientID}}</p>
            <p><strong>Channel ID:</strong> {{$report.Metric.ChannelID}}</p>
            {{if $report.Metric.Subtopic}}
            <p><strong>Subtopic:</strong> {{$report.Metric.Subtopic}}</p>
            {{end}}
        </div>

        <table>
            <thead>
                <tr>
                    <th>Time</th>
                    <th>Metric Name</th>
                    <th>Value</th>
                    <th>Unit</th>
                    <th>Subtopic</th>
                </tr>
            </thead>
            <tbody>
                {{range $report.SortedMessages}}
                <tr>
                    <td>{{.FormattedTime}}</td>
                    <td>{{$report.Metric.Name}}</td>
                    <td>{{.FormattedValue}}</td>
                    <td>{{.Unit}}</td>
                    <td>{{.Subtopic}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
    {{end}}
</body>
</html>
`

func generatePDFReport(reports []Report) ([]byte, error) {
	type MessageWithFormatting struct {
		senml.Message
		FormattedTime  string
		FormattedValue string
	}

	type ReportWithSortedMessages struct {
		Metric         Metric
		SortedMessages []MessageWithFormatting
	}

	type TemplateData struct {
		Reports []ReportWithSortedMessages
	}

	templateFuncs := template.FuncMap{
		"subtract": func(a, b int) int {
			return a - b
		},
	}

	templateReports := make([]ReportWithSortedMessages, 0, len(reports))

	for _, report := range reports {
		if len(report.Messages) == 0 {
			continue
		}

		sort.Slice(report.Messages, func(i, j int) bool {
			return report.Messages[i].Time < report.Messages[j].Time
		})

		formattedMessages := make([]MessageWithFormatting, len(report.Messages))
		for i, msg := range report.Messages {
			timeStr := time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05")
			if msg.Time > 9999999999 {
				timeStr = time.Unix(0, int64(msg.Time)*1000000).Format("2006-01-02 15:04:05")
			}

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

			formattedMessages[i] = MessageWithFormatting{
				Message:        msg,
				FormattedTime:  timeStr,
				FormattedValue: valueStr,
			}
		}

		templateReports = append(templateReports, ReportWithSortedMessages{
			Metric:         report.Metric,
			SortedMessages: formattedMessages,
		})
	}

	tmpl, err := template.New("report").Funcs(templateFuncs).Parse(reportTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %v", err)
	}

	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, TemplateData{Reports: templateReports})
	if err != nil {
		return nil, fmt.Errorf("error executing template: %v", err)
	}

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, fmt.Errorf("error creating PDF generator: %v", err)
	}

	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	pdfg.MarginTop.Set(15)
	pdfg.MarginBottom.Set(15)
	pdfg.MarginLeft.Set(15)
	pdfg.MarginRight.Set(15)
	pdfg.Title.Set("Metrics Report")

	page := wkhtmltopdf.NewPageReader(&htmlBuffer)
	page.EnableLocalFileAccess.Set(true)
	pdfg.AddPage(page)

	err = pdfg.Create()
	if err != nil {
		return nil, fmt.Errorf("error generating PDF: %v", err)
	}

	return pdfg.Bytes(), nil
}

func generateCSVReport(reports []Report) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{"Time", "Metric Name", "Device ID", "Channel ID", "Subtopic", "Value", "Unit"}

	for i, report := range reports {
		if len(report.Messages) == 0 {
			continue
		}

		if i > 0 {
			if err := writer.Write([]string{""}); err != nil {
				return nil, err
			}
			if err := writer.Write([]string{"=== NEW REPORT ==="}); err != nil {
				return nil, err
			}
			if err := writer.Write([]string{""}); err != nil {
				return nil, err
			}
		}

		if err := writer.Write([]string{"Report Information:"}); err != nil {
			return nil, err
		}
		if err := writer.Write([]string{"Metric Name", report.Metric.Name}); err != nil {
			return nil, err
		}
		if err := writer.Write([]string{"Device ID", report.Metric.ClientID}); err != nil {
			return nil, err
		}
		if err := writer.Write([]string{"Channel ID", report.Metric.ChannelID}); err != nil {
			return nil, err
		}
		if err := writer.Write([]string{"Subtopic", report.Metric.Subtopic}); err != nil {
			return nil, err
		}
		if err := writer.Write([]string{""}); err != nil {
			return nil, err
		}

		if err := writer.Write(headers); err != nil {
			return nil, err
		}

		sort.Slice(report.Messages, func(i, j int) bool {
			return report.Messages[i].Time < report.Messages[j].Time
		})

		for _, msg := range report.Messages {
			timeStr := time.Unix(int64(msg.Time), 0).Format("2006-01-02 15:04:05")
			if msg.Time > 9999999999 {
				timeStr = time.Unix(0, int64(msg.Time)*1000000).Format("2006-01-02 15:04:05")
			}

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
				report.Metric.Name,
				report.Metric.ClientID,
				report.Metric.ChannelID,
				msg.Subtopic,
				valueStr,
				msg.Unit,
			}

			if err := writer.Write(row); err != nil {
				return nil, err
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
