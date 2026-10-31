CREATE DATABASE telemetry;

\c telemetry;

CREATE TABLE IF NOT EXISTS telemetry_data (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(128) UNIQUE,
    device_id VARCHAR(64) NOT NULL,
    metric VARCHAR(50) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(20) NOT NULL DEFAULT '',
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_telemetry_device_recorded
    ON telemetry_data(device_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_telemetry_metric
    ON telemetry_data(metric);
