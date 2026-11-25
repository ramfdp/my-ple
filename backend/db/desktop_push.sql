CREATE TABLE IF NOT EXISTS desktop_subscriptions (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL UNIQUE,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    last_reminder DATE
);

CREATE TABLE IF NOT EXISTS desktop_deliveries (
    subscription_id UUID NOT NULL REFERENCES desktop_subscriptions(id) ON DELETE CASCADE,
    event_key TEXT NOT NULL,
    notification_id UUID REFERENCES notifications(id) ON DELETE CASCADE,
    reminder_date DATE,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(subscription_id, event_key),
    CHECK (num_nonnulls(notification_id, reminder_date) = 1)
);
CREATE INDEX IF NOT EXISTS desktop_deliveries_due ON desktop_deliveries(next_attempt);

CREATE OR REPLACE FUNCTION enqueue_desktop_notification() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO desktop_deliveries(subscription_id,event_key,notification_id)
    SELECT d.id,NEW.id::text,NEW.id FROM desktop_subscriptions d
    JOIN sessions s ON s.id=d.session_id
    WHERE s.revoked_at IS NULL AND s.expires_at>now() AND notification_visible(NEW.id,s.user_id);
    RETURN NULL;
END $$;
-- Recipients may be inserted after the notification in the same transaction.
DROP TRIGGER IF EXISTS desktop_notification_created ON notifications;
CREATE CONSTRAINT TRIGGER desktop_notification_created
AFTER INSERT ON notifications DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enqueue_desktop_notification();

