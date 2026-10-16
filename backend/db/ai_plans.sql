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
 reviewer_id uuid REFERENCES users(id),
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
 reviewer_id uuid NOT NULL REFERENCES users(id),
 reviewed_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(draft_id,recommendation_id) REFERENCES ai_plan_recommendations(draft_id,id),
 UNIQUE(draft_id,recommendation_id,version)
);
CREATE INDEX IF NOT EXISTS ai_plan_reviews_recommendation ON ai_plan_reviews(draft_id,recommendation_id,version DESC);
