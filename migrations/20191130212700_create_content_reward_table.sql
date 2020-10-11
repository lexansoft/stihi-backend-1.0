CREATE TABLE content_reward (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    content_id BIGINT NOT NULL,
    type_name VARCHAR(32) NOT NULL,
    value_10x6 BIGINT NOT NULL
);

CREATE UNIQUE INDEX idx_content_reward_content_type ON content_reward (content_id, type_name);
