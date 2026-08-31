CREATE TABLE users (
  id UUID PRIMARY KEY,
  username VARCHAR(50) NOT NULL UNIQUE,
  display_name VARCHAR(100) NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE report_sequences (
  primary_division VARCHAR(100) NOT NULL,
  work_division VARCHAR(100) NOT NULL,
  last_number INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (primary_division, work_division)
);

CREATE TABLE reports (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  primary_division VARCHAR(100) NOT NULL,
  work_division VARCHAR(100) NOT NULL,
  number INTEGER NOT NULL,
  description VARCHAR(500) NOT NULL,
  completed_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (primary_division, work_division, number)
);

CREATE INDEX reports_by_division ON reports (primary_division, work_division, number);
