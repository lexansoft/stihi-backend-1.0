CREATE TABLE users_keys (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    key_type VARCHAR(32),
    key TEXT
);

WITH all_users_keys AS (
    SELECT id as user_id, owner_key as key, 'owner' as key_type
    FROM users
    UNION
    SELECT id as user_id, active_key as key, 'active' as key_type
    FROM users
    UNION
    SELECT id as user_id, posting_key as key, 'posting' as key_type
    FROM users
    UNION
    SELECT id as user_id, memo_key as key, 'memo' as key_type
    FROM users
    WHERE memo_key IS NOT NULL AND memo_key <> ''
)
INSERT INTO users_keys (user_id, key, key_type) SELECT * FROM all_users_keys;

CREATE INDEX idx_users_keys_users_id_type ON users_keys (user_id, key_type);

ALTER TABLE users
    DROP COLUMN owner_key,
    DROP COLUMN active_key,
    DROP COLUMN memo_key,
    DROP COLUMN posting_key;
