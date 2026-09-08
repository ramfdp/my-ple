CREATE TABLE IF NOT EXISTS job_pics (
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (job_id, user_id)
);
CREATE INDEX IF NOT EXISTS job_pics_user_idx ON job_pics(user_id);

-- Current account names/status, aggregated once per job for report joins.
CREATE OR REPLACE VIEW job_pic_summary AS
SELECT p.job_id, jsonb_agg(jsonb_build_object(
    'userId', u.id, 'name', u.name, 'isActive', u.is_active
) ORDER BY lower(u.name), u.id) AS pics
FROM job_pics p JOIN users u ON u.id=p.user_id GROUP BY p.job_id;
