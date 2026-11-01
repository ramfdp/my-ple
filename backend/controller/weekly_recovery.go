package controller

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"my-ple-backend/model"
	"my-ple-backend/notification"
	"time"
)

type WeeklyRecoveryResult struct {
	ArchiveID    string `json:"archiveId"`
	Division     string `json:"division"`
	Label        string `json:"label"`
	ReportsReset int64  `json:"reportsReset"`
}

// RunRecovery owns the normal worker lock through preparation and archival.
// Emails are queued by finalize and delivered later by the regular worker.
func (s *WeeklyLifecycle) RunRecovery(ctx context.Context, expectedCount int, fingerprint string, now time.Time) ([]WeeklyRecoveryResult, error) {
	results := []WeeklyRecoveryResult{}
	connection, err := s.DB.Conn.Conn(ctx)
	if err != nil {
		return results, err
	}
	defer connection.Close()
	var locked bool
	if err = connection.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(71260401)`).Scan(&locked); err != nil {
		return results, err
	}
	if !locked {
		return results, errors.New("weekly lifecycle worker is active; stop workers before recovery")
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, e := connection.ExecContext(unlockCtx, `SELECT pg_advisory_unlock(71260401)`); e != nil {
			_ = connection.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	weeks, err := s.DB.ApplyWeeklyRecovery(ctx, expectedCount, fingerprint)
	if err != nil {
		return results, err
	}
	for _, week := range weeks {
		if !week.Start.Before(model.WeeklyFinalizationBoundary(now)) {
			return results, fmt.Errorf("recovery period %s is not due", week.Start.Format(time.RFC3339))
		}
		archive, err := s.DB.SnapshotWeek(ctx, week)
		if err != nil {
			return results, fmt.Errorf("snapshot %s: %w", week.Division, err)
		}
		if err = s.DB.ValidateWeeklyRecovery(ctx); err != nil {
			return results, err
		}
		if _, err = s.EnsurePDF(ctx, &archive); err != nil {
			return results, fmt.Errorf("PDF %s failed; reports retained: %w", archive.ID, err)
		}
		if archive.Status != "finalized" {
			if err = s.finalize(ctx, &archive, now); err != nil {
				return results, fmt.Errorf("finalize %s failed; reports retained: %w", archive.ID, err)
			}
		}
		var reset int64
		if archive.ResetAt == nil {
			reset, err = s.DB.ResetArchivedWeek(ctx, archive, now)
			if err != nil {
				return results, fmt.Errorf("reset %s: %w", archive.ID, err)
			}
			(&notification.Service{DB: s.DB}).Changed(ctx, "reports.changed", 0, archive.ID, 0)
		}
		results = append(results, WeeklyRecoveryResult{ArchiveID: archive.ID, Division: archive.Division, Label: archive.Period().Label, ReportsReset: reset})
	}
	return results, s.DB.VerifyWeeklyRecovery(ctx)
}
