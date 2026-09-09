-- Reuse divisions as workspace identities and account roles as membership.
CREATE OR REPLACE VIEW workspace_members AS
SELECT u.id AS user_id,d.id AS division_id,u.name,u.avatar_url
FROM users u JOIN roles r ON r.id=u.role_id
JOIN divisions d ON d.name=CASE r.name WHEN 'Finance' THEN 'Keuangan' WHEN 'IT' THEN 'AI Dev' ELSE r.name END
WHERE u.is_active;

CREATE OR REPLACE VIEW workspace_access AS
SELECT u.id AS user_id,d.id AS division_id,r.name IN ('Admin','Management') AS can_manage
FROM users u JOIN roles r ON r.id=u.role_id CROSS JOIN divisions d
WHERE u.is_active AND (r.name IN ('Admin','Management') OR EXISTS(SELECT 1 FROM workspace_members m WHERE m.user_id=u.id AND m.division_id=d.id));

CREATE TABLE IF NOT EXISTS work_items (
 id UUID PRIMARY KEY, division_id INTEGER NOT NULL REFERENCES divisions(id),
 job_id INTEGER REFERENCES jobs(id), title VARCHAR(200) NOT NULL CHECK(btrim(title)<>''),
 description VARCHAR(10000) NOT NULL DEFAULT '', related_url VARCHAR(2000) NOT NULL DEFAULT '',
 start_date DATE, due_date DATE,
 status TEXT NOT NULL DEFAULT 'TODO' CHECK(status IN ('TODO','IN_PROGRESS','NEED_REVISION','DONE')),
 priority TEXT NOT NULL DEFAULT 'MEDIUM' CHECK(priority IN ('LOW','MEDIUM','HIGH','URGENT')),
 created_by UUID NOT NULL REFERENCES users(id), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), version INTEGER NOT NULL DEFAULT 1,
 archived_at TIMESTAMPTZ, CHECK(due_date>=start_date)
);
CREATE INDEX IF NOT EXISTS work_items_workspace_status ON work_items(division_id,status) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS work_items_workspace_due ON work_items(division_id,due_date) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS work_items_job ON work_items(job_id);
CREATE INDEX IF NOT EXISTS work_items_creator ON work_items(created_by);
CREATE TABLE IF NOT EXISTS work_item_assignees (
 work_item_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
 user_id UUID NOT NULL REFERENCES users(id), assigned_by UUID NOT NULL REFERENCES users(id),
 assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), PRIMARY KEY(work_item_id,user_id)
);
CREATE INDEX IF NOT EXISTS work_assignees_user ON work_item_assignees(user_id,work_item_id);
CREATE TABLE IF NOT EXISTS work_item_comments (
 id UUID PRIMARY KEY, work_item_id UUID NOT NULL REFERENCES work_items(id),
 user_id UUID NOT NULL REFERENCES users(id), content VARCHAR(4000) NOT NULL CHECK(btrim(content)<>''),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS work_comments_task ON work_item_comments(work_item_id,created_at);
CREATE TABLE IF NOT EXISTS work_item_activity (
 id UUID PRIMARY KEY, work_item_id UUID NOT NULL REFERENCES work_items(id),
 actor_user_id UUID NOT NULL REFERENCES users(id), action TEXT NOT NULL, field_name TEXT NOT NULL DEFAULT '',
 old_value JSONB, new_value JSONB, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS work_activity_task ON work_item_activity(work_item_id,created_at);
CREATE TABLE IF NOT EXISTS workspace_pages (
 id UUID PRIMARY KEY, division_id INTEGER NOT NULL REFERENCES divisions(id),
 parent_id UUID REFERENCES workspace_pages(id), title VARCHAR(200) NOT NULL CHECK(btrim(title)<>''),
 content JSONB NOT NULL DEFAULT '[]', visibility TEXT NOT NULL CHECK(visibility IN ('PERSONAL','DIVISION')),
 created_by UUID NOT NULL REFERENCES users(id), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), version INTEGER NOT NULL DEFAULT 1,
 CHECK(parent_id<>id), CHECK(jsonb_typeof(content)='array')
);
CREATE INDEX IF NOT EXISTS workspace_pages_division ON workspace_pages(division_id,visibility,created_by);
CREATE INDEX IF NOT EXISTS workspace_pages_parent ON workspace_pages(parent_id);

-- Preserve the existing broadcast notification model for Zoom; new events are private.
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS workspace_division_id INTEGER REFERENCES divisions(id);
CREATE TABLE IF NOT EXISTS notification_recipients (
 notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, PRIMARY KEY(notification_id,user_id)
);
CREATE INDEX IF NOT EXISTS notification_recipients_user ON notification_recipients(user_id,notification_id);

CREATE OR REPLACE FUNCTION notification_visible(notification UUID, viewer UUID) RETURNS BOOLEAN
LANGUAGE SQL STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM notifications n JOIN users u ON u.id=viewer JOIN roles r ON r.id=u.role_id
 WHERE n.id=notification AND u.is_active AND n.actor_user_id<>viewer AND (
 (n.type='zoom_booking_created' AND n.workspace_division_id IS NULL) OR
 (n.workspace_division_id IS NOT NULL AND EXISTS(SELECT 1 FROM notification_recipients nr WHERE nr.notification_id=n.id AND nr.user_id=viewer)
 AND EXISTS(SELECT 1 FROM workspace_access a WHERE a.user_id=viewer AND a.division_id=n.workspace_division_id))))
$$;

CREATE OR REPLACE FUNCTION validate_workspace_job() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.job_id IS NOT NULL THEN
  UPDATE jobs SET used_at=COALESCE(used_at,NOW()) WHERE id=NEW.job_id AND division_id=NEW.division_id;
  IF NOT FOUND THEN RAISE EXCEPTION 'Invalid workspace job' USING ERRCODE='23514'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER work_item_job_context BEFORE INSERT OR UPDATE OF job_id,division_id ON work_items
FOR EACH ROW EXECUTE FUNCTION validate_workspace_job();
