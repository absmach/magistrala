// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"
	"time"
)

func (re *report) StartScheduler(ctx context.Context) error {
	defer re.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-re.ticker.Tick():
			due := time.Now().UTC()

			pm := PageMeta{
				Status:          EnabledStatus,
				ScheduledBefore: &due,
			}

			reportConfigs, err := re.repo.ListReportsConfig(ctx, pm)
			if err != nil {
				return err
			}

			for _, cfg := range reportConfigs.ReportConfigs {
				go func(config ReportConfig) {
					err = re.processReportConfig(ctx, config)
				}(cfg)
				if _, err := re.repo.UpdateReportDue(ctx, cfg.ID, cfg.Schedule.NextDue()); err != nil {
					return err
				}
			}
		}
	}
}

func (re *report) processReportConfig(ctx context.Context, cfg ReportConfig) error {
	if _, err := re.generateReport(ctx, cfg, EmailReport); err != nil {
		return err
	}
	return nil
}
