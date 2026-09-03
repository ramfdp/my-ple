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

-- 7. Fixed Zoom resources and their user-visible operational information.
CREATE TABLE IF NOT EXISTS zoom_resources (
  id SMALLINT PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE,
  code VARCHAR(20) NOT NULL UNIQUE,
  description VARCHAR(1000) NOT NULL DEFAULT '',
  meeting_url TEXT NOT NULL DEFAULT '',
  meeting_id VARCHAR(100) NOT NULL DEFAULT '',
	 host_key VARCHAR(100) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE zoom_resources ADD COLUMN IF NOT EXISTS host_key VARCHAR(100) NOT NULL DEFAULT '';

INSERT INTO zoom_resources (id, name, code) VALUES
  (1, 'Zoom 1', 'NP'),
  (2, 'Zoom 2', 'MRP'),
  (3, 'Zoom 3', 'IB2')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, code = EXCLUDED.code;

-- 8. Existing bookings are assigned to Zoom 1 during the nullable migration step.
CREATE TABLE IF NOT EXISTS zoom_bookings (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  zoom_resource_id SMALLINT NOT NULL DEFAULT 1,
  title VARCHAR(200) NOT NULL,
  description VARCHAR(1000) NOT NULL DEFAULT '',
  -- Legacy per-booking URLs are retained to avoid deleting existing data.
  meeting_url TEXT NOT NULL DEFAULT '',
  start_at TIMESTAMPTZ NOT NULL,
  end_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (start_at < end_at)
);

ALTER TABLE zoom_bookings ADD COLUMN IF NOT EXISTS zoom_resource_id SMALLINT;
UPDATE zoom_bookings SET zoom_resource_id = 1 WHERE zoom_resource_id IS NULL;
ALTER TABLE zoom_bookings ALTER COLUMN zoom_resource_id SET DEFAULT 1;
ALTER TABLE zoom_bookings ALTER COLUMN zoom_resource_id SET NOT NULL;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'zoom_bookings_resource_fk' AND conrelid = 'zoom_bookings'::regclass) THEN
    ALTER TABLE zoom_bookings ADD CONSTRAINT zoom_bookings_resource_fk FOREIGN KEY (zoom_resource_id) REFERENCES zoom_resources(id);
  END IF;
END $$;

DROP INDEX IF EXISTS zoom_bookings_by_time;
CREATE INDEX IF NOT EXISTS zoom_bookings_by_time ON zoom_bookings (zoom_resource_id, start_at, end_at);
CREATE INDEX IF NOT EXISTS zoom_bookings_by_user ON zoom_bookings (user_id);
CREATE EXTENSION IF NOT EXISTS btree_gist;
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'zoom_bookings_no_overlap'
      AND conrelid = 'zoom_bookings'::regclass
      AND pg_get_constraintdef(oid) NOT LIKE '%zoom_resource_id%'
  ) THEN
    ALTER TABLE zoom_bookings DROP CONSTRAINT zoom_bookings_no_overlap;
  END IF;
END $$;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'zoom_bookings_no_overlap' AND conrelid = 'zoom_bookings'::regclass) THEN
    ALTER TABLE zoom_bookings ADD CONSTRAINT zoom_bookings_no_overlap
      EXCLUDE USING gist (zoom_resource_id WITH =, tstzrange(start_at, end_at, '[)') WITH &&);
  END IF;
END $$;
