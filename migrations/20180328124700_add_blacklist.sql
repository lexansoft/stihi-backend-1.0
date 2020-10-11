CREATE TABLE blacklist (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    ignore_author VARCHAR(255)
);
CREATE UNIQUE INDEX idx_blacklist_user_id_ignore_author ON blacklist (user_id, ignore_author);

CREATE UNIQUE INDEX idx_follows_user_id_subscribed_for ON follows (user_id, subscribed_for);
DROP INDEX idx_follows_user_id;

ALTER TABLE content
    ADD COLUMN votes_count_positive INT DEFAULT 0,
    ADD COLUMN votes_count_negative INT DEFAULT 0;
