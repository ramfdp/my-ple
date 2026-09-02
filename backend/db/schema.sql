-- 1. Roles table
CREATE TABLE IF NOT EXISTS roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Predefined roles seed
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

-- 3. Users table (includes all core authentication and employee profile fields)
CREATE TABLE IF NOT EXISTS users (
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

CREATE INDEX IF NOT EXISTS idx_users_role_id ON users (role_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);

-- 4. Safe column migration for existing database instances
ALTER TABLE users ADD COLUMN IF NOT EXISTS job_title VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS salary BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS residence VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS employment_status VARCHAR(50) NOT NULL DEFAULT 'Permanent Employee';
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- 5. Report sequences table
CREATE TABLE IF NOT EXISTS report_sequences (
  primary_division VARCHAR(100) NOT NULL,
  work_division VARCHAR(100) NOT NULL,
  last_number INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (primary_division, work_division)
);

-- 6. Reports table
CREATE TABLE IF NOT EXISTS reports (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  primary_division VARCHAR(100) NOT NULL,
  work_division VARCHAR(100) NOT NULL,
	 report_role VARCHAR(50) NOT NULL DEFAULT '',
	 work VARCHAR(255) NOT NULL DEFAULT '',
  number INTEGER NOT NULL,
  description VARCHAR(500) NOT NULL,
  completed_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (primary_division, work_division, number)
);

CREATE INDEX IF NOT EXISTS reports_by_division ON reports (primary_division, work_division, number);
ALTER TABLE reports ADD COLUMN IF NOT EXISTS report_role VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE reports ADD COLUMN IF NOT EXISTS work VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE report_sequences DROP COLUMN IF EXISTS report_role;
ALTER TABLE report_sequences DROP COLUMN IF EXISTS work;
