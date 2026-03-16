\c telemetry_db;

CREATE TABLE IF NOT EXISTS telemetry_records (
    id         SERIAL PRIMARY KEY,
    device_id  INTEGER     NOT NULL,
    value      FLOAT       NOT NULL,
    unit       VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telemetry_device_id ON telemetry_records(device_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_created_at ON telemetry_records(created_at DESC);
