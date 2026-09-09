# Email delivery audit — 9 September 2026

## Findings from the local runtime configuration and database

`main` loads `.env` relative to its working directory; existing process environment
values override that file. Run the backend and commands below from `backend/`.
Local configuration has `ZOOM_EMAIL_ENABLED=false`, no `EMAIL_ENABLED`, and a
nonempty SMTP password. Host is `smtp.gmail.com`, port `587`, and both From and
SMTP username are `aiple5766@gmail.com`. No password was printed or copied.

The disabled shared worker explains the four pending local messages: two Zoom,
one leave-created, one leave-approved. There are no sent, failed, or sending rows
in that local snapshot. Both local users are active and have nonempty,
syntactically valid email; missing, malformed, whitespace, uppercase, and
duplicate-address counts are all zero. Syntax does not prove mailbox existence.

`dummyhendrik@gmail.com` was not found in local users, either email outbox,
notification payloads, Graphify-indexed source, or the inspected account seed.
Its exact historical/environment source cannot be established from this database.
Do not replace any user's address with a guess. The reported `550 5.1.1` denotes
an unavailable/nonexistent recipient, not an authentication failure.

## Event coverage

| Existing event | Website recipients | Email policy |
| --- | --- | --- |
| `zoom_booking_created` | Existing active-user feed, excluding creator | Active accounts, including creator confirmation; preserved policy |
| `work_item_assigned` | Newly added assignees | Same newly added active assignees |
| `workspace_task` — update/comment | Persisted assignees/creator recipients, excluding actor | Now uses those exact persisted recipients |
| `leave_request_created` | Active Admin/Management except requester | Same reviewers |
| `leave_request_approved` | Request owner | Active request owner |
| `leave_request_rejected` | Request owner | Active request owner |
| Weekly archive finalized | No website event | Existing Management-only PDF delivery |

No recruitment notification producer was found in the inspected notification
architecture. No additional events or email on arbitrary edits were introduced.

## Changes and operational behavior

- `EMAIL_ENABLED` controls shared notification email. If unset, the existing
  `ZOOM_EMAIL_ENABLED` value remains the compatibility fallback. Weekly delivery
  retains its previous behavior unless `EMAIL_ENABLED` is explicitly set.
- New mail stores the recipient user ID and notification ID. The worker resolves
  the current active `users.email`, trims/lowercases it, validates it, and checks
  event-specific eligibility before SMTP. Legacy Zoom addresses are linked to a
  user by migration only when unambiguous; ambiguous legacy rows are skipped rather than assigned to a guessed user or
  substituted with the sender.
- Existing event keys remain. New notification/user and notification/address
  uniqueness prevents duplicate queue entries. Individual messages expose no
  recipient list. User email records are never rewritten.
- SMTP 5xx responses are terminal `failed_permanent`, including authentication
  535 and recipient 550; sanitized phase/category and code are stored. Repair
  configuration/data and review before manually retrying terminal messages.
  4xx/network errors retain bounded backoff: three shared-worker attempts, five
  weekly-worker attempts. Uncertain DATA acceptance is never blindly replayed.
- Claims older than five minutes become `uncertain` for operator investigation,
  not permanently `sending`. Automatic replay could duplicate accepted mail.
  Graceful HTTP shutdown cancels and waits for the shared worker; final delivery
  status writes get an independent short timeout.
- SMTP runs after business commit. Workspace enqueue failures use a savepoint;
  the task and website event survive. Leave enqueue failures now also preserve
  website notifications; durable leave flags retry email enqueue idempotently.
  Zoom/assignment savepoint failures still require operator enqueue recovery,
  as before; no retrospective bulk mailing was performed.
- Templates retain escaped HTML and plain text. The new workspace messages
  identify the action, actor, task, project, and workspace route.

## Operator commands

```powershell
cd backend
go run . email-audit
```

The audit command is read-only and prints aggregate data, not passwords or raw
SMTP replies. The email-test utility and automated test files were removed at the user's request after verification. There is no public SMTP endpoint.

To enable normal delivery, set `EMAIL_ENABLED=true` in deployment configuration
and restart the backend. This also processes eligible pending mail. The local
configuration was left disabled pending confirmation of a known valid recipient;
no live messages were sent during this audit and no SMTP/inbox success is claimed.
Historical Gmail rejection proves a delivery attempt reached mail infrastructure,
not that current local credentials are valid. Asynchronous bounces after SMTP
acceptance are not automatically consumed by this transport.

## Verification

- Forward embedded migration applied to local PostgreSQL; history preserved.
- `go test ./...` passed with `TEST_DATABASE_URL` and isolated test schemas.
- `go build ./...` passed.
- Fake transport tests cover Zoom, assignment, update/comment, leave create and
  decisions; recipient changes, inactive/missing/malformed recipients, duplicate
  events, retry/backoff, permanent 550, auth 535 classification, uncertain claims,
  enqueue failure isolation, ownership/RBAC, and concurrent leave decisions.
- No live Gmail in automated tests. Frontend contracts were unchanged, so no
  frontend build was required for this audit.

Changed implementation: `main.go`, `email_audit_command.go`,
`email/{delivery,smtp,zoom,leave,work_assignment}.go`,
`notification/zoom_email.go`, `controller/weekly_lifecycle.go`,
`db/{db,notification,zoom_email,work_assignment,work_items,work_notification_email,leave_notifications}.go`,
`db/email_delivery.sql`, `zoom.env.example`, and documentation.
