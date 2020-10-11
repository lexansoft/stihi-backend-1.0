CREATE TABLE delegate_return_cron (
  id          BIGSERIAL NOT NULL PRIMARY KEY,
  user_name   VARCHAR(255) NOT NULL,
  return_to   VARCHAR(255) NOT NULL,
  val1000     BIGINT NOT NULL,
  created_at  TIMESTAMP NOT NULL,
  return_at   TIMESTAMP NOT NULL,
  is_return   BOOLEAN NOT NULL DEFAULT 'f'
);

CREATE INDEX idx_delegate_return_cron_return_at ON delegate_return_cron (return_at) WHERE NOT is_return;
