ALTER TABLE content
    ADD COLUMN last_sync_time TIMESTAMP;

ALTER TABLE users
    ADD COLUMN val_cyber_10x6 BIGINT DEFAULT 0,
    ADD COLUMN last_sync_time TIMESTAMP;
