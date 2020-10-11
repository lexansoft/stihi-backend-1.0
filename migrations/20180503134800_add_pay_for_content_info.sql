ALTER TABLE content
    ADD COLUMN val_gold BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN val_golos BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN val_power BIGINT NOT NULL DEFAULT 0;

ALTER TABLE users_history
    ADD COLUMN content_id BIGINT,
    ADD COLUMN operation_type VARCHAR(128),
    ADD COLUMN block_num BIGINT,
    ADD COLUMN from_user_id BIGINT;
