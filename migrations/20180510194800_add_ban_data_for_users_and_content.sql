ALTER TABLE content
    ADD COLUMN ban BOOLEAN DEFAULT 'f';

ALTER TABLE users
    ADD COLUMN ban BOOLEAN DEFAULT 'f';

CREATE TABLE ban_history (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    ban_object_type VARCHAR(16) NOT NULL,
    ban_object_id BIGINT NOT NULL,
    unban BOOLEAN NOT NULL DEFAULT 'f',
    admin_name VARCHAR(32),
    admin_description TEXT,
    time TIMESTAMP NOT NULL
);
CREATE INDEX idx_ban_history_object ON ban_history (ban_object_type, ban_object_id, time);
