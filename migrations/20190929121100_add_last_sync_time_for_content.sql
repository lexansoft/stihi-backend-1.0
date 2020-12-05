ALTER TABLE content
    ADD COLUMN last_sync_time TIMESTAMP;

ALTER TABLE users
    ADD COLUMN last_sync_time TIMESTAMP;
