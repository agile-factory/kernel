-- Rollback Africa-specific enhancements

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_user_kyc ON kyc_documents;
DROP TRIGGER IF EXISTS trigger_create_payout ON bookings;

-- Drop functions
DROP FUNCTION IF EXISTS update_user_kyc_status();
DROP FUNCTION IF EXISTS create_payout_after_checkout();

-- Drop tables (in reverse dependency order)
DROP TABLE IF EXISTS sync_queue;
DROP TABLE IF EXISTS user_velocity;
DROP TABLE IF EXISTS fraud_checks;
DROP TABLE IF EXISTS regional_taxes;
DROP TABLE IF EXISTS checkin_codes;
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS calculated_prices;
DROP TABLE IF EXISTS pricing_rules;
DROP TABLE IF EXISTS listing_photos;
DROP TABLE IF EXISTS kyc_documents;
DROP TABLE IF EXISTS payment_webhooks;

-- Remove added columns
ALTER TABLE users DROP COLUMN IF EXISTS can_list_properties;
ALTER TABLE users DROP COLUMN IF EXISTS kyc_completed_at;
ALTER TABLE users DROP COLUMN IF EXISTS kyc_status;

ALTER TABLE payments DROP COLUMN IF EXISTS reconciliation_status;
ALTER TABLE payments DROP COLUMN IF EXISTS provider_reference;
ALTER TABLE payments DROP COLUMN IF EXISTS phone_number;

-- Drop types
DROP TYPE IF EXISTS payout_status;
DROP TYPE IF EXISTS moderation_status;
DROP TYPE IF EXISTS kyc_status;
DROP TYPE IF EXISTS kyc_document_type;
DROP TYPE IF EXISTS payment_provider;
