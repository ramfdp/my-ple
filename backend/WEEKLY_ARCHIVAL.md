# Weekly archival lifecycle

This extends the existing Go/PostgreSQL/Svelte Weekly Progress feature. No queue
service or new dependency is required. `WeeklyLifecycle` reuses the existing
PDF renderer and the background cleanup loop in `main.go`.

## Schedule and working data

All reporting boundaries use `Asia/Jakarta` and Go's bundled timezone database.
The database's generated reporting-week column explicitly uses the same zone.
A period is `[Monday 00:00, next Monday 00:00)`; the UI/PDF display Monday–Sunday.

The background loop runs immediately at startup, then on aligned minute boundaries
(including Monday/Tuesday 00:00). Work is based on overdue periods, not an exact
timestamp equality, so restarts and downtime do not skip a week. Execution can
finish after midnight depending on PDF/network work; data freezes at the boundary.
Session/notification cleanup still runs hourly in that same loop.

* Monday 00:00: each nonempty completed division/week gets an immutable snapshot.
  The PDF is generated from that snapshot. PDF success and recipient queuing must
  complete before the archive becomes `finalized`.
* During Monday: the previous period remains visible and read-only. Once a snapshot
  exists, the current view and manual PDF export read it too. Creation is closed
  during Monday; new working entry begins Tuesday, within that Monday–Sunday week.
* Tuesday 00:00: the worker verifies the stored PDF checksum, then deletes only
  source IDs in the matching finalized snapshot. Source rows remain if snapshot
  or PDF creation fails. Failed older periods remain visible read-only until safe
  reset. New weekly counters cannot inherit the prior week's numbers.
* Empty periods produce no archive or email. A late startup processes every
  retained overdue period, including safely performing overdue resets.

The old request-time `CleanupExpiredReports` is now a non-destructive compatibility
method. No controller request performs a weekly reset. Account deletion cannot
remove a user's working reports indirectly; it returns 409 until they are reset.
Account deactivation remains available.

## Schema and snapshot

`db/weekly_archive.sql` runs after the existing embedded schema in `Database.Migrate`.
It is repeatable. The backend fails startup if migration fails, instead of running
partially migrated code. Deploy all backend instances with the new version together:
older versions contain the unsafe request-time deletion logic.

* `weekly_report_archives`: unique `(primary_division, week_start, week_end)`,
  JSONB snapshot, processing/finalization/reset dates, PDF filename/storage key,
  SHA-256 digest, PDF creation date and processing errors.
* `weekly_report_email_deliveries`: unique `(archive_id, recipient_email)`, recipient
  user ID/address, status, attempts, last attempt/error and accepted-send timestamp.
* `reports.report_week`: generated local Monday date, included in number uniqueness.
* `report_sequences.week_start`: period included in the unique job scope. The
  numbering migration initializes job counters from retained report numbers and
  never lowers an existing counter. Table/PDF rendering uses stored numbers;
  see [Weekly Entry](WEEKLY_ENTRY.md). Finalized snapshots and PDF files are unchanged.

The existing application has text divisions, categories and projects rather than
separate foreign-key tables. Each snapshot therefore stores every `WeeklyReport`
field (including source/user ID, report role, category, project, description,
number and completion time), plus employee name and account role ID/name.
Snapshot items are embedded JSONB instead of an unnecessary second item table.
There are no live user/project joins when reading history. A database trigger
rejects changes to snapshot identity/content and rejects archive deletion.

## PDF storage and endpoints

Set `WEEKLY_REPORT_STORAGE_DIR` to a private persistent directory, outside the public
`UPLOAD_DIR`. Default: `storage/weekly-reports`. Files use `<local-week-year>/<archive-id>.pdf`;
PostgreSQL stores the relative storage key. Multi-instance deployments must mount
the **same persistent volume** there. Back up this volume and PostgreSQL together.
Writes use a synced temporary file and rename; no browser participates in archival.

Download filenames include the division and inclusive reporting dates, for example
`weekly-report-project-engineer-2026-08-31-to-2026-09-06.pdf`. The renderer's division
title, role/category merges, project grouping, QA normalization, pagination, fonts
and CONFIDENTIAL watermark remain unchanged.

Authenticated routes, retaining the current all-authenticated-users/all-divisions
report scope:

* `GET /reports/history?division=Project%20Engineer&date=2026-09-03&page=1`:
  20 summaries per page plus `hasMore`; date means any day within the period.
* `GET /reports/history/:id`: immutable items and metadata.
* `GET /reports/history/:id/pdf`: private PDF download, safe filename and MIME type.
  A missing or corrupt file is regenerated only from the snapshot; temporary
  storage failure returns 503. Physical paths/storage keys are not returned.

`/weekly-progress/history` provides filters, pagination, details and downloads.
The current page retains manual export and adds a History link and read-only
period status. The direct input page also checks closure; backend guards remain
authoritative when a tab has stale data.

## Email and concurrency

Configure `SMTP_HOST`, `SMTP_PORT` (default 587), `SMTP_FROM`, and, if the provider
requires authentication, `SMTP_USERNAME`/`SMTP_PASSWORD`. See `weekly.env.example`.
Port 465 uses implicit TLS; other ports require STARTTLS. Certificate verification
is enabled. Connections have timeouts. Credentials never enter frontend variables,
logs, archives, or delivery errors. MIME messages carry the PDF as a base64 attachment.

Recipients come from `users JOIN roles`, with role name `Management`, active status,
and a valid nonempty address. Addresses are trimmed, lowercased and deduplicated.
The queued address is captured at finalization. No valid recipients is logged but
does not prevent the archive/PDF from finalizing.

One PostgreSQL session advisory lock elects the worker across backend instances.
Snapshot transactions share a separate advisory lock with report mutations.
Per-archive locks serialize PDF generation/recovery. Database uniqueness and a
conditional durable delivery claim provide additional duplicate protection.

Known email failures retry at most five attempts, with 15/30/60/120-minute delays.
`sent` rows are never automatically resent. Delivery failures do not roll back
archives or PDFs and do not block an otherwise safe Tuesday reset.

SMTP cannot atomically commit delivery and PostgreSQL status. The durable
`sending` state is written before network transmission. If a process crashes or
SMTP acceptance is uncertain, the row becomes `uncertain`, and automatic resend
is disabled. This chooses duplicate prevention over silently retrying an email
that might already have arrived. Provider-side idempotency would be needed for
both guaranteed eventual delivery and exactly-once acceptance.

Operators can inspect `status`, `attempt_count`, `last_attempt_at`, `last_error`
and `sent_at` in `weekly_report_email_deliveries`. After resolving configuration
or provider trouble, reset a **specific failed delivery** to `pending` with
`attempt_count=0,last_attempt_at=NULL` to authorize a new bounded retry cycle.
For `uncertain`, first confirm non-delivery against provider logs using the
stable Message-ID `<delivery-id@weekly-report.local>`; do not blindly retry it.

## Verification

Implementation files:

* Backend lifecycle/API: `main.go`, `controller/weekly_lifecycle.go`,
  `controller/weekly_history.go`, `controller/weekly_controller.go`,
  `controller/weekly_pdf.go`, and the report-preservation guard in
  `controller/auth_controller.go`.
* Persistence/models/email: `db/db.go`, `db/weekly_archive.go`,
  `db/weekly_archive.sql`, `model/report.go`, `model/weekly_archive.go`,
  and `email/smtp.go`.
* Frontend: `src/lib/api.ts`, `src/routes/weekly-progress/+page.svelte`,
  `src/routes/weekly-progress/new/+page.svelte`, and
  `src/routes/weekly-progress/history/+page.svelte`.
* Verification: `main_test.go`, `controller/weekly_lifecycle_test.go`,
  `db/project_engineer_integration_test.go`, `email/smtp_test.go`, and
  `frontend/tests/weekly-history.browser.test.mjs`.

`go test ./...` runs the existing PDF regression tests and MIME validation tests.
Set `TEST_DATABASE_URL` to a disposable/local PostgreSQL connection to include
`TestWeekly*Postgres`. Those tests create and drop unique temporary schemas and
private PDF directories, leaving application tables untouched. SMTP is replaced
with a capturing/failing sender; these tests do not email real Management users.

Coverage includes Sunday/Monday boundaries, Monday read-only source retention,
Tuesday reset and numbering, snapshot immutability, PDF recovery, authenticated
historical downloads, date/division validation, repeated/concurrent/restarted
workers, PDF and email failure, bounded retries, uncertain sends, no recipients,
and missed Tuesday catch-up. Migrations run twice in each isolated test schema.

Frontend checks: `npm run check`, `npm run build`, and existing Node tests.
The existing `auth-flow.test.mjs` additionally requires Playwright (not installed
in this checkout); it is separate from the dependency-free smoke test below.
For real-browser smoke tests, run Vite on `127.0.0.1:5190`, then
`node tests/weekly-history.browser.test.mjs`. The test uses installed Edge by
default (`CHROME_PATH` can point to another Chromium browser) and intercepts API
responses with fixed fixtures. It verifies filters, pagination, snapshot detail,
PDF download and Monday's read-only controls without external email or live data.
