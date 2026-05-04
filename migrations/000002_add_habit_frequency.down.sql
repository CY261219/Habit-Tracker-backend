-- Down Migration
ALTER TABLE habits DROP COLUMN frequency_type;
ALTER TABLE habits DROP COLUMN frequency_config;
