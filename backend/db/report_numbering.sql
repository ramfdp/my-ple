-- Jobs already encode division/report-role/category. Retain nullable legacy
-- counters, but allocate new work only in a relational job/week scope.
ALTER TABLE report_sequences ADD COLUMN IF NOT EXISTS job_id INTEGER REFERENCES jobs(id);
ALTER TABLE report_sequences DROP CONSTRAINT IF EXISTS report_sequences_week_pk;
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='report_sequences'::regclass AND conname='report_sequences_job_scope') THEN
  ALTER TABLE report_sequences ADD CONSTRAINT report_sequences_job_scope
   UNIQUE NULLS NOT DISTINCT(primary_division,work_division,week_start,job_id);
 END IF;
END $$;

-- No report is renumbered. Conflicting legacy job numbers fail the migration
-- rather than silently modifying history. Unresolved reports retain protection.
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_week_number_key;
CREATE UNIQUE INDEX IF NOT EXISTS reports_job_week_number_key
 ON reports(job_id,report_week,number) WHERE job_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS reports_legacy_week_number_key
 ON reports(primary_division,work_division,report_week,number) WHERE job_id IS NULL;

-- Initialize from preserved numbers, including any gaps. Never lower a counter
-- after a deletion, job move, or repeated migration.
INSERT INTO report_sequences(primary_division,work_division,week_start,job_id,last_number)
SELECT j.division_name,j.category_name,r.report_week,r.job_id,MAX(r.number)
FROM reports r JOIN job_context j ON j.id=r.job_id GROUP BY 1,2,3,4
ON CONFLICT ON CONSTRAINT report_sequences_job_scope DO UPDATE
 SET last_number=GREATEST(report_sequences.last_number,EXCLUDED.last_number);
