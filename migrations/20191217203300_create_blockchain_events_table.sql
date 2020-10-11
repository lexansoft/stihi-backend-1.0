CREATE TABLE blockchain_events (
    id          BIGSERIAL NOT NULL PRIMARY KEY,
    event       TEXT,
    event_json  JSONB,
    checksum    VARCHAR(128),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    log_time    BIGINT NOT NULL,
    log_offset  BIGINT NOT NULL
);

CREATE UNIQUE INDEX idx_blockchain_events_checksum ON blockchain_events (checksum);
CREATE INDEX idx_blockchain_events_offset ON blockchain_events (log_time, log_offset);
CREATE INDEX idx_blockchain_events_json ON blockchain_events USING GIN (event_json jsonb_path_ops);
