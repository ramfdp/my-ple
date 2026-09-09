# Zoom Booking additions

Existing `zoom_bookings.description` now travels in the existing
`notification.created` event's `notification.booking.description` JSON field.
The notification bell shows a three-line preview and links to Booking Zoom.
Creator exclusion, 72-hour website retention, booking permissions and resource
mapping (Zoom 1/NP, Zoom 2/MRP, Zoom 3/IB2) are unchanged. Older notifications
without descriptions remain readable.

Zoom Information displays each configured URL with **Salin** for all viewers.
Clipboard success/failure feedback lasts two seconds. The fallback is used only
when Clipboard API is unavailable; denied access shows a friendly error.

## Configuration

Copy the placeholders from `zoom.env.example` into backend `.env` or deployment
secrets. The existing SMTP settings are shared with Weekly Report; do not change
them without considering its sender configuration. No Weekly Report behavior or
recipient selection was changed.

```dotenv
ZOOM_EMAIL_ENABLED=true
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_FROM=aiple5766@gmail.com
SMTP_USERNAME=aiple5766@gmail.com
SMTP_PASSWORD=
```

`SMTP_PASSWORD` must contain a Gmail **App Password**, never the normal account
password. Port 587 uses mandatory STARTTLS; port 465 uses TLS. Certificate checks
remain enabled. Missing/invalid configuration logs an error and stops the Zoom
worker, while bookings continue to save. Restart after changing configuration.
Disabled by default: no Zoom work is queued or sent while the flag is false.
Previously queued work remains available when re-enabled.

The existing `backend/.gitignore` excludes `.env`; example files contain no
secrets. Never put these values in `VITE_*`, frontend config, or tracked files.

## Delivery

The booking transaction queues individual records in
`zoom_booking_email_deliveries`, unique by `(booking_id, recipient_email)`.
Recipients come from active `users.email`, trimmed, lowercased, validated and
deduplicated. A shared address gets the creator subject if any matching active
account is the creator. Eligibility is checked again immediately before delivery.
No recipient list is exposed to other users.

The existing background lifecycle starts a Zoom worker every ten seconds. It
reuses the existing TLS SMTP transport and Weekly Report's durable claim pattern;
the weekly archive-specific table and worker are unchanged. Each message has a
stable delivery Message-ID, plain text and escaped HTML, booking creator/title/
description/date/time (WIB), and the booked resource's configured URL/passcode.
Host keys and unrelated account fields are excluded. The payload snapshots the
creation event; edits/cancellations do not create new email events.

SMTP runs after commit, outside the HTTP request. At most three attempts are made
per recipient, with approximately one then four minutes between retries, plus
polling time. Successful deliveries are not resent. Failed rows remain for
diagnosis. Existing booking overlap protection rejects ordinary duplicate POSTs;
the outbox uniqueness rule prevents duplicate records for the same event/address.

SMTP cannot guarantee exactly-once receipt. If DATA acceptance is uncertain, the
row becomes `uncertain`; a process crash after claiming leaves `sending`. Neither
is automatically retried. Check Gmail Sent Mail/provider logs using the delivery
Message-ID before any operator retry. A stable Message-ID alone does not ensure
provider deduplication.

An enqueue database failure is isolated with a savepoint and logged with booking
ID, so a valid booking still commits. That rare failure requires operator recovery
from the saved booking; there is no automatic catch-up for unqueued work. Mail
delivery records are retained for diagnosis; no unrelated admin UI or retention
job was added. The worker sends up to 50 messages sequentially per poll.

## Verification

```powershell
# From backend; TEST_DATABASE_URL targets a local/disposable PostgreSQL database.
go test ./...
go build ./...

# From frontend
npm run check
npm run build
npm run dev -- --host 127.0.0.1 --port 5190
# Another terminal, with Chrome installed (or set CHROME_PATH to Chromium):
node tests/zoom.browser.test.mjs
```

`TestZoomEmailPostgres` creates/drops its own random schema and migrates twice.
It uses fake SMTP, checks recipients, creator/other subjects, description storage,
retry limits, duplicate requests/events, disabled mode and failure isolation.
The browser test uses a fresh profile and fixture APIs: no real accounts or mail.
MIME/config and real local WebSocket tests require no Gmail credentials.
Live Gmail authentication and inbox delivery require deployment credentials and
are intentionally not exercised by automated tests.

## Files changed for this addition

Frontend: `src/lib/components/ZoomInformationPanel.svelte`,
`src/lib/components/NotificationBell.svelte`, `src/lib/api.ts`,
`tests/zoom.browser.test.mjs`.

Backend: `db/notification.go`, `db/schema.sql`, `db/zoom_email.go`,
`model/notification.go`, `notification/service.go`, `notification/zoom_email.go`,
`email/smtp.go`, `email/zoom.go`, `main.go`, `zoom.env.example`, this document,
`email/zoom_test.go`, `notification/zoom_email_test.go`, `realtime/zoom_test.go`,
`zoom_startup_test.go`.

Pre-existing uncommitted Zoom resource/model/form changes were preserved.
Graphify inspection covered the booking page/form/sidebar/controller/database,
notification payload/feed/hub, User email/active fields, SMTP transport and startup
configuration. A targeted extraction of the changed paths produced 67 nodes and
176 edges in `graphify-out/zoom-impact.json`; shared consumers were rechecked
against the existing graph without rebuilding or crawling unrelated modules.
