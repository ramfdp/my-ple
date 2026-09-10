ALTER TABLE reports ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE work_items ADD COLUMN IF NOT EXISTS requires_qa BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE work_items ADD COLUMN IF NOT EXISTS qa_stage TEXT NOT NULL DEFAULT ''
 CHECK(qa_stage IN ('','WAITING','CHECKING','COMPLETE'));
-- Each assignment recipient is an account, even if two accounts normalize to one address.
DROP INDEX IF EXISTS notification_email_address;
CREATE UNIQUE INDEX notification_email_address ON zoom_booking_email_deliveries(notification_id,recipient_email)
 WHERE notification_type<>'work_item_assignment';
ALTER TABLE weekly_report_archives ADD COLUMN IF NOT EXISTS pdf_template_version INTEGER NOT NULL DEFAULT 0;
