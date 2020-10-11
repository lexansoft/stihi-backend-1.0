DROP INDEX idx_articles_meta_id;
CREATE UNIQUE INDEX idx_articles_uniq_meta_id ON articles (author, permlink);

DROP INDEX idx_comments_meta_id;
CREATE UNIQUE INDEX idx_comments_uniq_meta_id ON comments (author, permlink);
