CREATE TABLE IF NOT EXISTS ai_plan_drafts (
 id char(64) PRIMARY KEY,
 workspace_id integer NOT NULL REFERENCES divisions(id),
 plan_date date NOT NULL,
 snapshot_hash char(64) NOT NULL,
 draft jsonb NOT NULL,
 imported_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ai_plan_drafts_workspace_date ON ai_plan_drafts(workspace_id,plan_date DESC,imported_at DESC);
CREATE TABLE IF NOT EXISTS ai_plan_recommendations (
 draft_id char(64) NOT NULL REFERENCES ai_plan_drafts(id),
 id char(64) NOT NULL,
 data jsonb NOT NULL,
 source_evidence jsonb NOT NULL,
 decision text NOT NULL DEFAULT 'PENDING' CHECK (decision IN ('PENDING','SUITABLE','NEEDS_REVISION','REJECTED')),
 note text NOT NULL DEFAULT '',
 reviewer_id uuid REFERENCES users(id) ON DELETE SET NULL,
 reviewer_identity text NOT NULL DEFAULT '',
 reviewer_name text NOT NULL DEFAULT '',
 reviewed_at timestamptz,
 version integer NOT NULL DEFAULT 0,
 PRIMARY KEY(draft_id,id)
);
CREATE TABLE IF NOT EXISTS ai_plan_reviews (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 draft_id char(64) NOT NULL,
 recommendation_id char(64) NOT NULL,
 version integer NOT NULL,
 decision text NOT NULL,
 note text NOT NULL,
 reviewer_id uuid REFERENCES users(id) ON DELETE SET NULL,
 reviewer_identity text NOT NULL DEFAULT '',
 reviewer_name text NOT NULL DEFAULT '',
 reviewed_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(draft_id,recommendation_id) REFERENCES ai_plan_recommendations(draft_id,id),
 UNIQUE(draft_id,recommendation_id,version)
);
CREATE INDEX IF NOT EXISTS ai_plan_reviews_recommendation ON ai_plan_reviews(draft_id,recommendation_id,version DESC);
ALTER TABLE ai_plan_recommendations ADD COLUMN IF NOT EXISTS reviewer_identity text NOT NULL DEFAULT '';
ALTER TABLE ai_plan_recommendations ADD COLUMN IF NOT EXISTS reviewer_name text NOT NULL DEFAULT '';
ALTER TABLE ai_plan_reviews ADD COLUMN IF NOT EXISTS reviewer_identity text NOT NULL DEFAULT '';
ALTER TABLE ai_plan_reviews ADD COLUMN IF NOT EXISTS reviewer_name text NOT NULL DEFAULT '';
ALTER TABLE ai_plan_reviews ALTER COLUMN reviewer_id DROP NOT NULL;
UPDATE ai_plan_recommendations r SET reviewer_identity=COALESCE(NULLIF(r.reviewer_identity,''),r.reviewer_id::text), reviewer_name=COALESCE(NULLIF(r.reviewer_name,''),u.name)
 FROM users u WHERE r.reviewer_id=u.id AND (r.reviewer_identity='' OR r.reviewer_name='');
UPDATE ai_plan_reviews h SET reviewer_identity=COALESCE(NULLIF(h.reviewer_identity,''),h.reviewer_id::text), reviewer_name=COALESCE(NULLIF(h.reviewer_name,''),u.name)
 FROM users u WHERE h.reviewer_id=u.id AND (h.reviewer_identity='' OR h.reviewer_name='');
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='ai_plan_recommendations'::regclass AND conname='ai_plan_recommendations_reviewer_id_fkey' AND confdeltype<>'n') THEN
  ALTER TABLE ai_plan_recommendations DROP CONSTRAINT ai_plan_recommendations_reviewer_id_fkey;
  ALTER TABLE ai_plan_recommendations ADD CONSTRAINT ai_plan_recommendations_reviewer_id_fkey FOREIGN KEY(reviewer_id) REFERENCES users(id) ON DELETE SET NULL;
 END IF;
 IF EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='ai_plan_reviews'::regclass AND conname='ai_plan_reviews_reviewer_id_fkey' AND confdeltype<>'n') THEN
  ALTER TABLE ai_plan_reviews DROP CONSTRAINT ai_plan_reviews_reviewer_id_fkey;
  ALTER TABLE ai_plan_reviews ADD CONSTRAINT ai_plan_reviews_reviewer_id_fkey FOREIGN KEY(reviewer_id) REFERENCES users(id) ON DELETE SET NULL;
 END IF;
END $$;
