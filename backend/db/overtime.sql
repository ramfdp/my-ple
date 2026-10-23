CREATE TABLE IF NOT EXISTS overtime_sheets (
 id UUID PRIMARY KEY,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 period DATE NOT NULL CHECK(EXTRACT(DAY FROM period)=1),
 status TEXT NOT NULL DEFAULT 'DRAFT' CHECK(status IN ('DRAFT','SUBMITTED','APPROVED','REJECTED')),
 version INTEGER NOT NULL DEFAULT 0,
 submitted_at TIMESTAMPTZ,
 reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
 reviewed_at TIMESTAMPTZ,
 reviewer_note VARCHAR(4000) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 created_notified BOOLEAN NOT NULL DEFAULT false,
 reviewed_notified BOOLEAN NOT NULL DEFAULT false,
 UNIQUE(user_id,period),
 CHECK(reviewed_by IS NULL OR reviewed_by<>user_id),
 CHECK((status IN ('DRAFT','SUBMITTED') AND reviewed_at IS NULL) OR status IN ('APPROVED','REJECTED'))
);

CREATE TABLE IF NOT EXISTS overtime_entries (
 sheet_id UUID NOT NULL REFERENCES overtime_sheets(id) ON DELETE CASCADE,
 work_date DATE NOT NULL,
 start_time TIME NOT NULL,
 end_time TIME NOT NULL,
 day_type TEXT NOT NULL CHECK(day_type IN ('WORKDAY','HOLIDAY')),
 is_override BOOLEAN NOT NULL DEFAULT false,
 override_reason VARCHAR(500) NOT NULL DEFAULT '',
 project VARCHAR(255) NOT NULL CHECK(btrim(project)<>''),
 description VARCHAR(4000) NOT NULL CHECK(btrim(description)<>''),
 actual_minutes INTEGER NOT NULL CHECK(actual_minutes>=0),
 weighted_minutes INTEGER NOT NULL CHECK(weighted_minutes>=0),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(sheet_id,work_date)
);

CREATE INDEX IF NOT EXISTS overtime_sheets_period_status ON overtime_sheets(period,status,updated_at DESC);
CREATE INDEX IF NOT EXISTS overtime_sheets_user_period ON overtime_sheets(user_id,period DESC);

ALTER TABLE zoom_booking_email_deliveries ADD COLUMN IF NOT EXISTS overtime_sheet_id UUID REFERENCES overtime_sheets(id) ON DELETE CASCADE;

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
 (n.type IN ('leave_request_approved','leave_request_rejected') AND l.user_id=viewer)))) OR
 (n.entity_type='overtime_sheet' AND EXISTS(SELECT 1 FROM notification_recipients nr WHERE nr.notification_id=n.id AND nr.user_id=viewer)
 AND EXISTS(SELECT 1 FROM overtime_sheets o WHERE o.id=n.entity_id AND
 ((n.type='overtime_submitted' AND r.name IN ('Finance','Admin') AND o.user_id<>viewer) OR
 (n.type IN ('overtime_approved','overtime_rejected') AND o.user_id=viewer))))))
$$;
