-- Reporting roles are separate from authentication roles (users.role_id).
CREATE TABLE IF NOT EXISTS divisions (
  id SERIAL PRIMARY KEY, name VARCHAR(100) NOT NULL UNIQUE, sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS report_roles (
  id SERIAL PRIMARY KEY, division_id INTEGER NOT NULL REFERENCES divisions(id),
  name VARCHAR(50) NOT NULL, sort_order INTEGER NOT NULL DEFAULT 0,
  UNIQUE(division_id,name), UNIQUE(id,division_id)
);
CREATE TABLE IF NOT EXISTS job_categories (
  id SERIAL PRIMARY KEY, division_id INTEGER NOT NULL REFERENCES divisions(id),
  role_id INTEGER REFERENCES report_roles(id), name VARCHAR(100) NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0, UNIQUE(id,division_id),
  FOREIGN KEY(role_id,division_id) REFERENCES report_roles(id,division_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS job_categories_context ON job_categories(division_id,COALESCE(role_id,0),lower(btrim(name)));
CREATE TABLE IF NOT EXISTS jobs (
  id SERIAL PRIMARY KEY, division_id INTEGER NOT NULL REFERENCES divisions(id),
  role_id INTEGER REFERENCES report_roles(id), job_category_id INTEGER REFERENCES job_categories(id),
  name VARCHAR(255) NOT NULL CHECK(btrim(name) <> ''), description VARCHAR(2000) NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE, sort_order INTEGER NOT NULL DEFAULT 0,
  used_at TIMESTAMPTZ, created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  FOREIGN KEY(role_id,division_id) REFERENCES report_roles(id,division_id),
  FOREIGN KEY(job_category_id,division_id) REFERENCES job_categories(id,division_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS jobs_context_name ON jobs(division_id,COALESCE(role_id,0),COALESCE(job_category_id,0),lower(btrim(name)));
CREATE INDEX IF NOT EXISTS jobs_selection ON jobs(division_id,role_id,job_category_id,is_active,sort_order,id);
ALTER TABLE reports ADD COLUMN IF NOT EXISTS job_id INTEGER REFERENCES jobs(id);
CREATE INDEX IF NOT EXISTS reports_job ON reports(job_id);
-- Snapshots/PDFs remain immutable; this side table preserves their relational references.
CREATE TABLE IF NOT EXISTS weekly_archive_jobs (
  archive_id UUID NOT NULL REFERENCES weekly_report_archives(id),
  report_id TEXT NOT NULL, job_id INTEGER NOT NULL REFERENCES jobs(id),
  PRIMARY KEY(archive_id,report_id)
);
CREATE INDEX IF NOT EXISTS weekly_archive_jobs_job ON weekly_archive_jobs(job_id);
CREATE TABLE IF NOT EXISTS data_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW());

CREATE OR REPLACE FUNCTION validate_job_context() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.job_category_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM job_categories c WHERE c.id=NEW.job_category_id AND c.division_id=NEW.division_id AND c.role_id IS NOT DISTINCT FROM NEW.role_id
  ) THEN RAISE EXCEPTION 'Invalid job category context' USING ERRCODE='23514'; END IF;
  IF TG_OP='UPDATE' AND OLD.used_at IS NOT NULL AND
    (NEW.division_id IS DISTINCT FROM OLD.division_id OR NEW.role_id IS DISTINCT FROM OLD.role_id OR NEW.job_category_id IS DISTINCT FROM OLD.job_category_id)
  THEN RAISE EXCEPTION 'Referenced job context cannot change' USING ERRCODE='23514'; END IF;
  RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER jobs_validate_context BEFORE INSERT OR UPDATE ON jobs FOR EACH ROW EXECUTE FUNCTION validate_job_context();

-- Frozen import from the original Go and Svelte catalogs; never used by runtime selection.
DO $$ BEGIN
IF NOT EXISTS(SELECT 1 FROM data_migrations WHERE name='job-master-v1') THEN
INSERT INTO divisions(name,sort_order) VALUES ('Project Engineer',0),('AI Dev',1),('Keuangan',2),('Business Development',3),('HR',4),('PMO',5) ON CONFLICT DO NOTHING;
INSERT INTO report_roles(division_id,name,sort_order) SELECT id,'PE',0 FROM divisions WHERE name='Project Engineer' ON CONFLICT DO NOTHING;
INSERT INTO report_roles(division_id,name,sort_order) SELECT id,'QA',1 FROM divisions WHERE name='Project Engineer' ON CONFLICT DO NOTHING;
INSERT INTO report_roles(division_id,name,sort_order) SELECT id,'CIHPK',2 FROM divisions WHERE name='Project Engineer' ON CONFLICT DO NOTHING;
INSERT INTO job_categories(division_id,role_id,name,sort_order)
SELECT d.id,r.id,seed.category,seed.sort_order FROM (VALUES
 ('AI Dev','','Pengembangan',0),
 ('AI Dev','','Pengujian Model',1),
 ('AI Dev','','Deployment',2),
 ('AI Dev','','Dokumentasi',3),
 ('Keuangan','','Pembukuan',0),
 ('Keuangan','','Pembayaran',1),
 ('Keuangan','','Anggaran',2),
 ('Keuangan','','Pelaporan',3),
 ('Business Development','','Prospek',0),
 ('Business Development','','Proposal',1),
 ('Business Development','','Kemitraan',2),
 ('Business Development','','Tindak Lanjut',3),
 ('HR','','Kehadiran',0),
 ('HR','','Rekrutmen',1),
 ('HR','','Administrasi',2),
 ('HR','','Pengembangan SDM',3),
 ('PMO','','Milestone',0),
 ('PMO','','Rapat',1),
 ('PMO','','Risiko',2),
 ('PMO','','Pelaporan',3),
 ('Project Engineer','PE','MK supervisi',0),
 ('Project Engineer','PE','Non-MK supervisi',1),
 ('Project Engineer','QA','QA1',0),
 ('Project Engineer','QA','QA2',1),
 ('Project Engineer','CIHPK','JAKG',0),
 ('Project Engineer','CIHPK','Manggarai Hydrant',1),
 ('Project Engineer','CIHPK','Jember',2)
) AS seed(division,role,category,sort_order)
JOIN divisions d ON d.name=seed.division
LEFT JOIN report_roles r ON r.division_id=d.id AND r.name=seed.role
ON CONFLICT DO NOTHING;
INSERT INTO jobs(division_id,role_id,job_category_id,name,sort_order)
SELECT d.id,r.id,c.id,seed.name,seed.sort_order FROM (VALUES
 ('AI Dev','','Pengembangan','Pengembangan',0),
 ('AI Dev','','Pengujian Model','Pengujian Model',1),
 ('AI Dev','','Deployment','Deployment',2),
 ('AI Dev','','Dokumentasi','Dokumentasi',3),
 ('Keuangan','','Pembukuan','Pembukuan',0),
 ('Keuangan','','Pembayaran','Pembayaran',1),
 ('Keuangan','','Anggaran','Anggaran',2),
 ('Keuangan','','Pelaporan','Pelaporan',3),
 ('Business Development','','Prospek','Prospek',0),
 ('Business Development','','Proposal','Proposal',1),
 ('Business Development','','Kemitraan','Kemitraan',2),
 ('Business Development','','Tindak Lanjut','Tindak Lanjut',3),
 ('HR','','Kehadiran','Kehadiran',0),
 ('HR','','Rekrutmen','Rekrutmen',1),
 ('HR','','Administrasi','Administrasi',2),
 ('HR','','Pengembangan SDM','Pengembangan SDM',3),
 ('PMO','','Milestone','Milestone',0),
 ('PMO','','Rapat','Rapat',1),
 ('PMO','','Risiko','Risiko',2),
 ('PMO','','Pelaporan','Pelaporan',3),
 ('Project Engineer','PE','MK supervisi','Penataan Kawasan Stasiun Jember',0),
 ('Project Engineer','PE','MK supervisi','Pengawasan Renovasi Gudang Besar JAKG',1),
 ('Project Engineer','PE','MK supervisi','Pengawasan Instalasi Hidran BY Manggarai',2),
 ('Project Engineer','PE','Non-MK supervisi','MPS PIT',0),
 ('Project Engineer','PE','Non-MK supervisi','MPS SGR',1),
 ('Project Engineer','PE','Non-MK supervisi','MPS SGP',2),
 ('Project Engineer','PE','Non-MK supervisi','SHL JN',3),
 ('Project Engineer','PE','Non-MK supervisi','SOP Tata Kelola',4),
 ('Project Engineer','PE','Non-MK supervisi','Studi Daya Beli Gas Bumi',5),
 ('Project Engineer','PE','Non-MK supervisi','Review FS dan DED Pelabuhan Waipirit',6),
 ('Project Engineer','PE','Non-MK supervisi','Project Management Office',7),
 ('Project Engineer','PE','Non-MK supervisi','Reimagine Rumah IPS',8),
 ('Project Engineer','QA','QA1','MPS PIT',0),
 ('Project Engineer','QA','QA1','MPS SGR',1),
 ('Project Engineer','QA','QA1','MPS SGP',2),
 ('Project Engineer','QA','QA1','SHL JN',3),
 ('Project Engineer','QA','QA1','SOP Tata Kelola',4),
 ('Project Engineer','QA','QA1','Studi Daya Beli Gas Bumi',5),
 ('Project Engineer','QA','QA1','Review FS dan DED Pelabuhan Waipirit',6),
 ('Project Engineer','QA','QA1','Project Management Office',7),
 ('Project Engineer','QA','QA1','Reimagine Rumah IPS',8),
 ('Project Engineer','QA','QA1','Penataan Kawasan Stasiun Jember',9),
 ('Project Engineer','QA','QA1','Pengawasan Renovasi Gudang Besar JAKG',10),
 ('Project Engineer','QA','QA1','Pengawasan Instalasi Hidran BY Manggarai',11),
 ('Project Engineer','QA','QA1','Internal',12),
 ('Project Engineer','QA','QA2','MPS PIT',0),
 ('Project Engineer','QA','QA2','MPS SGR',1),
 ('Project Engineer','QA','QA2','MPS SGP',2),
 ('Project Engineer','QA','QA2','SHL JN',3),
 ('Project Engineer','QA','QA2','SOP Tata Kelola',4),
 ('Project Engineer','QA','QA2','Studi Daya Beli Gas Bumi',5),
 ('Project Engineer','QA','QA2','Review FS dan DED Pelabuhan Waipirit',6),
 ('Project Engineer','QA','QA2','Project Management Office',7),
 ('Project Engineer','QA','QA2','Reimagine Rumah IPS',8),
 ('Project Engineer','QA','QA2','Penataan Kawasan Stasiun Jember',9),
 ('Project Engineer','QA','QA2','Pengawasan Renovasi Gudang Besar JAKG',10),
 ('Project Engineer','QA','QA2','Pengawasan Instalasi Hidran BY Manggarai',11),
 ('Project Engineer','QA','QA2','Internal',12),
 ('Project Engineer','CIHPK','JAKG','Konsultan Pengawas JAKG (Tanggal SPMK: 12 Maret 2026)',0),
 ('Project Engineer','CIHPK','Manggarai Hydrant','Konsultan Pengawas Instalasi Hydrant Balai Yasa Manggarai (Tanggal SPMK: 08 Juni 2026)',0),
 ('Project Engineer','CIHPK','Jember','Supervisi Penataan Kawasan Stasiun Jember (Tanggal SPMK: 30 Desember 2025)',0)
) AS seed(division,role,category,name,sort_order)
JOIN divisions d ON d.name=seed.division
LEFT JOIN report_roles r ON r.division_id=d.id AND r.name=seed.role
JOIN job_categories c ON c.division_id=d.id AND c.role_id IS NOT DISTINCT FROM r.id AND c.name=seed.category
ON CONFLICT DO NOTHING;
INSERT INTO data_migrations(name) VALUES('job-master-v1');
END IF; END $$;

DO $$ DECLARE keep_id integer;
BEGIN
IF NOT EXISTS(SELECT 1 FROM data_migrations WHERE name='cihpk-category-merge-v1') THEN
  SELECT min(c.id) INTO keep_id FROM job_categories c JOIN report_roles r ON r.id=c.role_id WHERE r.name='CIHPK';
  UPDATE jobs SET job_category_id=keep_id WHERE keep_id IS NOT NULL AND role_id=(SELECT role_id FROM job_categories WHERE id=keep_id) AND job_category_id<>keep_id;
  DELETE FROM job_categories WHERE keep_id IS NOT NULL AND role_id=(SELECT role_id FROM job_categories WHERE id=keep_id) AND id<>keep_id;
  UPDATE job_categories SET name='MK/Supervisi',sort_order=0 WHERE id=keep_id;
  INSERT INTO data_migrations(name) VALUES('cihpk-category-merge-v1');
END IF;
END $$;

-- Preserve job/category IDs and references; only canonicalize the three CIHPK job labels.
DO $$ BEGIN
IF NOT EXISTS(SELECT 1 FROM data_migrations WHERE name='cihpk-job-labels-v2') THEN
 UPDATE jobs j SET name=CASE j.name
  WHEN 'Jember Station Area Redevelopment' THEN 'Supervisi Penataan Kawasan Stasiun Jember (Tanggal SPMK: 30 Desember 2025)'
  WHEN 'Penataan Kawasan Stasiun Jember' THEN 'Supervisi Penataan Kawasan Stasiun Jember (Tanggal SPMK: 30 Desember 2025)'
  WHEN 'JAKG Supervising Consultant (SPMK Date: March 12, 2026)' THEN 'Konsultan Pengawas JAKG (Tanggal SPMK: 12 Maret 2026)'
  WHEN 'Pengawasan Renovasi Gudang Besar JAKG' THEN 'Konsultan Pengawas JAKG (Tanggal SPMK: 12 Maret 2026)'
  WHEN 'Supervising Consultant for the Manggarai Balai Yasa Hydrant Installation (SPMK Date: June 8, 2026)' THEN 'Konsultan Pengawas Instalasi Hydrant Balai Yasa Manggarai (Tanggal SPMK: 08 Juni 2026)'
  WHEN 'Pengawasan Instalasi Hidran BY Manggarai' THEN 'Konsultan Pengawas Instalasi Hydrant Balai Yasa Manggarai (Tanggal SPMK: 08 Juni 2026)'
 END, updated_at=NOW()
 FROM report_roles r, divisions d
 WHERE j.role_id=r.id AND r.division_id=d.id AND r.name='CIHPK' AND d.name='Project Engineer'
 AND j.name IN ('Jember Station Area Redevelopment','Penataan Kawasan Stasiun Jember','JAKG Supervising Consultant (SPMK Date: March 12, 2026)','Pengawasan Renovasi Gudang Besar JAKG','Supervising Consultant for the Manggarai Balai Yasa Hydrant Installation (SPMK Date: June 8, 2026)','Pengawasan Instalasi Hidran BY Manggarai');
 INSERT INTO data_migrations(name) VALUES('cihpk-job-labels-v2');
END IF; END $$;

CREATE OR REPLACE VIEW job_context AS
SELECT j.*, d.name AS division_name, COALESCE(r.name,'') AS role_name, COALESCE(c.name,'') AS category_name
FROM jobs j JOIN divisions d ON d.id=j.division_id LEFT JOIN report_roles r ON r.id=j.role_id LEFT JOIN job_categories c ON c.id=j.job_category_id;

-- Context is mandatory. Case/whitespace normalization is safe only with exactly one match.
CREATE OR REPLACE FUNCTION legacy_job_match(division TEXT, role_name TEXT, category TEXT, work TEXT)
RETURNS INTEGER LANGUAGE sql STABLE AS $$
 SELECT CASE WHEN count(*)=1 THEN min(id) END FROM job_context j
 WHERE lower(btrim(j.division_name))=lower(btrim($1))
 AND lower(btrim(j.role_name))=lower(btrim(CASE WHEN $2='Project Engineer' THEN 'PE' WHEN $2 IN ('Quality Assurance','QA 1','QA 2','QA1','QA2') THEN 'QA' ELSE $2 END))
 AND ($2='CIHPK' OR lower(btrim(j.category_name))=lower(btrim(CASE $3 WHEN 'MK/Supervisi' THEN 'MK supervisi' WHEN 'Non MK/Non Supervisi' THEN 'Non-MK supervisi' WHEN 'QA 1' THEN 'QA1' WHEN 'QA 2' THEN 'QA2' WHEN 'QA' THEN 'QA1' ELSE $3 END)))
 AND lower(btrim(j.name))=lower(btrim(CASE WHEN $4='' AND $2='' THEN $3 ELSE $4 END))
$$;
DO $$ BEGIN
IF NOT EXISTS(SELECT 1 FROM data_migrations WHERE name='job-backfill-v1') THEN
 UPDATE reports SET job_id=legacy_job_match(primary_division,report_role,work_division,work) WHERE job_id IS NULL;
 INSERT INTO weekly_archive_jobs(archive_id,report_id,job_id)
 SELECT a.id,item->>'id',legacy_job_match(a.primary_division,COALESCE(item->>'role',''),COALESCE(item->>'workDivision',''),COALESCE(item->>'work',''))
 FROM weekly_report_archives a CROSS JOIN LATERAL jsonb_array_elements(a.snapshot) item
 WHERE item->>'id' IS NOT NULL AND legacy_job_match(a.primary_division,COALESCE(item->>'role',''),COALESCE(item->>'workDivision',''),COALESCE(item->>'work','')) IS NOT NULL
 ON CONFLICT DO NOTHING;
 UPDATE jobs SET used_at=NOW() WHERE id IN (SELECT job_id FROM reports UNION SELECT job_id FROM weekly_archive_jobs);
 INSERT INTO data_migrations(name) VALUES('job-backfill-v1');
END IF; END $$;
CREATE OR REPLACE VIEW unmatched_job_references AS
 SELECT 'reports' AS source,id::text AS record_id,primary_division,report_role AS role,work_division AS category,work FROM reports WHERE job_id IS NULL
 UNION ALL
 SELECT 'weekly_report_archives',a.id::text || '/' || COALESCE(item->>'id','?'),a.primary_division,item->>'role',item->>'workDivision',item->>'work'
 FROM weekly_report_archives a CROSS JOIN LATERAL jsonb_array_elements(a.snapshot) item
 WHERE NOT EXISTS(SELECT 1 FROM weekly_archive_jobs ref WHERE ref.archive_id=a.id AND ref.report_id=item->>'id');
CREATE OR REPLACE FUNCTION retain_used_job() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.used_at IS NOT NULL THEN RAISE EXCEPTION 'Referenced job cannot be deleted' USING ERRCODE='23503'; END IF;
 RETURN OLD;
END $$;
CREATE OR REPLACE TRIGGER jobs_retain_used BEFORE DELETE ON jobs FOR EACH ROW EXECUTE FUNCTION retain_used_job();
