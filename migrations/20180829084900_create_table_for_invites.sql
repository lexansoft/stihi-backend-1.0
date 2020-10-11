CREATE TABLE invites (
  id      BIGSERIAL NOT NULL PRIMARY KEY,
  author  varchar(255) NOT NULL,
  place_time TIMESTAMP DEFAULT NOW(),
  payer VARCHAR(255) NOT NULL,
  payer_id BIGINT NOT NULL,
  pay_data TEXT
);
CREATE INDEX idx_invites_time ON invites (place_time DESC);
