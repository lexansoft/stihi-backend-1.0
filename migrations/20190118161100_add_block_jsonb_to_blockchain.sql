ALTER TABLE blockchain
  ADD COLUMN block_json JSONB;

UPDATE blockchain SET block_json = block::jsonb;
