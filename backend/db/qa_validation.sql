-- Add audit information without inventing approvals for historical Done items.
ALTER TABLE work_items ADD COLUMN IF NOT EXISTS qa_validated_at TIMESTAMPTZ;
ALTER TABLE work_items ADD COLUMN IF NOT EXISTS qa_validated_by UUID REFERENCES users(id);
ALTER TABLE work_items DROP CONSTRAINT IF EXISTS work_items_qa_validated_by_fkey;
ALTER TABLE work_items ADD CONSTRAINT work_items_qa_validated_by_fkey FOREIGN KEY(qa_validated_by) REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS work_items_qa_validator ON work_items(qa_validated_by) WHERE qa_validated_by IS NOT NULL;
ALTER TABLE reports ALTER COLUMN description TYPE TEXT;

-- NULL marks an existing revision whose historical count was never recorded.
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='work_items' AND column_name='revision_count') THEN
  ALTER TABLE work_items ADD COLUMN revision_count INTEGER DEFAULT 0 CHECK(revision_count >= 0);
  UPDATE work_items SET revision_count=NULL WHERE status='NEED_REVISION';
 END IF;
END $$;
