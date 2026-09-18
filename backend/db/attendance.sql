CREATE TABLE IF NOT EXISTS attendance (
 id UUID PRIMARY KEY,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 attendance_date DATE NOT NULL,
 mode TEXT NOT NULL CHECK (mode IN ('WFO', 'WFA')),
 clock_in TIMESTAMPTZ NOT NULL,
 clock_out TIMESTAMPTZ CHECK (clock_out >= clock_in),
 latitude DOUBLE PRECISION,
 longitude DOUBLE PRECISION,
 UNIQUE (user_id, attendance_date),
 CHECK ((mode = 'WFO' AND latitude IS NULL AND longitude IS NULL) OR
        (mode = 'WFA' AND latitude IS NOT NULL AND longitude IS NOT NULL
         AND latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180))
);
CREATE UNIQUE INDEX IF NOT EXISTS attendance_one_open_per_user ON attendance(user_id) WHERE clock_out IS NULL;
