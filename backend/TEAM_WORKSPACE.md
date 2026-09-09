# Team Workspace

Division-private work management integrated with the existing Go/PostgreSQL backend and SvelteKit frontend. `divisions.id` is the workspace ID. There is no second user, division, project, session, or workspace master.

## Data and migrations

`db/workspace.sql` runs after the existing embedded migrations, in the same transaction and advisory migration lock. Use `go run ./cmd/migrate` with `DATABASE_URL`, or the existing application startup migration. Apply migrations before serving traffic, following the application's existing startup convention.

New tables:

| Table | Relationships and purpose |
|---|---|
| `work_items` | Existing `divisions`, optional existing `jobs`, creator `users`; dates, status, priority, URL, version, timestamps and soft archive |
| `work_item_assignees` | Composite task/user primary key; assignment actor and timestamp |
| `work_item_comments` | Task and existing user; chronological comments |
| `work_item_activity` | Task, actor, action, field and JSONB old/new values |
| `workspace_pages` | Division, nullable parent page, creator, visibility, version and bounded formatted document |
| `notification_recipients` | Existing notification/user pair; private task event recipients |

`notifications.workspace_division_id` identifies private workspace events. `workspace_members` and `workspace_access` are SQL views, not duplicate account records. Constraints validate dates, task enums, unique assignments, nonempty titles, page visibility and self-parenting. Indexes cover workspace/status/due date, project, creator, assignments, comments, activity and page hierarchy. A trigger freezes referenced job context through the existing `jobs.used_at` mechanism and rejects jobs from another division.

Task saves and assignments, audit records and notification records commit together. Integer versions plus row locks reject stale changes with 409. Tasks are archived so their audit history survives. Account deletion with workspace history returns 409 and directs administrators to deactivate the account.

## Authorization

All routes are behind existing `RequireAuth` cookie-session middleware. Workspace access is resolved from current PostgreSQL account records, independently of browser role labels or submitted IDs.

- Exact account role names map to matching existing divisions. Existing aliases: `Finance → Keuangan`, `IT → AI Dev`. An unmapped role receives no workspace.
- Admin and Management can access/manage every division. Other accounts can access only their mapped division.
- Members may view division tasks, create their own work, and edit tasks they created or are assigned. Only creators or managers may archive tasks.
- Regular editors may add/remove themselves; they cannot add/remove other assignees. Managers may assign current active division members. Unchanged historical assignments survive employee transfers/deactivation, but no longer contribute to that division's workload or receive its notifications.
- Personal pages are creator-only, including against Admin/Management. Division pages are readable by division members; the creator or manager may edit/delete them.
- Parents must belong to the same division and visibility; Personal parents must have the same creator. Per-workspace transaction locks serialize hierarchy writes, preventing concurrent cycles. Pages with children must have children moved/deleted before deletion or visibility changes.
- Workspace members expose only account ID, name and avatar, never payroll/contact/profile fields.

The existing HR account-management role can change account roles. HR therefore remains a trusted identity administrator under existing application policy; this module does not alter that policy.

## HTTP routes

These follow the existing API convention without adding an `/api` prefix.

| Method | Route | Result |
|---|---|---|
| GET | `/workspaces` | Accessible division directory and counts |
| GET | `/workspaces/:id` | Workspace metadata, permissions and members |
| GET | `/workspaces/:id/members` | Minimal active division member list |
| GET | `/workspaces/:id/jobs` | Existing division jobs |
| GET / POST | `/workspaces/:id/tasks` | Filtered task list / create |
| GET / PATCH / PUT / DELETE | `/work-items/:id` | Detail / versioned edit / archive |
| GET / POST | `/work-items/:id/comments` | Chronological comments / add |
| GET | `/work-items/:id/activity` | Latest 100 task activity records |
| GET | `/workspaces/:id/activity` | Latest 100 division activity records, including archival |
| GET | `/workspaces/:id/overview` | Division-scoped status and overdue counts |
| GET | `/workspaces/:id/workload` | Active member workloads |
| GET / POST | `/workspaces/:id/pages` | Visibility-filtered page metadata / create |
| GET / PATCH / PUT / DELETE | `/workspace-pages/:id` | Authorized page detail / versioned edit / delete leaf |

Task list query parameters: `search`, `assignee` (user UUID), `jobId`, `status` (also `ACTIVE`), `priority`, `start`, `end`, `sort` (`title`, `due`, `priority`, `updated`), `limit` (default 100, maximum 200), `offset`. Date filters select overlapping date ranges. Undated work remains visible in Table/Board. Responses include `tasks` and `total`; views explicitly show pagination.

Task create/update fields: `title`, `description`, `relatedUrl`, nullable `jobId`, `startDate`, `dueDate` (`YYYY-MM-DD` or empty), `status`, `priority`, `assigneeIds`. Omitted assignment on regular creation defaults to self; an explicit empty array means unassigned. Assignment changes use the same task PATCH transaction. Updates and deletes require the current `version`. PATCH supports partial fields. A task's workspace cannot be changed through its payload.

Page bodies contain a maximum of 200 formatted paragraphs and 10,000 characters. Supported kinds: `text`, `heading`, `bold`, `italic`, `bullet`, `check`, `link`. Links allow only HTTP(S), without embedded credentials. Svelte renders escaped text; no stored HTML or universal block engine is used.

## Views and consistency

Dashboard and sidebar link to `/workspace`, which opens the accessible Division Directory. `/workspace/[id]` starts at Overview and provides Table, Board, Timeline, Calendar, Workload and Notes tabs.

All task views use the same task API and shared task detail dialog. Board status changes are pessimistic: the UI retains the old state until the server commits, reports failures and refreshes the shared dataset. Within-column ordering follows the chosen stable server sort; manual ordering is not implemented.

Timeline is a CSS grid with clipped date intervals, horizontal scrolling, Week/Month navigation and Today. Date edits occur in the shared detail dialog. Calendar is a month grid showing every covered date. Both request only tasks intersecting the visible date window. Tasks without dates are not rendered on those grids. Detail requests use a latest-request guard and a task-keyed dialog to prevent mismatched forms after reordered network responses.

## Workload rules

`workspace/workload.go` is the single calculation service. `DONE` and archived work are excluded. An assigned task contributes to each current division assignee.

- Priority weights: Low 1, Medium 2, High 3, Urgent 5.
- Due within seven days: +2; overdue: +3 instead.
- Scheduled duration: +1 per complete 14 days, capped at +4.
- Additional tasks overlapping today: +1 each, capped at +4.
- No active tasks: Available. Otherwise Normal below 10, Busy from 10, Overloaded from 18.

Responses explain active count, high-priority count, approaching deadlines, overdue count, overlap count and score. This is a capacity indicator, not a productivity measurement. Calendar dates and deadline boundaries use Asia/Jakarta. Query scope is the selected division, never the entire company.

## Collaboration

Important field changes, assignment changes, creation, comments and archival are recorded through `recordWorkActivity`. The existing notification tables/feed and `/ws/notifications` hub are reused. Persistent task notifications target assignees and the creator, excluding the actor. Every notification list/count/read/dismiss operation verifies both explicit recipient membership and current workspace access. Zoom notifications retain their existing visibility.

WebSocket messages contain only `workspace.changed` and the division ID, and are sent only to freshly resolved active authorized users, excluding the actor. Clients refetch authorized task data and notification feed. Notes are not remounted by task notifications, preserving drafts. Notification state is cleared across session lifecycle transitions.

## Files

Added backend files: `model/workspace.go`; `db/workspace.sql`, `workspace.go`, `work_items.go`, `workspace_pages.go`, `workspace_workload.go`, `workspace_events.go`, `workspace_integration_test.go`; `controller/workspace_controller.go`; `workspace/workload.go`, `workload_test.go`; `workspace_http_test.go`; this document.

Modified backend files: `main.go` (routing/dashboard), `db/db.go` (migration and account-history deletion protection), `controller/auth_controller.go` (safe conflict response), `db/notification.go`, `model/notification.go`, `realtime/hub.go` (scoped collaboration).

Added frontend files: `/workspace` and `/workspace/[id]` routes; `lib/workspace.ts`; `lib/styles/workspace.css`; `lib/components/workspace/{WorkspaceView,TaskTable,TaskDetail,TaskBoard,TaskTimeline,TaskCalendar,WorkloadView,WorkspaceNotes}.svelte`; `tests/workspace.browser.mjs`.

Modified frontend files: Dashboard page, Sidebar, NotificationBell, `lib/api.ts`, `lib/notifications.ts`. Weekly Progress/report/PDF, booking and user-role mutation behavior are otherwise preserved. No project dependencies were added.

## Verification

Use a disposable PostgreSQL database, never production:

```powershell
$env:WORKSPACE_TEST_DATABASE_URL='postgres://USER@127.0.0.1:PORT/TEST_DB?sslmode=disable'
$env:DATABASE_URL=$env:WORKSPACE_TEST_DATABASE_URL
go -C backend test ./...
go -C backend vet ./...
npm --prefix frontend run check
npm --prefix frontend run build
```

Workspace integration tests create unique schemas and remove them afterward. They verify migration repeatability, division access, Admin/Management policy, role aliases, assignment restrictions, multi-assignee workload/removal, stale writes, referenced jobs, comments, audit, private notes, hierarchy deletion and concurrent cycle prevention, archival and transferred-user notification privacy. HTTP tests exercise real cookie sessions, direct resource requests and real authorized/unauthorized/deactivated WebSocket connections.

`frontend/tests/workspace.browser.mjs` requires a disposable backend on `127.0.0.1:5312` and Vite on `127.0.0.1:5193` (`VITE_API_URL=http://127.0.0.1:5312`, backend `FRONTEND_ORIGIN=http://127.0.0.1:5193`). Seed only that test database with the existing seed command. The script creates test accounts, drives isolated Edge through DevTools, and exercises live PostgreSQL task flows. `WORKSPACE_API`, `WORKSPACE_SITE`, `WORKSPACE_ADMIN`, `WORKSPACE_ADMIN_PASSWORD`, and `CHROME_PATH` can override defaults. Screenshot artifacts go to ignored `graphify-out/`.

The existing frontend has no lint script. Its existing unused `.brand-badge` CSS warning remains. Existing Node unit tests, auth-flow browser tests, Weekly Progress/history, job settings and responsive browser regressions were also exercised.

## Deliberate limits

No attachments: existing uploads are public avatar storage and unsuitable for private division documents. No timeline dragging/resizing, custom Kanban ordering, task checklist, note embeds, arbitrary schemas, formulas, or automatic Weekly Progress integration. Date edits are available through task details. Activity shows the latest 100 entries; the database retains full history. Notes use practical paragraph formatting rather than unrestricted inline rich HTML. Task views are paginated; large datasets require subsequent pages. Workload is a documented heuristic, not a forecast of hours or productivity.
