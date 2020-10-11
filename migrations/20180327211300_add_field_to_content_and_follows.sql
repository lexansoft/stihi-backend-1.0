ALTER TABLE content
    ADD COLUMN votes_count        INTEGER DEFAULT 0,
    ADD COLUMN votes_sum_positive BIGINT DEFAULT 0,
    ADD COLUMN votes_sum_negative BIGINT DEFAULT 0;

ALTER TABLE articles
    ADD COLUMN image              varchar(1024),
    ADD COLUMN last_comment_time  TIMESTAMP,
    ADD COLUMN comments_count     INTEGER DEFAULT 0;
CREATE INDEX idx_articles_last_comment_time ON articles (last_comment_time);
CREATE INDEX idx_articles_comments_count ON articles (comments_count);

CREATE TABLE follows (
    id              BIGSERIAL NOT NULL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    subscribed_for  VARCHAR(255) NOT NULL
);
CREATE INDEX idx_follows_user_id ON follows (user_id);
CREATE INDEX idx_follows_subscribed ON follows (subscribed_for);
