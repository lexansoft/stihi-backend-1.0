-- Поле confirmed:
-- 't' - если данные подтверждены блокчейном
-- 'f' - если данные сгенерированы на сайте, но еще не были получены из блока blockchain

CREATE TABLE content (
  id bigserial NOT NULL PRIMARY KEY,
  parent_author VARCHAR(255),
  parent_permlink VARCHAR(255),
  author VARCHAR(255) NOT NULL,
  permlink VARCHAR(1024) NOT NULL,
  title VARCHAR(255),
  body TEXT NOT NULL,
  time TIMESTAMP,
  confirmed BOOLEAN default 't'
);

CREATE TABLE articles (
  CHECK ( parent_author IS NULL OR parent_author = '' )
) INHERITS (content) ;

CREATE TABLE comments (
  CHECK ( parent_author IS NOT NULL )
) INHERITS (content) ;

CREATE INDEX idx_articles_meta_id ON articles (author, permlink);
CREATE INDEX idx_comments_meta_id ON comments (author, permlink);
CREATE INDEX idx_comments_parent_meta_id ON comments (parent_author, parent_permlink);

CREATE TABLE content_tags (
  id bigserial NOT NULL PRIMARY KEY,
  content_id bigint NOT NULL,
  tag VARCHAR(255) NOT NULL,
  time TIMESTAMP,
  confirmed BOOLEAN default 't'
);

CREATE INDEX idx_content_tags_content ON content_tags (content_id);
CREATE INDEX idx_content_tags_tag ON content_tags (tag);

CREATE TABLE content_votes (
  id bigserial NOT NULL PRIMARY KEY,
  content_id bigint NOT NULL,
  voter VARCHAR(255) NOT NULL,
  weight int DEFAULT 0,
  time TIMESTAMP,
  confirmed BOOLEAN default 't'
);

CREATE INDEX idx_content_votes_content ON content_votes (content_id);
CREATE INDEX idx_content_votes_voter ON content_votes (voter);

CREATE TABLE users (
  id bigserial NOT NULL PRIMARY KEY,
  name varchar(255) NOT NULL,
  val_gold bigint default 0,
  val_golos bigint default 0,
  val_power bigint default 0
);

CREATE TABLE users_history (
  id bigserial NOT NULL PRIMARY KEY,
  user_id bigint NOT NULL,
  virtual boolean DEFAULT 'f',
  operation TEXT NOT NULL,
  val_gold_change bigint DEFAULT 0,
  val_golos_change bigint DEFAULT 0,
  val_power_change bigint DEFAULT 0
);

CREATE INDEX idx_users_history_user_id ON users_history (user_id);
