package controller

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"my-ple-backend/db"
	"my-ple-backend/email"
	"my-ple-backend/model"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WeeklyLifecycle struct {
	DB          *db.Database
	StorageDir  string
	GeneratePDF func(string, model.ReportingPeriod, []model.WeeklyReport) ([]byte, error)
	SendEmail   func(context.Context, string, string, string, string, string, []byte) error
}

func NewWeeklyLifecycle(database *db.Database) *WeeklyLifecycle {
	dir := os.Getenv("WEEKLY_REPORT_STORAGE_DIR")
	if dir == "" {
		dir = "storage/weekly-reports"
	}
	return &WeeklyLifecycle{DB: database, StorageDir: dir, GeneratePDF: weeklyPDF, SendEmail: email.Send}
}

// A private deterministic storage key; never derived from a request path.
func (s *WeeklyLifecycle) pdfPath(a model.WeeklyArchive) string {
	return filepath.Join(s.StorageDir, a.Period().Start.Format("2006"), a.ID+".pdf")
}

func validPDF(data []byte) bool {
	return bytes.HasPrefix(data, []byte("%PDF-")) && bytes.Contains(data, []byte("%%EOF"))
}

func (s *WeeklyLifecycle) EnsurePDF(ctx context.Context, a *model.WeeklyArchive) ([]byte, error) {
	tx, err := s.DB.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,71260403))`, a.ID); err != nil {
		return nil, err
	}
	// Refresh the digest after waiting for a concurrent download/worker.
	if err = tx.QueryRowContext(ctx, `SELECT pdf_sha256,pdf_filename FROM weekly_report_archives WHERE id=$1`, a.ID).Scan(&a.SHA256, &a.Filename); err != nil {
		return nil, err
	}
	path := s.pdfPath(*a)
	data, readErr := os.ReadFile(path)
	if readErr == nil && validPDF(data) && fmt.Sprintf("%x", sha256.Sum256(data)) == a.SHA256 {
		return data, tx.Commit()
	}
	reports := a.Reports()
	db.SortReports(reports)
	data, err = s.GeneratePDF(a.Division, a.Period(), reports)
	if err != nil {
		return nil, err
	}
	if !validPDF(data) {
		return nil, errors.New("invalid generated PDF")
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".weekly-*.pdf")
	if err != nil {
		return nil, err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return nil, err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	if err = os.Rename(temp, path); err != nil {
		return nil, err
	}
	a.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
	period := a.Period()
	a.Filename = fmt.Sprintf("weekly-report-%s-%s-to-%s.pdf", strings.ToLower(strings.Trim(unsafeFilename.ReplaceAllString(a.Division, "-"), "-")), period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"))
	a.StorageKey = a.Period().Start.Format("2006") + "/" + a.ID + ".pdf"
	if _, err = tx.ExecContext(ctx, `UPDATE weekly_report_archives SET pdf_filename=$2,pdf_sha256=$3,pdf_storage_key=$4,pdf_generated_at=clock_timestamp() WHERE id=$1`, a.ID, a.Filename, a.SHA256, a.StorageKey); err != nil {
		return nil, err
	}
	log.Printf("weekly PDF generated archive=%s", a.ID)
	return data, tx.Commit()
}

func (s *WeeklyLifecycle) finalize(ctx context.Context, a *model.WeeklyArchive, now time.Time) error {
	tx, err := s.DB.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT u.id,lower(trim(u.email)) FROM users u JOIN roles r ON r.id=u.role_id WHERE lower(r.name)='management' AND u.is_active AND trim(u.email)<>'' ORDER BY u.id`)
	if err != nil {
		return err
	}
	recipients := map[string]string{}
	for rows.Next() {
		var id, address string
		if err = rows.Scan(&id, &address); err != nil {
			rows.Close()
			return err
		}
		if email.ValidAddress(address) {
			recipients[address] = id
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for address, id := range recipients {
		if _, err = tx.ExecContext(ctx, `INSERT INTO weekly_report_email_deliveries(id,archive_id,recipient_user_id,recipient_email) VALUES($1,$2,$3,$4) ON CONFLICT(archive_id,recipient_email) DO NOTHING`, db.NewID(), a.ID, id, address); err != nil {
			return err
		}
	}
	if len(recipients) == 0 {
		log.Printf("weekly email: no eligible Management recipients archive=%s", a.ID)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE weekly_report_archives SET status='finalized',finalized_at=COALESCE(finalized_at,$2),last_error='' WHERE id=$1 AND pdf_sha256<>''`, a.ID, now); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	a.Status = "finalized"
	return nil
}

func (s *WeeklyLifecycle) Run(ctx context.Context, now time.Time) error {
	connection, err := s.DB.Conn.Conn(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()
	var locked bool
	if err = connection.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(71260401)`).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return nil
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, e := connection.ExecContext(unlockCtx, `SELECT pg_advisory_unlock(71260401)`); e != nil {
			_ = connection.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	// Only a previous interrupted worker can leave 'sending' while this lock is held.
	if _, err = s.DB.Conn.ExecContext(ctx, `UPDATE weekly_report_email_deliveries SET status='uncertain',last_error='Worker interrupted during send; review before retry' WHERE status='sending'`); err != nil {
		return err
	}
	weeks, err := s.DB.DueWeeks(ctx, now)
	if err != nil {
		return err
	}
	for _, week := range weeks {
		log.Printf("weekly finalization started division=%q week=%s", week.Division, week.Start.Format(time.RFC3339))
		a, e := s.DB.SnapshotWeek(ctx, week)
		if errors.Is(e, sql.ErrNoRows) {
			continue
		}
		if e != nil {
			log.Printf("weekly snapshot failed; reset skipped: %v", e)
			continue
		}
		log.Printf("weekly archive ready/existing id=%s status=%s", a.ID, a.Status)
		if _, e = s.EnsurePDF(ctx, &a); e != nil {
			log.Printf("weekly PDF failed; reset skipped archive=%s: %v", a.ID, e)
			if _, updateErr := s.DB.Conn.ExecContext(ctx, `UPDATE weekly_report_archives SET status=CASE WHEN status='finalized' THEN status ELSE 'failed' END,last_error='PDF generation/storage failed' WHERE id=$1`, a.ID); updateErr != nil {
				return updateErr
			}
			continue
		}
		if a.Status != "finalized" {
			if e = s.finalize(ctx, &a, now); e != nil {
				log.Printf("weekly finalization failed; reset skipped: %v", e)
				continue
			}
		}
		if !now.Before(a.WeekEnd.Add(24*time.Hour)) && a.ResetAt == nil {
			log.Printf("weekly reset started archive=%s", a.ID)
			n, e := s.DB.ResetArchivedWeek(ctx, a, now)
			if e != nil {
				log.Printf("weekly reset skipped archive=%s: %v", a.ID, e)
			} else {
				log.Printf("weekly reset completed archive=%s rows=%d", a.ID, n)
			}
		}
	}
	return s.deliver(ctx, now)
}

func (s *WeeklyLifecycle) deliver(ctx context.Context, now time.Time) error {
	rows, err := s.DB.Conn.QueryContext(ctx, `SELECT id::text FROM weekly_report_email_deliveries WHERE status IN ('pending','failed') AND attempt_count<5 AND (last_attempt_at IS NULL OR last_attempt_at + interval '15 minutes' * power(2,attempt_count-1) <= $1) ORDER BY created_at LIMIT 50`, now)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		var archiveID, address string
		if err = s.DB.Conn.QueryRowContext(ctx, `SELECT archive_id,recipient_email FROM weekly_report_email_deliveries WHERE id=$1`, id).Scan(&archiveID, &address); err != nil {
			return err
		}
		a, err := s.DB.Archive(ctx, archiveID)
		if err != nil {
			return err
		}
		if a.Status != "finalized" {
			continue
		}
		data, err := s.EnsurePDF(ctx, &a)
		if err != nil {
			log.Printf("weekly delivery waiting for PDF archive=%s", a.ID)
			continue
		}
		// Durable claim BEFORE SMTP; a crash must not result in an automatic resend.
		claim, err := s.DB.Conn.ExecContext(ctx, `UPDATE weekly_report_email_deliveries SET status='sending',attempt_count=attempt_count+1,last_attempt_at=$2 WHERE id=$1 AND status IN ('pending','failed') AND attempt_count<5`, id, now)
		if err != nil {
			return err
		}
		if n, err := claim.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			continue
		}
		period := a.Period()
		rangeLabel := period.Start.Format("2 Jan 2006") + " to " + period.End.Format("2 Jan 2006")
		log.Printf("weekly email send/retry delivery=%s", id)
		err = s.SendEmail(ctx, address, "Weekly Report - "+a.Division+" - "+rangeLabel, "Weekly Report "+a.Division+"\r\nReporting Period: "+rangeLabel+"\r\n\r\nThe finalized Weekly Report is attached as a PDF.\r\n", a.Filename, id, data)
		status, lastError := "sent", ""
		if err != nil {
			status = "failed"
			lastError = err.Error() // email.Send returns sanitized phase errors only.
			if errors.Is(err, email.ErrUncertain) {
				status = "uncertain"
				lastError = "SMTP acceptance uncertain; review before retry"
			}
		}
		if _, e := s.DB.Conn.ExecContext(ctx, `UPDATE weekly_report_email_deliveries SET status=$2,last_error=$3,sent_at=CASE WHEN $2='sent' THEN $4::timestamptz ELSE NULL END WHERE id=$1`, id, status, lastError, now); e != nil {
			return e
		}
		log.Printf("weekly email %s delivery=%s", status, id)
	}
	return nil
}
