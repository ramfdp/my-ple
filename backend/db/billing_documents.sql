CREATE TABLE IF NOT EXISTS billing_stages (
 id SERIAL PRIMARY KEY, job_id INTEGER NOT NULL REFERENCES jobs(id),
 name VARCHAR(255) NOT NULL CHECK(btrim(name)<>''), period_label VARCHAR(255) NOT NULL DEFAULT '',
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
ALTER TABLE billing_documents ADD COLUMN IF NOT EXISTS stage_id INTEGER REFERENCES billing_stages(id);
ALTER TABLE billing_documents ALTER COLUMN milestone_id DROP NOT NULL;
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='billing_documents'::regclass AND conname='billing_document_owner') THEN
  ALTER TABLE billing_documents ADD CONSTRAINT billing_document_owner CHECK ((milestone_id IS NOT NULL) <> (stage_id IS NOT NULL));
 END IF;
END $$;
CREATE INDEX IF NOT EXISTS billing_documents_stage ON billing_documents(stage_id,sort_order,id);

-- Billing periods are optional; retain existing values for older records/clients.
ALTER TABLE billing_stages DROP CONSTRAINT IF EXISTS billing_stages_period_label_check;
ALTER TABLE billing_stages ALTER COLUMN period_label SET DEFAULT '';
