ALTER TABLE users_history
    ADD COLUMN time TIMESTAMP NOT NULL DEFAULT NOW();
CREATE INDEX idx_users_history_time ON users_history (time);
