# Hermes dashboard chat (testing)

The dashboard chat searches current MyHIS PostgreSQL data at request time: evaluation findings, accessible workspace tasks (including archived), current weekly reports, and saved weekly archives. Search and role filtering run in the MyHIS backend. Only the top eight bounded excerpts cross the private service link. No embedding index or conversation table is created.

## Enable

1. Complete corporate model approval and configure Hermes with an approved model credential. Keep `AI_PROVIDER_APPROVED=false` until that is done. Without it, chat stays unavailable.
2. Generate a separate random chat token of at least 32 characters. Put identical contents in two private files readable only by the respective backend and chat container users. Do not put the token in Git or browser configuration.
3. In the MyHIS testing stack, add `compose.hermes-chat-testing.yaml` to the existing testing Compose files and set `MY_PLE_HERMES_CHAT_TOKEN_FILE` to the backend token file path.
4. In `my-ple-hermes`, set `HERMES_CHAT_TOKEN_FILE` to its token file path and start the `chat` profile with `compose.myple-testing.yaml`. The service listens only on `myple_read_testing` and `myple_ai`; do not publish port 8787.
5. Confirm the MyHIS dashboard returns answers with source links for authorized users. Check that public gateway requests to `/v1/chat` fail and that a role without evaluation access receives no evaluation source.

The backend URL is fixed to `http://hermes-chat:8787`; assign this alias to the chat service on `myple_read_testing`. A missing service or token yields HTTP 503, with no model call. Disable by removing the MyHIS chat override. Existing evaluations, tasks, reports, and Hermes planner flow remain intact.
