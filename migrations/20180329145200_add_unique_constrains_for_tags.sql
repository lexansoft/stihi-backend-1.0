DROP INDEX idx_content_tags_content;
CREATE UNIQUE INDEX idx_content_tags_unique_content_tag ON content_tags (content_id, tag);

ALTER TABLE content
  ADD COLUMN level INTEGER NOT NULL DEFAULT 0;
