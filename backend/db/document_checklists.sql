CREATE TABLE IF NOT EXISTS document_checklists (
 id SERIAL PRIMARY KEY,
 job_id INTEGER NOT NULL REFERENCES jobs(id),
 title VARCHAR(255) NOT NULL CHECK (btrim(title) <> ''),
 header_labels JSONB NOT NULL DEFAULT '{}'::jsonb,
 require_digital_signature BOOLEAN NOT NULL DEFAULT FALSE,
 version INTEGER NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Upgrade existing single-table checklists without changing their identities.
ALTER TABLE document_checklists ADD COLUMN IF NOT EXISTS title VARCHAR(255) NOT NULL DEFAULT 'Checklist Dokumen' CHECK (btrim(title) <> '');
ALTER TABLE document_checklists ALTER COLUMN title DROP DEFAULT;
ALTER TABLE document_checklists ADD COLUMN IF NOT EXISTS header_labels JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE document_checklists DROP CONSTRAINT IF EXISTS document_checklists_job_id_key;
CREATE INDEX IF NOT EXISTS document_checklists_job_order ON document_checklists(job_id,id);
CREATE TABLE IF NOT EXISTS document_checklist_signers (
 id SERIAL PRIMARY KEY,
 checklist_id INTEGER NOT NULL REFERENCES document_checklists(id),
 signer_type TEXT NOT NULL CHECK (signer_type IN ('internal','external')),
 name VARCHAR(255) NOT NULL CHECK (btrim(name) <> ''),
 position VARCHAR(255) NOT NULL CHECK (btrim(position) <> ''),
 display_order INTEGER NOT NULL CHECK (display_order >= 0),
 archived_at TIMESTAMPTZ,
 UNIQUE(id,checklist_id)
);
CREATE INDEX IF NOT EXISTS document_checklist_signers_order ON document_checklist_signers(checklist_id,display_order,id);
CREATE TABLE IF NOT EXISTS document_checklist_items (
 id SERIAL PRIMARY KEY,
 checklist_id INTEGER NOT NULL REFERENCES document_checklists(id),
 parent_item_id INTEGER,
 document_name VARCHAR(255) NOT NULL CHECK (btrim(document_name) <> ''),
 digital_signed BOOLEAN NOT NULL DEFAULT FALSE,
 document_position VARCHAR(255) NOT NULL DEFAULT '',
 sent_date DATE,
 notes VARCHAR(4000) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 deleted_at TIMESTAMPTZ,
 UNIQUE(id,checklist_id),
 CONSTRAINT document_checklist_items_parent FOREIGN KEY(parent_item_id,checklist_id) REFERENCES document_checklist_items(id,checklist_id)
);
ALTER TABLE document_checklist_items ADD COLUMN IF NOT EXISTS parent_item_id INTEGER;
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='document_checklist_items'::regclass AND conname='document_checklist_items_parent') THEN
  ALTER TABLE document_checklist_items ADD CONSTRAINT document_checklist_items_parent FOREIGN KEY(parent_item_id,checklist_id) REFERENCES document_checklist_items(id,checklist_id);
 END IF;
END $$;
-- IDs provide stable sibling insertion order.
CREATE INDEX IF NOT EXISTS document_checklist_items_order ON document_checklist_items(checklist_id,id);
CREATE INDEX IF NOT EXISTS document_checklist_items_parent_order ON document_checklist_items(checklist_id,parent_item_id,id);
CREATE TABLE IF NOT EXISTS document_checklist_signatures (
 checklist_id INTEGER NOT NULL,
 item_id INTEGER NOT NULL,
 signer_id INTEGER NOT NULL,
 signed BOOLEAN NOT NULL DEFAULT FALSE,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(item_id,signer_id),
 FOREIGN KEY(item_id,checklist_id) REFERENCES document_checklist_items(id,checklist_id),
 FOREIGN KEY(signer_id,checklist_id) REFERENCES document_checklist_signers(id,checklist_id)
);
CREATE INDEX IF NOT EXISTS document_checklist_signatures_signer ON document_checklist_signatures(signer_id,checklist_id);
