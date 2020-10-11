ALTER TABLE content
  ADD column val_power_10x6 bigint NOT NULL DEFAULT 0;

UPDATE content
  SET
      val_power_10x6 = val_power * 1000;
