CREATE TABLE endpoints (
    id               BIGSERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    url              TEXT NOT NULL,
    interval_seconds INT NOT NULL DEFAULT 30,
    expected_status  INT NOT NULL DEFAULT 200,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE checks (
    id          BIGSERIAL PRIMARY KEY,
    endpoint_id BIGINT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    succeeded   BOOLEAN NOT NULL,
    status_code INT,
    latency_ms  INT,
    error_msg   TEXT
);

CREATE INDEX idx_checks_endpoint_id ON checks(endpoint_id);
CREATE INDEX idx_checks_checked_at  ON checks(checked_at);