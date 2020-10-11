ALTER TABLE articles
    ADD COLUMN mat BOOLEAN DEFAULT 'f';
CREATE INDEX idx_articles_author_time_not_mat ON articles (author, "time") WHERE NOT mat;
CREATE INDEX idx_articles_author_time_desc_not_mat ON articles (author, "time" DESC) WHERE NOT mat;
UPDATE articles SET mat = 't' WHERE id IN (SELECT DISTINCT content_id FROM content_tags WHERE tag IN ('nsfw','ru--mat'));
