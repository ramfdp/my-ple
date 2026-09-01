-- Roles table
CREATE TABLE roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Predefined roles
INSERT INTO roles (name) VALUES
  ('Admin'),
  ('Management'),
  ('Project Engineer'),
  ('HR'),
  ('Finance'),
  ('IT'),
  ('Business Development'),
  ('PMO')
ON CONFLICT (name) DO NOTHING;

-- Users table (each user has exactly one role)
CREATE TABLE users (
  id UUID PRIMARY KEY,
  role_id INTEGER NOT NULL REFERENCES roles(id),
  username VARCHAR(50) NOT NULL UNIQUE,
  display_name VARCHAR(100) NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_role_id ON users (role_id);

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
