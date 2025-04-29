// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/johnfercher/maroto/pkg/color"
	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
)

func generatePDFReport(reports []Report) ([]byte, error) {
	m := pdf.NewMaroto(consts.Portrait, consts.A4)
	m.SetPageMargins(10, 15, 10)

	primaryColor := color.Color{Red: 41, Green: 128, Blue: 185}   // Blue
	secondaryColor := color.Color{Red: 26, Green: 82, Blue: 118}  // Darker blue
	subtleColor := color.Color{Red: 189, Green: 195, Blue: 199}   // Light gray
	tableHeaderBg := color.Color{Red: 236, Green: 240, Blue: 241} // Very light gray
	alternateRow := color.Color{Red: 245, Green: 247, Blue: 249}  // Even lighter gray
	textPrimary := color.Color{Red: 44, Green: 62, Blue: 80}      // Dark blue-gray
	textSecondary := color.Color{Red: 127, Green: 140, Blue: 141} // Medium gray
	white := color.NewWhite()

	m.RegisterHeader(func() {
		m.SetBackgroundColor(primaryColor)
		m.Row(2, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(white)

		m.Row(20, func() {
			m.Col(2, func() {})

			m.Col(8, func() {
				m.Text("Magistrala IoT Report", props.Text{
					Size:  20,
					Style: consts.Bold,
					Color: primaryColor,
					Align: consts.Center,
					Top:   6,
				})
			})

			m.Col(2, func() {
				m.Text(time.Now().Format("02 Jan 2006"), props.Text{
					Size:  10,
					Style: consts.Italic,
					Align: consts.Right,
					Color: textSecondary,
					Top:   8,
				})
			})
		})

		m.SetBackgroundColor(subtleColor)
		m.Row(0.5, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(white)
		m.Row(0.25, func() {})
		m.SetBackgroundColor(subtleColor)
		m.Row(0.25, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(white)

		m.Row(5, func() {})
	})

	m.RegisterFooter(func() {
		currentPage := m.GetCurrentPage()

		m.Row(5, func() {})
		m.SetBackgroundColor(subtleColor)
		m.Row(0.25, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(white)
		m.Row(0.25, func() {})
		m.SetBackgroundColor(subtleColor)
		m.Row(0.5, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(white)

		m.Row(10, func() {
			m.Col(4, func() {
				m.Text("Generated: "+time.Now().Format("15:04:05"), props.Text{
					Size:  8,
					Style: consts.Italic,
					Align: consts.Left,
					Color: textSecondary,
					Top:   3,
				})
			})

			m.Col(4, func() {
				m.Text(fmt.Sprintf("Page %d", currentPage+1), props.Text{
					Size:  9,
					Style: consts.Bold,
					Align: consts.Center,
					Color: textPrimary,
					Top:   3,
				})
			})

			m.Col(4, func() {
				m.Text("Magistrala System", props.Text{
					Size:  8,
					Style: consts.Italic,
					Align: consts.Right,
					Color: textSecondary,
					Top:   3,
				})
			})
		})
	})

	headers := []string{"Time", "Value", "Unit", "Protocol", "Subtopic"}
	widths := []uint{3, 2, 2, 2, 3}

	for i, report := range reports {
		if i > 0 {
			m.AddPage()
		}

		m.Row(0.5, func() {
			m.Col(1, func() {})
		})
		m.SetBackgroundColor(white)

		m.Row(10, func() {
			m.Col(12, func() {
				m.Text("Metrics", props.Text{
					Size:  16,
					Style: consts.Bold,
					Color: secondaryColor,
					Top:   2,
				})
			})
		})

		m.SetBackgroundColor(alternateRow)
		m.Row(0.5, func() { m.Col(12, func() {}) })

		m.Row(8, func() {
			m.Col(2, func() {
				m.Text("Name:	", props.Text{
					Size:  11,
					Style: consts.Bold,
					Align: consts.Left,
					Color: textPrimary,
					Top:   1,
				})
			})

			m.Col(10, func() {
				m.Text(report.Metric.Name, props.Text{
					Size:  11,
					Style: consts.Italic,
					Color: textPrimary,
					Top:   1,
				})
			})
		})

		if report.Metric.ClientID != "" {
			m.Row(8, func() {
				m.Col(2, func() {
					m.Text("Device ID:	", props.Text{
						Size:  11,
						Style: consts.Bold,
						Align: consts.Left,
						Color: textPrimary,
						Top:   1,
					})
				})

				m.Col(10, func() {
					m.Text(report.Metric.ClientID, props.Text{
						Size:  11,
						Style: consts.Italic,
						Color: textPrimary,
						Top:   1,
					})
				})
			})
		}
		m.Row(8, func() {
			m.Col(2, func() {
				m.Text("Channel ID:	", props.Text{
					Size:  11,
					Style: consts.Bold,
					Align: consts.Left,
					Color: textPrimary,
					Top:   1,
				})
			})

			m.Col(10, func() {
				m.Text(report.Metric.ChannelID, props.Text{
					Size:  11,
					Style: consts.Italic,
					Color: textPrimary,
					Top:   1,
				})
			})
		})

		m.SetBackgroundColor(alternateRow)
		m.Row(0.5, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(white)

		m.Row(10, func() {
			m.Col(12, func() {
				m.Text(fmt.Sprintf("Total Records: %d", len(report.Messages)), props.Text{
					Size:  10,
					Style: consts.Italic,
					Align: consts.Right,
					Color: textSecondary,
					Top:   2,
				})
			})
		})

		m.SetBackgroundColor(primaryColor)
		m.Row(1, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(tableHeaderBg)
		m.Row(10, func() {
			for i, header := range headers {
				m.Col(widths[i], func() {
					m.Text(header, props.Text{
						Size:  11,
						Style: consts.Bold,
						Align: consts.Center,
						Top:   2,
						Color: secondaryColor,
					})
				})
			}
		})
		m.SetBackgroundColor(subtleColor)
		m.Row(0.5, func() { m.Col(12, func() {}) })
		m.SetBackgroundColor(white)

		useAlternateColor := false
		for _, msg := range report.Messages {
			if useAlternateColor {
				m.SetBackgroundColor(alternateRow)
			}

			m.Row(9, func() {
				m.Col(widths[0], func() {
					m.Text(formatTime(msg.Time), props.Text{
						Size:  10,
						Align: consts.Center,
						Top:   2,
						Color: textPrimary,
					})
				})

				m.Col(widths[1], func() {
					m.Text(formatValue(msg), props.Text{
						Size:  10,
						Style: consts.Normal,
						Align: consts.Center,
						Top:   2,
						Color: textPrimary,
					})
				})

				m.Col(widths[2], func() {
					m.Text(msg.Unit, props.Text{
						Size:  10,
						Style: consts.Italic,
						Align: consts.Center,
						Top:   2,
						Color: textSecondary,
					})
				})

				m.Col(widths[3], func() {
					m.Text(msg.Protocol, props.Text{
						Size:  10,
						Align: consts.Center,
						Top:   2,
						Color: textPrimary,
					})
				})

				m.Col(widths[4], func() {
					m.Text(msg.Subtopic, props.Text{
						Size:  10,
						Align: consts.Center,
						Top:   2,
						Color: secondaryColor,
					})
				})
			})

			if !useAlternateColor {
				m.Row(0.2, func() {
					m.Col(12, func() {})
				})
			}

			useAlternateColor = !useAlternateColor
			m.SetBackgroundColor(white)
		}
	}

	buf, err := m.Output()
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return buf.Bytes(), nil
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

func generateCSVReport(reports []Report) ([]byte, error) {
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
