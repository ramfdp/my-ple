# Graph Report - backend  (2026-10-01)

## Corpus Check
- 83 files · ~43,709 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 5 file(s) not represented in the graph (top: (none) 3, .ttf 2)

## Summary
- 517 nodes · 1680 edges · 19 communities (16 shown, 3 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 129 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `9d567e2a`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- go_pkg_my_ple_backend_model
- net/http.ResponseWriter
- time.Time
- NewID
- smtp.go
- Hub
- db.go
- context.Context
- Database
- User
- Attendance
- document_checklist.go
- .CreateLeave
- Job
- lockBilling
- resolveReportJob
- .checkAndSendReminders
- mutationResponse
- my-ple-backend

## God Nodes (most connected - your core abstractions)
1. `Database` - 53 edges
2. `WriteJSON()` - 45 edges
3. `GetClaims()` - 26 edges
4. `NewID()` - 21 edges
5. `User` - 20 edges
6. `Hub` - 20 edges
7. `WeeklyReport` - 19 edges
8. `AuthController` - 16 edges
9. `DecodeStrictJSON()` - 16 edges
10. `ValidAddress()` - 16 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `runEmailAudit()`  [INFERRED]
  main.go → email_audit_command.go
- `main()` --calls--> `loadDotEnv()`  [INFERRED]
  main.go → env.go
- `changeEvents()` --calls--> `GetClaims()`  [EXTRACTED]
  change_events.go → middleware/auth.go
- `newHandler()` --calls--> `changeEvents()`  [INFERRED]
  main.go → change_events.go
- `main()` --calls--> `NewDatabase()`  [EXTRACTED]
  cmd/migrate/main.go → db/db.go

## Import Cycles
- None detected.

## Communities (19 total, 3 thin omitted)

### Community 0 - "go_pkg_my_ple_backend_model"
Cohesion: 0.09
Nodes (42): go_pkg_archive_zip, go_pkg_bytes, go_pkg_context, go_pkg_database_sql, go_pkg_database_sql_driver, go_pkg_encoding_json, go_pkg_encoding_xml, go_pkg_errors (+34 more)

### Community 1 - "net/http.ResponseWriter"
Cohesion: 0.11
Nodes (27): attendanceError(), AuthController, ownsAvatar(), FeedbackHandler(), DecodeJSON(), DecodeStrictJSON(), WriteJSON(), leaveError() (+19 more)

### Community 2 - "time.Time"
Cohesion: 0.07
Nodes (40): exportCategory, exportJob, exportProgress, exportRole, weeklyDOCX(), validPDF(), documentReports(), exportGroups() (+32 more)

### Community 3 - "NewID"
Cohesion: 0.09
Nodes (32): main(), NewID(), workAssignmentNotification(), Database, recordWorkActivity(), ValidateWorkItem(), validWorkURL(), workPermissions() (+24 more)

### Community 4 - "smtp.go"
Cohesion: 0.08
Nodes (41): scanNotification(), AttendanceReminderMessage(), SendAttendanceReminder(), runEmailAudit(), Enabled(), Failure(), MaskAddress(), NormalizeAddress() (+33 more)

### Community 5 - "Hub"
Cohesion: 0.08
Nodes (30): changeEvents(), NewAuthController(), LeaveController, NewWeeklyController(), loadDotEnv(), go_pkg_crypto_rand, go_pkg_crypto_sha256, go_pkg_encoding_base64 (+22 more)

### Community 6 - "db.go"
Cohesion: 0.17
Nodes (15): cleanZoomBooking(), scanZoomBooking(), scanZoomResource(), zoomBookingSelect(), zoomConflict(), queueZoomEmail(), go_pkg_embed, go_pkg_sort (+7 more)

### Community 7 - "context.Context"
Cohesion: 0.13
Nodes (8): Database, Database, Database, UserMissingAttendance, Database, context.Context, JobPIC, Service

### Community 8 - "Database"
Cohesion: 0.13
Nodes (12): main(), NewAttendanceController(), AttendanceController, BillingDocumentsController, DocumentChecklistController, JobController, NewWeeklyLifecycle(), Database (+4 more)

### Community 9 - "User"
Cohesion: 0.16
Nodes (6): scanUser(), Credentials, Role, UpdateProfileInput, UpdateUserInput, User

### Community 10 - "Attendance"
Cohesion: 0.26
Nodes (9): attendanceCoordinate(), AttendanceOffice, Database, scanAttendance(), Attendance, AttendanceHistory, AttendanceInput, AttendanceOverview (+1 more)

### Community 11 - "document_checklist.go"
Cohesion: 0.20
Nodes (8): checklistPDF(), Database, ChecklistItem, ChecklistSigner, CanUseDocumentChecklist(), ChecklistConfigInput, ChecklistItemPatch, DocumentChecklist

### Community 12 - ".CreateLeave"
Cohesion: 0.28
Nodes (7): Database, Database, scanLeave(), ValidateLeave(), LeaveDecision, LeaveInput, LeaveRequest

### Community 13 - "Job"
Cohesion: 0.29
Nodes (6): Database, normalizeJob(), scanJob(), Job, JobMaster, CanonicalWeeklyRoleCategory()

### Community 14 - "lockBilling"
Cohesion: 0.33
Nodes (3): Database, lockBilling(), BillingPatch

### Community 15 - "resolveReportJob"
Cohesion: 0.60
Nodes (3): allocateReportNumber(), resolveReportJob(), WeeklyReportInput

## Knowledge Gaps
- **7 isolated node(s):** `WeeklyController`, `Database`, `my-ple-backend`, `claimsKey`, `NotificationActor` (+2 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 48 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Database` connect `Database` to `net/http.ResponseWriter`, `time.Time`, `smtp.go`, `Hub`, `db.go`, `User`, `resolveReportJob`, `.checkAndSendReminders`?**
  _High betweenness centrality (0.088) - this node is a cross-community bridge._
- **Why does `GetClaims()` connect `net/http.ResponseWriter` to `Hub`, `context.Context`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Why does `Hub` connect `Hub` to `go_pkg_my_ple_backend_model`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Are the 39 inferred relationships involving `WriteJSON()` (e.g. with `attendanceError()` and `.ServeHTTP()`) actually correct?**
  _`WriteJSON()` has 39 INFERRED edges - model-reasoned connections that need verification._
- **What connects `WeeklyController`, `Database`, `my-ple-backend` to the rest of the system?**
  _7 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `go_pkg_my_ple_backend_model` be split into smaller, more focused modules?**
  _Cohesion score 0.08614232209737828 - nodes in this community are weakly interconnected._
- **Should `net/http.ResponseWriter` be split into smaller, more focused modules?**
  _Cohesion score 0.10900045228403438 - nodes in this community are weakly interconnected._