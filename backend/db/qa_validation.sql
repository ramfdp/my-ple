-- Add audit information without inventing approvals for historical Done items.
ALTER TABLE work_items ADD COLUMN IF NOT EXISTS qa_validated_at TIMESTAMPTZ;
ALTER TABLE work_items ADD COLUMN IF NOT EXISTS qa_validated_by UUID REFERENCES users(id);
CREATE INDEX IF NOT EXISTS work_items_qa_validator ON work_items(qa_validated_by) WHERE qa_validated_by IS NOT NULL;
ALTER TABLE reports ALTER COLUMN description TYPE TEXT;
