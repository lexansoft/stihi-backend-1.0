ALTER TABLE content
    ADD column val_cyber_10x6 bigint NOT NULL DEFAULT 0,
    DROP column val_gold_10x6;

ALTER TABLE users
    ADD column val_cyber_10x6 bigint NOT NULL DEFAULT 0,
    DROP column val_gold_10x6;

ALTER TABLE users_history
    ADD COLUMN val_cyber_change_10x6 bigint NOT NULL DEFAULT 0,
    DROP COLUMN val_gold_change_10x6;
