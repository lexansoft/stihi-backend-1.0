CREATE TABLE users_info (
  user_id bigserial NOT NULL PRIMARY KEY,
  nickname VARCHAR(255),
  birthdate TIMESTAMP,
  biography TEXT
);
CREATE INDEX idx_users_info_nick ON users_info (nickname);