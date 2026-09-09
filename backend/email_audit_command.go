package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"my-ple-backend/email"
	"os"
	"strings"
	"time"
)

// Read-only local diagnostics. Never print credentials or arbitrary SMTP responses.
func runEmailAudit() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	fmt.Printf("email_enabled=%t smtp_host=%s smtp_port=%s sender=%s username=%s password_configured=%t\n", email.Enabled(), os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"), os.Getenv("SMTP_FROM"), os.Getenv("SMTP_USERNAME"), strings.TrimSpace(os.Getenv("SMTP_PASSWORD")) != "")
	conn, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		return errors.New("database configuration failed")
	}
	defer conn.Close()
	rows, err := conn.QueryContext(ctx, `SELECT COALESCE(email,''),is_active FROM users`)
	if err != nil {
		return errors.New("user audit query failed")
	}
	total, active, nonempty, missing, malformed, spaces, uppercase := 0, 0, 0, 0, 0, 0, 0
	seen := map[string]int{}
	for rows.Next() {
		var a string
		var on bool
		if err = rows.Scan(&a, &on); err != nil {
			rows.Close()
			return err
		}
		total++
		if on {
			active++
		}
		n := email.NormalizeAddress(a)
		if n == "" {
			missing++
		} else {
			nonempty++
			seen[n]++
			if !email.ValidAddress(n) {
				malformed++
			}
		}
		if a != strings.TrimSpace(a) {
			spaces++
		}
		if a != strings.ToLower(a) {
			uppercase++
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return errors.New("user audit read failed")
	}
	duplicates := 0
	for _, n := range seen {
		if n > 1 {
			duplicates++
		}
	}
	fmt.Printf("users total=%d active=%d nonempty=%d missing=%d malformed=%d duplicate_addresses=%d whitespace=%d uppercase=%d\n", total, active, nonempty, missing, malformed, duplicates, spaces, uppercase)
	for _, query := range []string{
		`SELECT 'outbox:'||notification_type||':'||status,count(*) FROM zoom_booking_email_deliveries GROUP BY 1 ORDER BY 1`,
		`SELECT 'weekly_outbox:'||status,count(*) FROM weekly_report_email_deliveries GROUP BY status`,
		`SELECT 'website:'||type,count(*) FROM notifications GROUP BY type`,
		`SELECT 'observed_bad_address_users',count(*) FROM users WHERE lower(btrim(email))='dummyhendrik@gmail.com'`,
		`SELECT 'observed_bad_address_outbox',count(*) FROM zoom_booking_email_deliveries WHERE lower(btrim(recipient_email))='dummyhendrik@gmail.com'`,
		`SELECT 'observed_bad_address_weekly_outbox',count(*) FROM weekly_report_email_deliveries WHERE lower(btrim(recipient_email))='dummyhendrik@gmail.com'`,
		`SELECT 'observed_bad_address_notification_payload',count(*) FROM notifications WHERE data::text LIKE '%dummyhendrik@gmail.com%'`,
	} {
		rows, err = conn.QueryContext(ctx, query)
		if err != nil {
			return errors.New("delivery audit query failed")
		}
		for rows.Next() {
			var label string
			var count int
			if err = rows.Scan(&label, &count); err != nil {
				rows.Close()
				return err
			}
			fmt.Printf("%s=%d\n", label, count)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return errors.New("delivery audit read failed")
		}
	}
	return nil
}
