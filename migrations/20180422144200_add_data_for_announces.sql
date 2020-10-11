ALTER TABLE users_info
    ADD COLUMN sex CHAR(1),
    ADD COLUMN place VARCHAR(255),
    ADD COLUMN web_site VARCHAR(255),
    ADD COLUMN avatar_image VARCHAR(255),
    ADD COLUMN background_image VARCHAR(255),
    ADD COLUMN pvt_posts_show_mode CHAR(1);

CREATE TABLE announces_pages (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    code VARCHAR(16) NOT NULL,
    name VARCHAR(255) NOT NULL,
    price BIGINT NOT NULL DEFAULT 0
);

INSERT INTO announces_pages (code, name, price)
    VALUES
      ('main',    'на главной странице',  50000),
      ('follow',  'в ленте подписок',     30000),
      ('new',     'в ленте новое',        30000),
      ('actual',  'в ленте актуальное',   30000),
      ('popular', 'в ленте лидеров',      30000),
      ('comments', 'в ленте комментариев', 5000);

CREATE TABLE announces (
    id BIGSERIAL NOT NULL PRIMARY KEY,
    page_code VARCHAR(16) NOT NULL,
    content_id BIGINT NOT NULL,
    place_time TIMESTAMP DEFAULT NOW(),
    payer VARCHAR(255) NOT NULL,
    payer_id BIGINT NOT NULL,
    pay_data TEXT
);
CREATE INDEX idx_announces_code_time ON announces (page_code, place_time DESC);
