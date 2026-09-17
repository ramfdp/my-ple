CREATE TABLE IF NOT EXISTS billing_stages (
 id SERIAL PRIMARY KEY, job_id INTEGER NOT NULL REFERENCES jobs(id),
 name VARCHAR(255) NOT NULL CHECK(btrim(name)<>''), period_label VARCHAR(255) NOT NULL CHECK(btrim(period_label)<>''),
 sort_order INTEGER NOT NULL DEFAULT 0, notes TEXT NOT NULL DEFAULT '', version INTEGER NOT NULL DEFAULT 1,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS billing_stages_job ON billing_stages(job_id,id);
CREATE TABLE IF NOT EXISTS billing_milestones (
 id SERIAL PRIMARY KEY, stage_id INTEGER NOT NULL REFERENCES billing_stages(id),
 name VARCHAR(255) NOT NULL CHECK(btrim(name)<>''), sort_order INTEGER NOT NULL,
 notes TEXT NOT NULL DEFAULT '', updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), archived_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS billing_milestones_stage ON billing_milestones(stage_id,sort_order,id);
CREATE TABLE IF NOT EXISTS billing_documents (
 id SERIAL PRIMARY KEY, milestone_id INTEGER NOT NULL REFERENCES billing_milestones(id),
 name VARCHAR(255) NOT NULL CHECK(btrim(name)<>''), sort_order INTEGER NOT NULL,
 status TEXT NOT NULL DEFAULT 'BELUM ADA' CHECK(status IN ('BELUM ADA','PEMBUATAN','KURANG LENGKAP','SIRKULIR TTD','OK')),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), archived_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS billing_documents_milestone ON billing_documents(milestone_id,sort_order,id);
