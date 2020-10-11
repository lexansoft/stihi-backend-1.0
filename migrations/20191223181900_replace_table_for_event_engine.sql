DROP TABLE IF EXISTS blockchain_events;

CREATE TABLE IF NOT EXISTS cyberway_actions (
    id          BIGSERIAL NOT NULL PRIMARY KEY,
    trx_id      VARCHAR(65),
    block_num   BIGINT,
    block_time  TIMESTAMP,
    action_idx  INT,
    action      JSONB
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cyberway_actions_trx_id ON cyberway_actions (trx_id);
CREATE INDEX IF NOT EXISTS idx_cyberway_actions_json ON cyberway_actions USING GIN (action jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_cyberway_actions_action_idx ON cyberway_actions (block_num, action_idx);
