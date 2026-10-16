# AI Daily Job Planner: Phase 5

The testing backend accepts an existing Worker `daily-v3` draft through `POST /internal/ai/v1/drafts/import`. This route is disabled by default and blocked at both public Nginx gateways, including `/api/internal/ai/`. Only the Worker uses the private `myple_read_testing` network. Its import token differs from the read-only snapshot token; neither token reaches the browser. Hermes stays on `myple_ai` and cannot reach PostgreSQL. The Worker keeps its existing named state volume.

## Storage and authorization

Startup migration under existing PostgreSQL advisory locks adds `ai_plan_drafts`, `ai_plan_recommendations`, and append-only application review events in `ai_plan_reviews`. Draft ID is the Worker SHA-256 identity. Draft JSON and original source evidence remain intact. A unique draft ID makes a repeated identical import idempotent; a conflicting payload returns 409. Indexes cover workspace/date history and recommendation review history. No AI route writes `work_items`.

Import requires `AI_IMPORT_ENABLED=true`, a 32-byte hex token file at `AI_IMPORT_TOKEN_FILE`, and `AI_IMPORT_ALLOWED_WORKSPACE_IDS`. The token must differ from `AI_SERVICE_TOKEN_FILE`. Backend validates schema `1.0`, prompt `daily-v3`, dates, IDs, selection counts, source references, deterministic identity, and current testing snapshot hash. Changed source returns 409. A later source change marks the stored plan stale; original evidence and review history remain visible. Stale drafts cannot receive another review through the UI.

Human routes require the existing `my_ple_session` cookie, a current database `workspace_access.can_manage` grant, and the Management or Admin role. Machine tokens grant no human access. Routes:

- `GET /ai-plans/workspaces/{id}`: latest published plan or `plan: null`.
- `GET /ai-plans/workspaces/{id}/{draftId}`: stored plan, evidence, reviews, stale flag.
- `GET /ai-plans/workspaces/{id}/history?offset=0`: 20 drafts per page.
- `PATCH /ai-plans/workspaces/{id}/{draftId}/recommendations/{recommendationId}/review`: `{ "decision": "SUITABLE", "note": "...", "version": 0 }`. Version mismatch returns 409. Decisions: `PENDING`, `SUITABLE`, `NEEDS_REVISION`, `REJECTED`. Every saved change appends an audit event. No decision approves or assigns a task.

In Workspace, Management/Admin with `can_manage` use **AI Daily Job Planner**. Select workspace, date, or history entry; inspect source evidence and the original task link. The task link uses `/workspace/{id}?task={taskId}`. Mock drafts carry a visible simulation label. Opening the page never generates a draft.

## Controlled testing deployment

Back up the isolated testing PostgreSQL volume before startup migration. Provision two different 64-character hex tokens in private files with mode `0600`. For example, `openssl rand -hex 32 | install -m 600 /dev/stdin /private/path/ai-import-token`; do this on a trusted host and do not paste the output or token into shell history. Mount the same import credential read-only into testing backend and Worker using `AI_BACKEND_IMPORT_TOKEN_FILE` and `AI_WORKER_IMPORT_TOKEN_FILE`. Set `AI_IMPORT_ALLOWED_WORKSPACE_IDS` explicitly and keep it aligned with Worker grants. Do not change `.env` files containing existing secrets. Verify the token files and permissions without printing contents.

Validate `docker compose -f compose.yaml -f compose.testing.yaml -f compose.ai-testing.yaml config --quiet` before testing. Start the isolated testing stack using its existing deployment procedure, then run backend migration through normal startup. Verify the public gateway returns 404 for `/internal/ai/v1/drafts/import` and `/api/internal/ai/v1/drafts/import`. Generate a mock draft separately, then publish with the Worker command documented in the Hermes repository. Confirm repeated publish creates no duplicate, the browser shows evidence and history, and Job Tracker records remain unchanged. Do not deploy this procedure to production.

## Tests, rollback, limits

Run `go test ./...`, `go test -race ./...`, and `go vet ./...` in `backend`; `npm run check` and `npm run build` in `frontend`. PostgreSQL integration tests require a separate local test database. Roll back application containers to prior images and set `AI_IMPORT_ENABLED=false`. Keep AI tables for audit and preserve all volumes; restore the testing database backup only if migration failure requires it. Do not drop existing tables or clear draft state.

Phase 6 must define final approval, source-version revalidation at synchronization, assignment authorization, idempotent Job Tracker writes, and notification semantics. Phase 5 has no final approval, task creation, scheduler, document RAG, corporate cloud-model authorization, or production deployment.
