# Continuous Weekly Progress entry and job numbering

Successful creation keeps the current input page and all selected context,
clears only the description, shows the existing success toast, and returns
focus to the textarea. Failed requests preserve all values. Pending submissions
disable submission and the textarea to prevent duplicate clicks or lost typing.
Selectors remain editable; Back ends entry using the existing navigation.

## Database authority

`report_sequences` is reused with nullable `job_id` referencing `jobs` and a
unique scope `(primary_division, work_division, week_start, job_id)`. The job
already identifies its division, reporting role and category. The existing
Jakarta `ReportingWeek` helper supplies the week; browser time is not involved.

Create and job reassignment share `allocateReportNumber`, an atomic PostgreSQL
`INSERT ... ON CONFLICT ... DO UPDATE ... RETURNING` in the report transaction.
Failed inserts roll back allocation. Description-only edits retain the number;
moving jobs, including within the same category, allocates in the target job.
Deletions leave gaps; subsequent creates never reuse those numbers.

An additional unique index on `reports(job_id, report_week, number)` protects
relational records. A partial legacy index retains the prior protection for
unresolved rows. The existing database advisory lock for weekly archival and
all weekly reset rules remain unchanged.

The table and PDF now display `report.number` directly, without local counters.
Grouping, merged cells, PDF layout and final archive files are unchanged.

## Upgrade

Backend startup runs `report_numbering.sql` after the job migration, in the
existing migration transaction. Restart the backend with the updated code to
apply the schema and allocator together; old binaries use the old conflict key.
The unique-null handling uses PostgreSQL 15+, supported by this development
environment's PostgreSQL 18.

The migration preserves all report numbers/text and legacy counters. It seeds
each job counter from its maximum retained number, so a legacy job ending at 9
continues at 10. Repeated migration cannot lower counters or restore the old
category-wide constraint. Conflicting legacy job numbers abort the transaction
instead of renumbering historical records. Existing finalized archive JSON and
PDF bytes are never rewritten.

## Regression checks

With `TEST_DATABASE_URL` set, `go test ./...` runs isolated PostgreSQL tests:
24 simultaneous creates by three users across two jobs, repeated in a new week;
independent old-week counters; rollback after a failed insert; same-category
job moves; deletion gaps; direct duplicate rejection; migration idempotency and
exact preservation of legacy report rows. Existing archival lifecycle tests
exercise the unchanged week closure and reset rules.

`node tests/weekly-history.browser.test.mjs` (Vite on port 5190) now also checks
continuous entry, retained IDs, focus, failed-save retry, duplicate-click
protection, editable dependent selectors, and explicit Back navigation.
Table and PDF tests assert the stored numbers, including gaps.
