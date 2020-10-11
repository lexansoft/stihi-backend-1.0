CREATE TABLE users_names (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    creator VARCHAR(128),
    name VARCHAR(255)
);

CREATE INDEX idx_users_names_user_id_creator ON users_names (user_id, creator);
