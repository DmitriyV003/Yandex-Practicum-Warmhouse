CREATE DATABASE telemetry_db;
\c telemetry_db;

CREATE TABLE IF NOT EXISTS telemetry_records (
    id         SERIAL PRIMARY KEY,
    sensor_id  INTEGER     NOT NULL,
    value      FLOAT       NOT NULL,
    unit       VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telemetry_sensor_id ON telemetry_records(sensor_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_created_at ON telemetry_records(created_at DESC);
