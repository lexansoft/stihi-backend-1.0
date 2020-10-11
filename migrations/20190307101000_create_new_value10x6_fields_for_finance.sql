ALTER TABLE content
  ADD column val_gold_10x6 bigint NOT NULL DEFAULT 0,
  ADD column val_golos_10x6 bigint NOT NULL DEFAULT 0;

UPDATE content
  SET
      val_gold_10x6 = val_gold * 1000,
      val_golos_10x6 = val_golos * 1000;

CREATE INDEX idx_content_val_gold_10x6 ON articles (val_gold_10x6 DESC);


ALTER TABLE delegate_return_cron
  ADD column val_10x6 bigint NOT NULL DEFAULT 0;

UPDATE delegate_return_cron
  SET val_10x6 = val1000 * 1000;


ALTER TABLE users
  ADD column val_gold_10x6 bigint NOT NULL DEFAULT 0,
  ADD column val_golos_10x6 bigint NOT NULL DEFAULT 0,
  ADD column val_power_10x6 bigint NOT NULL DEFAULT 0,
  ADD column val_delegated_10x6 bigint NOT NULL DEFAULT 0,
  ADD column val_received_10x6 bigint NOT NULL DEFAULT 0;

UPDATE users
  SET
    val_gold_10x6 = val_gold * 1000,
    val_golos_10x6 = val_golos * 1000,
    val_power_10x6 = val_power * 1000;

ALTER TABLE users_history
  ADD COLUMN val_gold_change_10x6 bigint NOT NULL DEFAULT 0,
  ADD COLUMN val_golos_change_10x6 bigint NOT NULL DEFAULT 0,
  ADD COLUMN val_power_change_10x6 bigint NOT NULL DEFAULT 0,
  ADD COLUMN val_delegated_change_10x6 bigint NOT NULL DEFAULT 0,
  ADD COLUMN val_received_change_10x6 bigint NOT NULL DEFAULT 0;

UPDATE users_history
  SET
    val_gold_change_10x6 = val_gold_change * 1000,
    val_golos_change_10x6 = val_golos_change * 1000,
    val_power_change_10x6 = val_power_change * 1000;

CREATE UNIQUE INDEX idx_users_history_uniq_constrain_10x6 ON users_history (user_id, from_user_id, content_id, operation_type, block_num, val_gold_change_10x6, val_golos_change_10x6, val_power_change_10x6);
