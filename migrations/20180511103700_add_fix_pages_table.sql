CREATE TABLE fix_pages (
    code VARCHAR(64) NOT NULL PRIMARY KEY,
    html TEXT,
    admin_name VARCHAR(32),
    time TIMESTAMP
);
