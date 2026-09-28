-- Weekly history owns its data; deleting/changing a user cannot change a snapshot.
CREATE TABLE IF NOT EXISTS weekly_report_archives (
  id UUID PRIMARY KEY,
  primary_division VARCHAR(100) NOT NULL,
  week_start TIMESTAMPTZ NOT NULL,
  week_end TIMESTAMPTZ NOT NULL,
  snapshot JSONB NOT NULL CHECK (jsonb_typeof(snapshot) = 'array' AND jsonb_array_length(snapshot) > 0),
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'finalized', 'failed')),
  finalized_at TIMESTAMPTZ,
  pdf_filename TEXT NOT NULL DEFAULT '',
  pdf_storage_key TEXT NOT NULL DEFAULT '',
  pdf_sha256 TEXT NOT NULL DEFAULT '',
  pdf_generated_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  reset_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (primary_division, week_start, week_end),
  CHECK (week_end > week_start)
);
CREATE INDEX IF NOT EXISTS weekly_archive_history ON weekly_report_archives (week_start DESC, id);
CREATE TABLE IF NOT EXISTS weekly_report_email_deliveries (
  id UUID PRIMARY KEY,
  archive_id UUID NOT NULL REFERENCES weekly_report_archives(id),
  recipient_user_id UUID NOT NULL,
  recipient_email TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sending','sent','failed','uncertain')),
  attempt_count INTEGER NOT NULL DEFAULT 0,
  last_attempt_at TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (archive_id, recipient_email)
);

-- Working sequences are weekly; a retained older week cannot affect new numbering.
ALTER TABLE report_sequences ADD COLUMN IF NOT EXISTS week_start DATE NOT NULL
  DEFAULT date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Jakarta')::date;
ALTER TABLE reports ADD COLUMN IF NOT EXISTS report_week DATE GENERATED ALWAYS AS
  (date_trunc('week', completed_at AT TIME ZONE 'Asia/Jakarta')::date) STORED;
DO $$ DECLARE expression TEXT; BEGIN
 SELECT pg_get_expr(d.adbin,d.adrelid) INTO expression
 FROM pg_attribute a JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
 WHERE a.attrelid='reports'::regclass AND a.attname='report_week';
 IF expression LIKE '%interval ''1 day''%' THEN
  DROP INDEX IF EXISTS reports_job_week_number_key;
  DROP INDEX IF EXISTS reports_legacy_week_number_key;
  ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_week_number_key;
  ALTER TABLE reports DROP COLUMN report_week;
  ALTER TABLE reports ADD COLUMN report_week DATE GENERATED ALWAYS AS
   (date_trunc('week',completed_at AT TIME ZONE 'Asia/Jakarta')::date) STORED;
  WITH duplicates AS (
   SELECT ctid,row_number() OVER(PARTITION BY job_id,report_week,number ORDER BY completed_at,id) AS n,
    max(number) OVER(PARTITION BY job_id,report_week) AS highest
   FROM reports WHERE job_id IS NOT NULL
  ) UPDATE reports r SET number=d.highest+d.n FROM duplicates d WHERE r.ctid=d.ctid AND d.n>1;
  WITH duplicates AS (
   SELECT ctid,row_number() OVER(PARTITION BY primary_division,work_division,report_week,number ORDER BY completed_at,id) AS n,
    max(number) OVER(PARTITION BY primary_division,work_division,report_week) AS highest
   FROM reports WHERE job_id IS NULL
  ) UPDATE reports r SET number=d.highest+d.n FROM duplicates d WHERE r.ctid=d.ctid AND d.n>1;
  DELETE FROM report_sequences;
  INSERT INTO report_sequences(primary_division,work_division,week_start,last_number)
  SELECT primary_division,work_division,report_week,MAX(number) FROM reports WHERE job_id IS NULL GROUP BY 1,2,3;
 END IF;
END $$;
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='report_sequences'::regclass AND conname='report_sequences_job_scope') THEN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='report_sequences'::regclass AND conname='report_sequences_week_pk') THEN
    ALTER TABLE report_sequences DROP CONSTRAINT report_sequences_pkey;
    ALTER TABLE report_sequences ADD CONSTRAINT report_sequences_week_pk PRIMARY KEY(primary_division,work_division,week_start);
	-- Legacy counters had no period. Rebuild them from retained report rows below.
	DELETE FROM report_sequences;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='reports'::regclass AND conname='reports_week_number_key') THEN
    ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_primary_division_work_division_number_key;
    ALTER TABLE reports ADD CONSTRAINT reports_week_number_key UNIQUE(primary_division,work_division,report_week,number);
  END IF;
INSERT INTO report_sequences(primary_division,work_division,week_start,last_number)
SELECT primary_division,work_division,report_week,MAX(number) FROM reports GROUP BY 1,2,3
ON CONFLICT(primary_division,work_division,week_start) DO UPDATE
SET last_number=GREATEST(report_sequences.last_number,EXCLUDED.last_number);
 END IF;
END $$;

CREATE OR REPLACE FUNCTION protect_weekly_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN RAISE EXCEPTION 'Weekly archives are permanent'; END IF;
  IF NEW.snapshot IS DISTINCT FROM OLD.snapshot OR NEW.primary_division <> OLD.primary_division
    OR NEW.week_start <> OLD.week_start OR NEW.week_end <> OLD.week_end OR NEW.id <> OLD.id THEN
    RAISE EXCEPTION 'Weekly snapshots are immutable';
  END IF;
  RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER weekly_snapshot_immutable BEFORE UPDATE OR DELETE ON weekly_report_archives
FOR EACH ROW EXECUTE FUNCTION protect_weekly_snapshot();
