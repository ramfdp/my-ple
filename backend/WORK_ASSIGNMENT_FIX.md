# Team Workspace creation fix — 9 September 2026

## Duplicate-option diagnosis

Graphify traced `WorkspaceView → TaskDetail → /workspaces/:id/jobs → WorkspaceController → jobs`, then the catalog's `jobs.sql`, `job_context`, and `JOB_SETTINGS.md`.

The original endpoint had **no JOIN** and returned each primary-key ID once. The frontend assigned its response once. The catalog deliberately imports separate contextual jobs: `MPS SGR`, `MPS PIT`, `MPS SGP`, and `SHL JN` each exist under PE / Non-MK supervisi, QA / QA1, and QA / QA2. Its uniqueness constraint includes division, reporting role, category, and normalized name. The workspace endpoint discarded role/category context, producing indistinguishable labels for different IDs.

The source fix selects from the existing `job_context` view (only primary-key joins) and adds a context label for repeated names. It preserves division authorization and one row per authoritative `jobs.id`. There is no `DISTINCT *`, name-based merge, canonical-ID guess, or data deletion. The combobox defensively deduplicates repeated IDs only.

**Consequently, searching `mps sgr` returns the distinct contextual records with distinguishable labels, rather than silently collapsing three IDs into one.** A literal single project option requires an authoritative project-to-job mapping that this schema does not contain. Production data was not queried or modified. To inspect same-name master records safely:

```sql
SELECT division_id, lower(btrim(name)) AS name,
       jsonb_agg(jsonb_build_object('id',id,'role',role_name,'category',category_name)) AS records
FROM job_context
GROUP BY division_id, lower(btrim(name)) HAVING count(*) > 1;
```

## Selector and assignment behavior

`ProjectCombobox.svelte` is scoped to the workspace modal; the shared Weekly Progress `JobPicker` remains unchanged. Search is case-insensitive, trimmed substring matching on job name. Typing keeps results open; selection saves a numeric job ID and closes results. Arrow keys, Enter, Escape, Tab, outside focus/click, empty results, inactive historical selection, and nullable “Tanpa project” are supported. “Semua project” remains only the workspace filter. Results occupy normal modal layout with a bounded scrolling list and wrapping labels, keeping Save reachable on mobile.

`SaveWorkItem` uses the existing previous-assignee comparison, row lock, and version check. Only inserted relationships enter `added`. One `work_item_assigned` event per successful assignment change targets exactly those user IDs through `notification_recipients`. Self-assignment is included. Existing generic notifications remain for non-assignment field edits/comments; they are not assignment events and do not enqueue assignment mail. Removal or unchanged-assignee saves create no assignment event.

Tasks, assignee relationships, audit records, and website events commit together. The existing `workspace.changed` hub message invalidates authorized workspace clients after commit, including the actor for self-assignment. It carries no task content. Clients refetch the existing private feed; recipient membership and current division access still govern visibility, read, dismissal, and the 72-hour retention window. Zoom's creator exclusion remains enforced by `notification_visible`.

## Existing email infrastructure

The existing `zoom_booking_email_deliveries` table and `ZoomEmailWorker` now accept `notification_type='work_item_assignment'`. No second queue, worker, SMTP client, credentials, or startup service was added. Assignment rows record task ID, recipient account ID, assignment version, and the account's validated `users.email`. The existing Zoom booking uniqueness rule remains unchanged.

The bot still uses `ZOOM_EMAIL_ENABLED` and the existing `SMTP_*` configuration for `aiple5766@gmail.com`. Assignment subjects and bodies include title, project, assigner, dates, priority, and description. Both MIME alternatives reuse the existing escaped HTML/plain-text strategy; headers reject injection. There is no fabricated task-detail page URL.

Outbox uniqueness is `(work_item_id, recipient_user_id, assignment_version)`. Creation retries from the modal reuse a per-form UUID `requestId`; a database lock and unique creator/request index return the original task. Updates retain existing version-conflict behavior. A removal followed by a later reassignment is a new event. Legacy callers omitting `requestId` retain ordinary POST semantics.

SMTP runs after commit through the existing worker. Failed enqueue SQL is isolated by a savepoint, allowing the task and website event to commit. As with Zoom, an enqueue failure logs the task ID and requires operator recovery; no new catch-up worker was introduced. Delivery rechecks account email, active division membership, current assignment, and archival state. Existing retry limits/backoff, durable claims, stable Message-ID, sent/uncertain handling are preserved. Actual Gmail inbox delivery was not attempted; tests use injected senders.

## Files changed

| Area | Files |
| --- | --- |
| Workspace API/persistence | `controller/workspace_controller.go`, `db/work_items.go`, `db/work_assignment.go`, `db/workspace.sql` |
| Notification payload/feed | `model/workspace.go`, `model/notification.go`, `db/notification.go`, `realtime/hub.go` |
| Shared email delivery | `notification/zoom_email.go`, `email/zoom.go`, `email/work_assignment.go` |
| Workspace UI | `frontend/src/lib/components/workspace/{ProjectCombobox,TaskDetail,WorkspaceView}.svelte`, `frontend/src/lib/workspace.ts` |
| Existing notification UI | `frontend/src/lib/api.ts`, `frontend/src/lib/notifications.ts`, `frontend/src/lib/components/NotificationBell.svelte` |
| Tests | `notification/work_assignment_test.go`, `email/work_assignment_test.go`, `frontend/tests/workspace.browser.test.mjs` |
| Documentation | `WORK_ASSIGNMENT_FIX.md`, `TEAM_WORKSPACE.md` |

Backend paths in the table are relative to `backend/`; frontend paths are relative to the repository root.

## Graphify and verification

Inspected graph nodes included `WorkspaceView`, `TaskDetail`, `WorkspaceJob`, `WorkspaceController`, `SaveWorkItem`, `WorkItem`, `work_item_assignees`, `Job`, `ListJobs`, `JobMasters`, the contextual catalog/backfill, `Notification`, `ListNotifications`, `Service`, `Hub`, `SendWorkspace`, `ZoomEmailWorker`, `ZoomMessage`, `SendZoom`, `NotificationBell`, and notification-store consumers. Shared-component consumer queries confirmed the Weekly Progress picker scope and existing Zoom email/notification consumers. Graphify's old documentation referenced workspace test files absent from this checkout; new tests were added at the real available seams. No graph rebuild or whole-repository crawl was needed; graph queries used no external LLM extraction.

- Reproduced the original notification failure in PostgreSQL before the fix.
- `go test ./...` passed with `TEST_DATABASE_URL` pointing to a disposable local PostgreSQL cluster; tests used isolated schemas and migrated twice.
- `go build ./...` passed.
- `npm --prefix frontend run check` passed with zero errors/warnings; frontend production build passed.
- PostgreSQL/controller tests cover contextual job identity, division filtering, initial one/multiple assignments, description-only edits, add/remove/unchanged assignees, stale writes, creation retries, self notification visibility/read/dismiss, rejected saves, SMTP failure/backoff/success/no resend, enqueue failure isolation, disabled email, transferred users, outbox entries, and real WebSocket invalidation.
- MIME tests verify both body formats and escaping/header safety.
- Edge/Chromium browser tests passed at 375/768/1440 widths: ID safety deduplication, search/case/trim, selection payload, keyboard, outside close, mobile touch selection, no-match, no-project, scroll/layout/Save access, and assignment feed rendering. The existing Zoom browser suite also passed.
- Client build output was checked for server-only SMTP/email configuration identifiers. Live production data and real Gmail inbox delivery remain unverified.
