CREATE TABLE IF NOT EXISTS leave_requests (
 id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id), request_id UUID NOT NULL,
 request_type TEXT NOT NULL CHECK(request_type IN ('CUTI','IZIN')),
 start_date DATE NOT NULL, end_date DATE NOT NULL CHECK(end_date>=start_date),
 reason VARCHAR(4000) NOT NULL CHECK(btrim(reason)<>''), notes VARCHAR(4000) NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'PENDING' CHECK(status IN ('PENDING','APPROVED','REJECTED')),
 reviewed_by UUID REFERENCES users(id), reviewed_at TIMESTAMPTZ, reviewer_note VARCHAR(4000) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 created_notified BOOLEAN NOT NULL DEFAULT false, reviewed_notified BOOLEAN NOT NULL DEFAULT false,
 UNIQUE(user_id,request_id), CHECK(reviewed_by<>user_id),
 CHECK((status='PENDING' AND reviewed_by IS NULL AND reviewed_at IS NULL) OR
       (status<>'PENDING' AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS leave_owner_created ON leave_requests(user_id,created_at DESC);
CREATE INDEX IF NOT EXISTS leave_status_created ON leave_requests(status,created_at DESC);
CREATE INDEX IF NOT EXISTS leave_notification_pending ON leave_requests(created_at) WHERE NOT created_notified OR (status<>'PENDING' AND NOT reviewed_notified);
ALTER TABLE zoom_booking_email_deliveries ADD COLUMN IF NOT EXISTS leave_request_id UUID REFERENCES leave_requests(id);
CREATE UNIQUE INDEX IF NOT EXISTS leave_email_event ON zoom_booking_email_deliveries(leave_request_id,notification_type,recipient_email);
CREATE UNIQUE INDEX IF NOT EXISTS leave_notification_event ON notifications(entity_id,type) WHERE entity_type='leave_request';

-- Keep existing visibility rules unchanged; leave events require explicit recipients and current authority.
CREATE OR REPLACE FUNCTION notification_visible(notification UUID, viewer UUID) RETURNS BOOLEAN
LANGUAGE SQL STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM notifications n JOIN users u ON u.id=viewer JOIN roles r ON r.id=u.role_id
 WHERE n.id=notification AND u.is_active AND (n.actor_user_id<>viewer OR n.type='work_item_assigned') AND (
 (n.type='zoom_booking_created' AND n.workspace_division_id IS NULL) OR
 (n.workspace_division_id IS NOT NULL AND EXISTS(SELECT 1 FROM notification_recipients nr WHERE nr.notification_id=n.id AND nr.user_id=viewer)
 AND EXISTS(SELECT 1 FROM workspace_access a WHERE a.user_id=viewer AND a.division_id=n.workspace_division_id)) OR
 (n.entity_type='leave_request' AND EXISTS(SELECT 1 FROM notification_recipients nr WHERE nr.notification_id=n.id AND nr.user_id=viewer)
 AND EXISTS(SELECT 1 FROM leave_requests l WHERE l.id=n.entity_id AND
 ((n.type='leave_request_created' AND r.name IN ('Admin','Management') AND l.user_id<>viewer) OR
 (n.type IN ('leave_request_approved','leave_request_rejected') AND l.user_id=viewer))))))
$$;
