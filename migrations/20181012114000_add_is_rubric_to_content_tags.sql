ALTER TABLE content_tags
    ADD COLUMN is_rubric BOOLEAN DEFAULT 'f';
CREATE INDEX idx_content_tags_only_rubrics ON content_tags (tag) WHERE is_rubric;

UPDATE content_tags SET is_rubric = 't' WHERE id IN ( SELECT id
                                                      FROM (SELECT id, content_id, tag, row_number()
                                                            OVER(PARTITION BY content_id ORDER BY content_id, id) rn
                                                            FROM content_tags
                                                            WHERE tag != 'stihi-io'
                                                            ORDER BY content_id, id)t
                                                      WHERE t.rn = 1);
