// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pkglog "github.com/absmach/magistrala/pkg/logger"
)

func (r *report) StartScheduler(ctx context.Context) error {
	defer r.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.ticker.Tick():
			due := time.Now().UTC()

			pm := PageMeta{
				Status:          EnabledStatus,
				ScheduledBefore: &due,
			}

			reportConfigs, err := r.repo.ListReportsConfig(ctx, pm)
			if err != nil {
				r.runInfo <- pkglog.RunInfo{
					Level:   slog.LevelError,
					Message: fmt.Sprintf("failed to list reports : %s", err),
					Details: []slog.Attr{slog.Time("due", due)},
				}
				continue
			}

			for _, c := range reportConfigs.ReportConfigs {
				go func(cfg ReportConfig) {
					if _, err := r.repo.UpdateReportDue(ctx, cfg.ID, cfg.Schedule.NextDue()); err != nil {
						r.runInfo <- pkglog.RunInfo{Level: slog.LevelError, Message: fmt.Sprintf("failed to update report: %s", err), Details: []slog.Attr{slog.Time("time", time.Now().UTC())}}
						return
					}
					_, err := r.generateReport(ctx, cfg, EmailReport)
					ret := pkglog.RunInfo{
						Details: []slog.Attr{
							slog.String("domain_id", cfg.DomainID),
							slog.String("report_id", cfg.ID),
							slog.String("report_name", cfg.Name),
							slog.Time("exec_time", time.Now().UTC()),
						},
					}
					if err != nil {
						ret.Level = slog.LevelError
						ret.Message = fmt.Sprintf("failed to generate report: %s", err)
					}
					r.runInfo <- ret
				}(c)
			}
		}
	}
}
