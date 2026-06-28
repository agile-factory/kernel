-- Rollback: 0001_create_billing_tables
DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS tenant_settings;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
