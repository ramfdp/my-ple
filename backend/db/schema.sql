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

-- Users table (columns: id, name, email, role (FK), username, job_title, salary, residence, phone_number, employment_status, avatar_url, is_active, created_at, updated_at)
CREATE TABLE users (
  id UUID PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  role_id INTEGER NOT NULL REFERENCES roles(id),
  username VARCHAR(50) NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  job_title VARCHAR(100) NOT NULL DEFAULT '',
  salary BIGINT NOT NULL DEFAULT 0,
  residence VARCHAR(255) NOT NULL DEFAULT '',
  phone_number VARCHAR(50) NOT NULL DEFAULT '',
  employment_status VARCHAR(50) NOT NULL DEFAULT 'Permanent Employee',
  avatar_url TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_role_id ON users (role_id);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_username ON users (username);

-- Migration helpers for existing databases
ALTER TABLE users ADD COLUMN IF NOT EXISTS job_title VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS salary BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS residence VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS employment_status VARCHAR(50) NOT NULL DEFAULT 'Permanent Employee';
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

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
