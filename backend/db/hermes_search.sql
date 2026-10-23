CREATE INDEX IF NOT EXISTS work_items_hermes_search ON work_items USING GIN (to_tsvector('simple', title || ' ' || description));
CREATE INDEX IF NOT EXISTS reports_hermes_search ON reports USING GIN (to_tsvector('simple', work || ' ' || description));
CREATE INDEX IF NOT EXISTS weekly_archive_hermes_search ON weekly_report_archives USING GIN (to_tsvector('simple', snapshot::text));
