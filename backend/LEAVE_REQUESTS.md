# Izin & Cuti

Employees use `/leave`; Admin and Management use `/leave/approvals`.
Both pages support `?request=<uuid>` notification links. Lists have status/type
filters, newest-first ordering, and 50-row pages. Dates are PostgreSQL DATE values
and remain `YYYY-MM-DD` in transport; the UI formats business dates without local
UTC shifts and displays timestamps in Asia/Jakarta.

## API

All routes use the existing session cookie and protected router.

| Method | Route | Access |
| --- | --- | --- |
| GET / POST | `/leave-requests` | Own history / create own request |
| GET | `/leave-requests/{id}` | Owner only |
| GET | `/leave-requests/approvals` | Admin / Management |
| GET | `/leave-requests/approvals/{id}` | Admin / Management |
| POST | `/leave-requests/approvals/{id}/decision` | Admin / Management, never the owner |

List parameters: `status`, `type`, `offset`.
Creation fields: `requestId` (UUID retry key), `requestType` (`CUTI`/`IZIN`),
`startDate`, `endDate`, `reason`, optional `notes`.
Decision fields: `status` (`APPROVED`/`REJECTED`), optional `reviewerNote`.
Unknown JSON fields are rejected. Identity is never supplied by the body.
Invalid input returns 400, forbidden review 403, inaccessible detail 404,
and an already-decided request 409. The UI reloads the detail after a conflict.

## Persistence and delivery

`db/leave.sql` extends the existing embedded migration. `leave_requests` references
existing users and records reviewer identity/time/note. There is no separate
employee table, audit system, history deletion, or status-correction workflow.
A unique `(user_id, request_id)` prevents duplicate submissions. Reusing a key with
different content returns 409. A guarded atomic `UPDATE ... WHERE status='PENDING'`
allows only one decision; backend authorization and a database constraint prohibit
self-review.

Business writes commit before `NotifyLeave`. Website notifications use the existing
`notifications`, `notification_recipients`, visibility function, and WebSocket hub.
Events are `leave_request_created`, `leave_request_approved`, and
`leave_request_rejected`. Private `leave.changed` invalidations carry no request
content; clients fetch their authorized feed. Current roles are checked on reads.

New-request recipients are active Admin/Management accounts except the requester.
Decision recipients are the active owner only. Email addresses are validated,
normalized, deduplicated, and queued one address per message in the existing
`zoom_booking_email_deliveries` table. `(leave_request_id, notification_type,
recipient_email)` is unique. Delivery rechecks recipient eligibility.

The existing `ZoomEmailWorker`, Gmail transport, MIME helper, retry limits, and
uncertain-delivery handling are reused. Existing `SMTP_*`, `FRONTEND_ORIGIN`, and
`ZOOM_EMAIL_ENABLED` configuration remains authoritative. Leave mail is queued
while sending is disabled and delivered when the existing flag/configuration
allows it. Do not start another worker.

The worker also retries incomplete leave-notification transactions using persistent
flags, even when mail sending is disabled. Flags prevent old events from being
recreated after the existing 72-hour website-notification cleanup. Enqueue/SMTP
failures never undo requests or decisions. Recovered website notifications appear
on the next feed refresh/reconnect; recovery itself does not broadcast a live event.

## Global sound

`notifications.ts` plays the local 8 KB, 180 ms `notification.wav` through one
`notification-sound.ts` instance. Volume is 25%. Pointer/keyboard interaction
unlocks playback; blocked playback is caught and never interrupts notifications.
Initial history and reconnect fetches seed a per-user in-memory seen-ID set silently.
New direct events and authorized live invalidation fetches sound once; an 800 ms
cooldown coalesces bursts. Opening/dismissing notifications does not trigger sound.
The layout keeps the connection alive across authenticated route changes.
No preference module or optional mute toggle was added.

## Verification

- Run migration with the configured `DATABASE_URL`: `go run ./cmd/migrate`.
- With `TEST_DATABASE_URL` set, `go test ./...` creates and removes isolated test
  schemas. Leave coverage uses real cookie sessions, HTTP, PostgreSQL, and the
  existing WebSocket hub. SMTP senders are injected; tests send no real mail.
- `go build ./...`; frontend `npm run check` and `npm run build`.
- Run Vite on `127.0.0.1:5190`, then `node tests/leave.browser.test.mjs` in frontend.
  The isolated Chrome test uses fixture APIs and real media playback with simulated
  blocked playback, covers approval flows and sound, and checks 320, 360, 390, 414,
  768, 1024, 1366, and 1920 px. Screenshots go to `.tmp-gocache`.
- Existing `workspace.browser.test.mjs` and `zoom.browser.test.mjs` cover shared UI
  regressions. `email/leave_test.go` checks MIME privacy, escaping, and date labels.

Graphify discovery used authentication/session, users/roles, Sidebar/layout,
NotificationBell/startNotifications, notification persistence/visibility, Hub,
work-assignment outbox integration, ZoomEmailWorker, SMTP/MIME/configuration, and
Workspace/Zoom forms/styles. No repository-wide scan or graph rebuild was needed.

## Changed files

New backend files: `db/leave.sql`, `db/leave.go`, `db/leave_notifications.go`,
`model/leave.go`, `controller/leave_controller.go`, `email/leave.go`,
`email/leave_test.go`, `notification/leave_test.go`, and this document.
Existing backend extensions: `db/db.go`, `db/notification.go`, `main.go`,
`model/notification.go`, `notification/zoom_email.go`, `realtime/hub.go`.

New frontend files: `src/lib/leave.ts`, `src/lib/components/LeaveView.svelte`,
`src/routes/leave/+page.svelte`, `src/routes/leave/approvals/+page.svelte`,
`src/lib/notification-sound.ts`, `static/notification.wav`,
`tests/leave.browser.test.mjs`.
Existing frontend extensions: `src/lib/api.ts`, `src/lib/notifications.ts`,
`src/lib/components/NotificationBell.svelte`, `src/lib/components/Sidebar.svelte`,
`src/routes/+layout.svelte`.
