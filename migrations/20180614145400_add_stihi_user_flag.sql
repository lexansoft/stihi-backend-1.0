ALTER TABLE users
    ADD COLUMN stihi_user BOOLEAN default 'f';
CREATE INDEX idx_users_parial_stihi_user_id ON users (id) WHERE stihi_user;
