-- platform/white_label/migrations/0001_create_brand_settings.down.sql
DROP INDEX IF EXISTS idx_brand_custom_domain;
DROP TABLE IF EXISTS brand_settings;
