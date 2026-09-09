ALTER TABLE zoom_booking_email_deliveries ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ;
ALTER TABLE zoom_booking_email_deliveries ADD COLUMN IF NOT EXISTS notification_id UUID;
ALTER TABLE zoom_booking_email_deliveries DROP CONSTRAINT IF EXISTS zoom_booking_email_deliveries_status_check;
ALTER TABLE zoom_booking_email_deliveries ADD CONSTRAINT zoom_booking_email_deliveries_status_check
 CHECK(status IN ('pending','sending','sent','failed','failed_permanent','uncertain','skipped'));
CREATE UNIQUE INDEX IF NOT EXISTS notification_email_event ON zoom_booking_email_deliveries(notification_id,recipient_user_id);
CREATE UNIQUE INDEX IF NOT EXISTS notification_email_address ON zoom_booking_email_deliveries(notification_id,recipient_email);
-- Attach legacy Zoom deliveries only where recipient identity is unambiguous.
UPDATE zoom_booking_email_deliveries d SET recipient_user_id=u.id
FROM users u WHERE d.notification_type='zoom_booking' AND d.recipient_user_id IS NULL
AND lower(btrim(u.email))=d.recipient_email
AND (SELECT count(*) FROM users x WHERE lower(btrim(x.email))=d.recipient_email)=1;

ALTER TABLE weekly_report_email_deliveries DROP CONSTRAINT IF EXISTS weekly_report_email_deliveries_status_check;
ALTER TABLE weekly_report_email_deliveries ADD CONSTRAINT weekly_report_email_deliveries_status_check
 CHECK(status IN ('pending','sending','sent','failed','failed_permanent','uncertain','skipped'));
