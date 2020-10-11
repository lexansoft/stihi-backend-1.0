DROP INDEX IF EXISTS idx_cyberway_actions_trx_id;
DROP INDEX IF EXISTS idx_cyberway_actions_action_idx;

CREATE UNIQUE INDEX idx_cyberway_actions_action_idx ON cyberway_actions (block_num, trx_id, action_idx);
