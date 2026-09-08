# Pengaturan Pekerjaan

The application uses SvelteKit for its live frontend and Go/database/sql with
PostgreSQL for persistence. Authentication `roles` are unchanged. Reporting
roles (PE, QA, CIHPK) now live in `report_roles`, under `divisions`.

## Migration and inventory

`Database.Migrate` runs `schema.sql`, `weekly_archive.sql`, and `jobs.sql` in
one transaction under the existing migration advisory lock. Backend startup
and `go run ./cmd/migrate` use this same migration. The migration command takes
`DATABASE_URL` from the environment; it does not load `.env` automatically.

The frozen import in `db/jobs.sql` is extracted from the former catalogs in
`model/report.go` and both Weekly Progress Svelte pages. It preserves the exact
labels and list order, including **Hidran** and **Review FS dan DED**:

| Division / reporting role | Categories | Jobs |
| --- | --- | ---: |
| Project Engineer / PE | MK supervisi; Non-MK supervisi | 3 + 9 |
| Project Engineer / QA | QA1; QA2 | 13 + 13 |
| Project Engineer / CIHPK | JAKG; Manggarai Hydrant; Jember | 1 + 1 + 1 |
| AI Dev | Existing four categories | 4 |
| Keuangan | Existing four categories | 4 |
| Business Development | Existing four categories | 4 |
| HR | Existing four categories | 4 |
| PMO | Existing four categories | 4 |

Total: **61 contextual jobs, 27 categories, 3 reporting roles, 6 divisions**.
The five divisions without separate job lists retain their categories and gain
one initial job with the same label per category. Their roles remain nullable.
Job title in employee profiles is a different concept and is unchanged. No
implemented task/workspace/assignment tables use this catalog. Sequence keys
remain division/category/week; no job ID is added to sequencing because that
would change numbering behavior.

`data_migrations` records the import and backfill separately. Re-running cannot
reset names/status/order or recreate a seed job that an administrator deleted.
Uniqueness uses division, nullable reporting role, nullable category, and
trimmed case-insensitive name. Category context is also checked in PostgreSQL.

## Historical data

`reports.job_id` is additive. Legacy text columns are retained without rewriting
their content during backfill. The migration matches full division/role/category
context and normalized job text, accepting exactly one candidate. Existing QA
aliases are normalized. A missing role never guesses between PE and QA.

Unmatched references are retained and counted in migration logs. Inspect them:

```sql
SELECT * FROM unmatched_job_references ORDER BY source, record_id;
SELECT division_name, role_name, category_name, count(*)
FROM job_context GROUP BY 1,2,3 ORDER BY 1,2,3;
```

Unresolved working reports allow description-only edits that retain their
original context/text, or explicit reassignment to an active job. Old clients
can still submit text; the write boundary resolves it using full context.
Updated clients submit `jobId`; supplied names cannot override the master.

Working reports and newly generated PDFs resolve the current master name,
including inactive jobs, falling back to legacy text. Already finalized archive
JSON and PDFs remain immutable, retaining the names at archival time.
`weekly_archive_jobs` provides foreign-key links without modifying those
snapshots. New snapshots also include `jobId`.

`used_at` permanently records usage, so deleting a working report or the weekly
reset cannot make its job deletable or movable. Database foreign keys and
triggers prevent deletion/context moves of used jobs. Metadata and status can
still change. New work validates the active job and its full context inside
the existing report transaction, locking against concurrent management edits.

## API and UI

Routes follow the application's existing unprefixed convention:

- `GET /jobs/masters`: division, reporting role, category IDs and labels.
- `GET /jobs`: optional `division_id`, `role_id`, `job_category_id`, `active`.
  Omit a filter for all; use `0` for no role/category.
- `GET /jobs/:id`: active or inactive master.
- `POST /jobs`, `PUT /jobs/:id`: camelCase job fields (`divisionId`, `roleId`,
  `jobCategoryId`, `name`, `description`, `isActive`, `sortOrder`).
- `PATCH /jobs/:id/status`: `{ "isActive": false }`.
- `DELETE /jobs/:id`: unused only; a referenced job returns a conflict.

All routes require the existing session. Every authenticated role can create,
edit, activate/deactivate, and safely delete jobs, including through direct HTTP calls. `/settings/jobs`
reuses the sidebar's Settings position. The shared Weekly Progress picker
fetches active jobs and includes the current stored selection during editing.

## Verification

```powershell
# From backend; TEST_DATABASE_URL must point at a development PostgreSQL DB.
# These controller integration tests create and remove their own isolated schemas.
go test ./...
go build ./...

# From frontend
npm run check
npm run build
npm run dev -- --host 127.0.0.1 --port 5190
# Separate terminal; Chrome is supported if Edge closes its debugging connection.
$env:CHROME_PATH = 'C:/Program Files/Google/Chrome/Application/chrome.exe'
node tests/weekly-history.browser.test.mjs
```

The browser runner covers job settings and existing history/PDF flows using
mocked HTTP responses. PostgreSQL/controller tests cover the actual write
boundary, RBAC, constraints, idempotency, backfill, archive links and PDF output.
The inspected development database had zero working reports and zero archives
before migration; historical safety was verified with isolated legacy fixtures.
