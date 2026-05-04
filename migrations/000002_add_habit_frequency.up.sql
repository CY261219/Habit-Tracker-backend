-- Up Migration
ALTER TABLE habits ADD COLUMN frequency_type VARCHAR(20) DEFAULT 'DAILY';
ALTER TABLE habits ADD COLUMN frequency_config JSONB DEFAULT '[]';

-- Down Migration
-- ALTER TABLE habits DROP COLUMN frequency_type;
-- ALTER TABLE habits DROP COLUMN frequency_config;
