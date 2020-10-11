ALTER TABLE content
    ADD COLUMN nodeos_id BIGINT;

CREATE INDEX idx_content_nodeos_id ON content (nodeos_id);
