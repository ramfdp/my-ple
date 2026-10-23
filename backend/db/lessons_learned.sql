CREATE TABLE IF NOT EXISTS lessons_learned (
  id SERIAL PRIMARY KEY,
  job_id INTEGER NOT NULL REFERENCES jobs(id),
  kind TEXT NOT NULL CHECK (kind IN ('failure','success')),
  found_on DATE,
  title TEXT NOT NULL CHECK (btrim(title) <> ''),
  location TEXT NOT NULL DEFAULT '',
  internal_factor TEXT NOT NULL DEFAULT '',
  external_factor TEXT NOT NULL DEFAULT '',
  impact TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','in_progress','resolved')),
  notes TEXT NOT NULL DEFAULT '',
  resolved_on DATE,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
ALTER TABLE lessons_learned ALTER COLUMN job_id DROP NOT NULL;
ALTER TABLE lessons_learned ADD COLUMN IF NOT EXISTS source_group TEXT NOT NULL DEFAULT '';
ALTER TABLE lessons_learned ADD COLUMN IF NOT EXISTS source_project TEXT NOT NULL DEFAULT '';
ALTER TABLE lessons_learned ADD COLUMN IF NOT EXISTS source_row INTEGER;
ALTER TABLE lessons_learned ADD COLUMN IF NOT EXISTS source_number TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS lessons_learned_source ON lessons_learned(source_row,kind);
CREATE INDEX IF NOT EXISTS lessons_learned_job ON lessons_learned(job_id,kind,id) WHERE deleted_at IS NULL;
