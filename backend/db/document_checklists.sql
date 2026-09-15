CREATE TABLE IF NOT EXISTS document_checklists (
 id SERIAL PRIMARY KEY,
 job_id INTEGER NOT NULL UNIQUE REFERENCES jobs(id),
 require_digital_signature BOOLEAN NOT NULL DEFAULT FALSE,
 version INTEGER NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
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
 document_name VARCHAR(255) NOT NULL CHECK (btrim(document_name) <> ''),
 digital_signed BOOLEAN NOT NULL DEFAULT FALSE,
 document_position VARCHAR(255) NOT NULL DEFAULT '',
 sent_date DATE,
 notes VARCHAR(4000) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 deleted_at TIMESTAMPTZ,
 UNIQUE(id,checklist_id)
);
-- IDs also provide stable insertion order; display numbers are calculated by the UI.
CREATE INDEX IF NOT EXISTS document_checklist_items_order ON document_checklist_items(checklist_id,id);
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
