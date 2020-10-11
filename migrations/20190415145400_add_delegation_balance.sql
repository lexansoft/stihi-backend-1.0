CREATE TABLE delegation_balance (
  id          BIGSERIAL NOT NULL PRIMARY KEY,
  from_user_id   VARCHAR(255) NOT NULL,
  to_user_id     VARCHAR(255) NOT NULL,
  val_10x6    BIGINT NOT NULL,
  updated_at  TIMESTAMP NOT NULL
);

CREATE INDEX idx_delegation_balance_user1 ON delegation_balance (from_user_id, to_user_id);
CREATE INDEX idx_delegation_balance_user2 ON delegation_balance (to_user_id, from_user_id);
