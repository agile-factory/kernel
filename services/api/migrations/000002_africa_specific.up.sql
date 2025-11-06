-- ============================================================================
-- AFRICA-SPECIFIC ENHANCEMENTS
-- Migration 000002: Paiement mobile, KYC, modération, pricing dynamique
-- ============================================================================

-- ============================================================================
-- PAYMENT PROVIDERS (Mobile Money)
-- ============================================================================
CREATE TYPE payment_provider AS ENUM ('wave', 'orange_money', 'free_money', 'card');

-- Extend payments table with Africa-specific fields
ALTER TABLE payments
ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20),
ADD COLUMN IF NOT EXISTS provider_reference VARCHAR(255),
ADD COLUMN IF NOT EXISTS reconciliation_status VARCHAR(20) DEFAULT 'pending'
    CHECK (reconciliation_status IN ('pending', 'reconciled', 'disputed'));

CREATE INDEX idx_payments_phone ON payments(phone_number) WHERE phone_number IS NOT NULL;
CREATE INDEX idx_payments_reconciliation ON payments(reconciliation_status)
    WHERE reconciliation_status = 'pending';

-- ============================================================================
-- PAYMENT WEBHOOKS (Idempotency Layer)
-- ============================================================================
CREATE TABLE payment_webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider payment_provider NOT NULL,
    transaction_id VARCHAR(255) NOT NULL,
    payload_hash VARCHAR(64) NOT NULL, -- SHA256 of raw payload
    raw_payload JSONB NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    processing_error TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Prevent duplicate processing
CREATE UNIQUE INDEX idx_webhooks_transaction_unique
ON payment_webhooks(provider, transaction_id);

CREATE UNIQUE INDEX idx_webhooks_payload_unique
ON payment_webhooks(payload_hash);

CREATE INDEX idx_webhooks_unprocessed
ON payment_webhooks(created_at)
WHERE processed_at IS NULL;

-- ============================================================================
-- KYC DOCUMENTS
-- ============================================================================
CREATE TYPE kyc_document_type AS ENUM ('cni', 'passport', 'selfie', 'proof_address');
CREATE TYPE kyc_status AS ENUM ('pending', 'processing', 'approved', 'rejected', 'expired');

CREATE TABLE kyc_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_type kyc_document_type NOT NULL,
    s3_url VARCHAR(500) NOT NULL,

    -- OCR extracted data
    ocr_data JSONB,
    -- Example: {"cni_number": "1234567890123", "birth_date": "1990-01-01", "full_name": "..."}

    -- Verification
    verification_score DECIMAL(3, 2), -- 0.00 to 1.00
    verification_method VARCHAR(50), -- 'ocr', 'manual', 'face_match'

    status kyc_status NOT NULL DEFAULT 'pending',

    -- Manual review
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,

    expires_at TIMESTAMP WITH TIME ZONE, -- CNI/Passport expiry
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_kyc_user_id ON kyc_documents(user_id);
CREATE INDEX idx_kyc_status ON kyc_documents(status) WHERE status = 'pending';
CREATE INDEX idx_kyc_expires ON kyc_documents(expires_at) WHERE expires_at IS NOT NULL;

-- User KYC status summary
ALTER TABLE users
ADD COLUMN IF NOT EXISTS kyc_status kyc_status DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS kyc_completed_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS can_list_properties BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_users_kyc_status ON users(kyc_status);

-- ============================================================================
-- PHOTO MODERATION
-- ============================================================================
CREATE TYPE moderation_status AS ENUM ('pending', 'approved', 'rejected', 'flagged');

CREATE TABLE listing_photos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    s3_url VARCHAR(500) NOT NULL,
    s3_key VARCHAR(500) NOT NULL,

    -- Image metadata
    original_size_bytes INTEGER,
    optimized_size_bytes INTEGER,
    width INTEGER,
    height INTEGER,
    format VARCHAR(10), -- 'webp', 'jpg'

    -- Moderation
    moderation_status moderation_status DEFAULT 'pending',
    moderation_labels JSONB, -- AWS Rekognition labels
    moderation_confidence DECIMAL(5, 2), -- 0.00 to 100.00
    moderated_at TIMESTAMP WITH TIME ZONE,

    -- Manual review
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,

    display_order INTEGER DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_photos_listing ON listing_photos(listing_id, display_order);
CREATE INDEX idx_photos_moderation ON listing_photos(moderation_status)
    WHERE moderation_status = 'pending';

-- Update listings table to remove old images JSONB
ALTER TABLE listings DROP COLUMN IF EXISTS images;

-- ============================================================================
-- DYNAMIC PRICING
-- ============================================================================
CREATE TABLE pricing_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,

    -- Base price
    base_price_per_day DECIMAL(10, 2) NOT NULL,

    -- Multipliers
    weekend_multiplier DECIMAL(3, 2) DEFAULT 1.00, -- 1.20 = +20%
    holiday_multiplier DECIMAL(3, 2) DEFAULT 1.00,
    peak_season_multiplier DECIMAL(3, 2) DEFAULT 1.00,

    -- Occupancy-based pricing
    low_occupancy_discount DECIMAL(3, 2) DEFAULT 1.00, -- 0.90 = -10%
    high_occupancy_premium DECIMAL(3, 2) DEFAULT 1.00, -- 1.10 = +10%

    -- Early booking discount
    early_booking_days INTEGER DEFAULT 30,
    early_booking_discount DECIMAL(3, 2) DEFAULT 1.00,

    -- Last minute discount
    last_minute_days INTEGER DEFAULT 3,
    last_minute_discount DECIMAL(3, 2) DEFAULT 0.80, -- -20%

    -- Min/Max price limits
    min_price DECIMAL(10, 2) NOT NULL,
    max_price DECIMAL(10, 2) NOT NULL,

    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_pricing_rules_listing ON pricing_rules(listing_id);

CREATE TABLE calculated_prices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    calculated_price DECIMAL(10, 2) NOT NULL,
    factors JSONB, -- {"weekend": true, "occupancy": 0.75, "applied_multipliers": [1.2, 1.1]}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_calculated_prices_listing_date
ON calculated_prices(listing_id, date);

CREATE INDEX idx_calculated_prices_date ON calculated_prices(date)
WHERE date >= CURRENT_DATE;

-- ============================================================================
-- PAYOUTS (Reversement propriétaires)
-- ============================================================================
CREATE TYPE payout_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');

CREATE TABLE payouts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    landlord_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,

    -- Amount breakdown
    booking_amount DECIMAL(10, 2) NOT NULL,
    platform_fee DECIMAL(10, 2) NOT NULL,
    payout_amount DECIMAL(10, 2) NOT NULL,

    -- Payment details
    provider payment_provider NOT NULL,
    phone_number VARCHAR(20),
    bank_account VARCHAR(100),

    status payout_status DEFAULT 'pending',

    -- Processing
    scheduled_for TIMESTAMP WITH TIME ZONE NOT NULL, -- checkout + 24h
    processed_at TIMESTAMP WITH TIME ZONE,
    provider_transaction_id VARCHAR(255),
    provider_response JSONB,

    -- Retry logic
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    last_error TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_payouts_landlord ON payouts(landlord_id);
CREATE INDEX idx_payouts_status ON payouts(status) WHERE status IN ('pending', 'failed');
CREATE INDEX idx_payouts_scheduled ON payouts(scheduled_for)
WHERE scheduled_for <= NOW() AND status = 'pending';

-- ============================================================================
-- CHECK-IN QR CODES
-- ============================================================================
CREATE TABLE checkin_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,

    -- QR code data (JWT signed)
    qr_code_token TEXT NOT NULL, -- JWT containing booking info
    qr_code_image_url VARCHAR(500), -- S3 URL of QR code image

    -- Validity
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Usage tracking
    checked_in_at TIMESTAMP WITH TIME ZONE,
    checked_in_by UUID REFERENCES users(id),
    device_info JSONB,

    -- Revocation
    revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_reason TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_checkin_codes_booking ON checkin_codes(booking_id);
CREATE INDEX idx_checkin_codes_valid ON checkin_codes(valid_from, valid_until)
WHERE NOT revoked;

-- ============================================================================
-- TAXES (Per Region)
-- ============================================================================
CREATE TABLE regional_taxes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country VARCHAR(50) NOT NULL,
    city VARCHAR(100) NOT NULL,

    -- Tax rates (as percentages)
    tourist_tax_rate DECIMAL(5, 2) DEFAULT 0.00, -- % of booking amount
    vat_rate DECIMAL(5, 2) DEFAULT 18.00, -- Senegal default

    -- Other fees
    platform_fee_rate DECIMAL(5, 2) DEFAULT 10.00, -- % charged to landlord

    effective_from DATE NOT NULL,
    effective_until DATE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_regional_taxes_location ON regional_taxes(country, city, effective_from);

-- Insert default Senegalese taxes
INSERT INTO regional_taxes (country, city, tourist_tax_rate, vat_rate, platform_fee_rate, effective_from)
VALUES
    ('Senegal', 'Dakar', 5.00, 18.00, 10.00, '2024-01-01'),
    ('Senegal', 'Saint-Louis', 3.00, 18.00, 10.00, '2024-01-01'),
    ('Senegal', 'Saly', 5.00, 18.00, 10.00, '2024-01-01');

-- ============================================================================
-- FRAUD PREVENTION
-- ============================================================================
CREATE TABLE fraud_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    event_type VARCHAR(50) NOT NULL, -- 'payment', 'booking', 'login'
    risk_score DECIMAL(3, 2) NOT NULL, -- 0.00 to 1.00

    -- Signals
    device_fingerprint VARCHAR(255),
    ip_address INET,
    user_agent TEXT,

    -- Velocity checks
    events_last_hour INTEGER,
    events_last_day INTEGER,

    -- Decision
    action VARCHAR(20) NOT NULL, -- 'allow', 'challenge', 'block'
    reason TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_fraud_checks_user ON fraud_checks(user_id, created_at DESC);
CREATE INDEX idx_fraud_checks_score ON fraud_checks(risk_score DESC)
WHERE action = 'block';

-- Velocity tracking (Redis preferred, but fallback to PG)
CREATE TABLE user_velocity (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    payments_last_hour INTEGER DEFAULT 0,
    bookings_last_hour INTEGER DEFAULT 0,
    last_payment_at TIMESTAMP WITH TIME ZONE,
    last_booking_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- OFFLINE SYNC (PWA support)
-- ============================================================================
CREATE TABLE sync_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    action VARCHAR(50) NOT NULL, -- 'create_booking', 'add_favorite'
    payload JSONB NOT NULL,
    device_id VARCHAR(255),

    synced BOOLEAN DEFAULT FALSE,
    synced_at TIMESTAMP WITH TIME ZONE,
    sync_error TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_sync_queue_user ON sync_queue(user_id) WHERE NOT synced;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update payout after booking completion
CREATE OR REPLACE FUNCTION create_payout_after_checkout()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'completed' AND OLD.status != 'completed' THEN
        INSERT INTO payouts (
            landlord_id,
            booking_id,
            booking_amount,
            platform_fee,
            payout_amount,
            provider,
            phone_number,
            scheduled_for
        )
        SELECT
            l.landlord_id,
            NEW.id,
            NEW.total_amount,
            NEW.total_amount * 0.10, -- 10% platform fee
            NEW.total_amount * 0.90,
            'wave', -- Default, user can change
            u.phone,
            NEW.check_out + INTERVAL '24 hours'
        FROM listings l
        JOIN users u ON u.id = l.landlord_id
        WHERE l.id = NEW.listing_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_payout
AFTER UPDATE ON bookings
FOR EACH ROW
EXECUTE FUNCTION create_payout_after_checkout();

-- Update KYC status on user when all documents approved
CREATE OR REPLACE FUNCTION update_user_kyc_status()
RETURNS TRIGGER AS $$
DECLARE
    required_docs INTEGER := 2; -- CNI + Selfie
    approved_docs INTEGER;
BEGIN
    -- Count approved documents
    SELECT COUNT(*) INTO approved_docs
    FROM kyc_documents
    WHERE user_id = NEW.user_id
    AND status = 'approved'
    AND document_type IN ('cni', 'passport', 'selfie');

    -- Update user if all required docs approved
    IF approved_docs >= required_docs THEN
        UPDATE users
        SET kyc_status = 'approved',
            kyc_completed_at = NOW(),
            can_list_properties = TRUE
        WHERE id = NEW.user_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_kyc
AFTER UPDATE ON kyc_documents
FOR EACH ROW
WHEN (NEW.status = 'approved' AND OLD.status != 'approved')
EXECUTE FUNCTION update_user_kyc_status();
