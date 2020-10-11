CREATE UNIQUE INDEX idx_users_history_uniq_constrain
    ON users_history (user_id, from_user_id, content_id, operation_type, block_num, val_gold_change, val_golos_change, val_power_change);
