ALTER TABLE users
  ADD COLUMN account_info JSONB;  -- JSON аккаунта прямо из get_account блокчейна

CREATE INDEX idx_users_account_info
  ON users USING GIN (account_info)
  WHERE stihi_user AND (account_info->'withdrawn')::TEXT::BIGINT > 0;
