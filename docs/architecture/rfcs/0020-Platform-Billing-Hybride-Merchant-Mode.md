# _RFC #0020:_ Platform `billing` — Hybride Stripe + Mobile Money + Merchant Mode

**date:** `2026 May 12`

> **Supersède :** [RFC-0012 — Platform Billing Abonnements](0012-Platform-Billing-Abonnements.md)

---

## **Status:** `ACCEPTED` *(Platform Mode prévu P1-2 ; Merchant Mode B2B2C en P1bis-8, cf. plan §6)*

---

## **Summary**

Ce RFC redéfinit la plateforme `billing` pour supporter une **stratégie hybride Stripe + Mobile Money** à deux niveaux :

| Niveau | Sens | Description |
|---|---|---|
| **Platform Mode** | Tenant → Plateforme | Le tenant paye son abonnement SaaS (Stripe international + Wave/Orange/Free Money CEDEAO) |
| **Merchant Mode** | End-customer → Tenant | Le tenant encaisse ses clients finaux via la plateforme (Stripe Connect + Mobile Money Marchand) — activable par tenant, pierre angulaire du profil B2B2C |

Décisions géographiques :
- **International** : Stripe (Checkout Session + webhooks)
- **CEDEAO** : Wave (Sénégal, Côte d'Ivoire) + Orange Money (CEDEAO 15 pays) + Free Money (Sénégal) via adapters individuels ou agrégateur

---

## **Motivation**

### Décision géographique validée

- **80–90% des end-customers en CEDEAO** utilisent Mobile Money, pas de carte bancaire. Forcer Stripe seul = perdre le marché B2B2C CEDEAO.
- **Stripe reste indispensable** pour les clients enterprise internationaux et pour Merchant Mode international (Stripe Connect).
- La stratégie hybride via le port `PaymentProcessor` (RFC-0012) est conservée et étendue.

### Insuffisances de RFC-0012

1. RFC-0012 restait orienté B2B : le Merchant Mode était esquissé mais l'entité `TenantCustomer` n'était pas alignée avec `platform/end_customers` (RFC-0033).
2. La stratégie de sélection du processor par tenant (`tenant_settings.payment_region`) n'était pas formalisée.
3. Le flux Merchant Mode Mobile Money (Wave Marchand, Orange Money Marchand) manquait de détails.

---

## **Proposal**

### 1. Port abstrait `PaymentProcessor` (Platform Mode)

```go
// platform/billing/domain/payment_processor.go
package billing

type PaymentProcessor interface {
    CreateCheckoutSession(ctx context.Context, tenant Tenant, plan Plan) (CheckoutResult, error)
    HandleWebhook(ctx context.Context, signature string, body []byte) (PaymentEvent, error)
    RecordPayment(ctx context.Context, tenant Tenant, ref string, amount int64, currency string) error
}

type CheckoutResult struct {
    RedirectURL string
    ExternalID  string  // Stripe Session ID, Wave Order ID, etc.
}

type PaymentEvent struct {
    Type     PaymentEventType  // PAYMENT_SUCCEEDED | PAYMENT_FAILED | SUBSCRIPTION_RENEWED
    TenantID string
    Amount   int64
    Currency string
    Ref      string
}
```

### 2. Adapters Platform Mode fournis

| Adapter | Marché | Mécanique |
|---|---|---|
| `StripeAdapter` | International (USD, EUR, GBP…) | Checkout Session + webhook `stripe-signature` |
| `WaveAdapter` | Sénégal, Côte d'Ivoire (XOF) | Wave Checkout Link + webhook |
| `OrangeMoneyAdapter` | CEDEAO 15 pays (XOF, XAF) | Orange Money API + callback |
| `FreeMoneyAdapter` | Sénégal (XOF) | Free Money Expresso API + callback |

Sélection du processor via `tenant_settings.payment_region` :
- `INTERNATIONAL` → `StripeAdapter`
- `CEDEAO` → `StripeAdapter` + `WaveAdapter` + `OrangeMoneyAdapter` + `FreeMoneyAdapter` (l'admin tenant choisit au checkout)

### 3. Port abstrait `MerchantProcessor` (Merchant Mode B2B2C)

```go
// platform/billing/domain/merchant_processor.go
package billing

type MerchantProcessor interface {
    OnboardMerchant(ctx context.Context, tenant Tenant) (MerchantAccount, error)
    CreateCustomerCheckout(ctx context.Context, tenant Tenant, ec EndCustomerRef, amount int64, currency string) (CheckoutResult, error)
    HandleMerchantWebhook(ctx context.Context, signature string, body []byte) (MerchantPaymentEvent, error)
    PayoutToMerchant(ctx context.Context, tenant Tenant, amount int64, currency string) (Payout, error)
}

type MerchantAccount struct {
    ExternalID string  // Stripe Connect Account ID, Wave Business ID, etc.
    KYCStatus  KYCStatus // PENDING | VERIFIED | REJECTED
}
```

### 4. Adapters Merchant Mode

| Adapter | Marché | KYC |
|---|---|---|
| `StripeConnectAdapter` | International | Stripe Identity (automatique) |
| `WaveMerchantAdapter` | Sénégal, Côte d'Ivoire | KYC Manuel (Wave for Business) |
| `OrangeMoneyMerchantAdapter` | CEDEAO | KYC Manuel (Orange Money PME) |
| `FreeMoneyMerchantAdapter` | Sénégal | KYC Manuel |

### 5. Activation Merchant Mode

```sql
-- Dans tenant_settings
merchant_mode         BOOLEAN NOT NULL DEFAULT FALSE,
merchant_processor    VARCHAR(50),  -- 'STRIPE_CONNECT' | 'WAVE_MERCHANT' | 'ORANGE_MERCHANT' | 'FREE_MERCHANT'
merchant_account_id   VARCHAR(255), -- ID compte marchand chez le provider
merchant_kyc_status   VARCHAR(20)   DEFAULT 'NOT_STARTED',
payment_region        VARCHAR(20)   DEFAULT 'INTERNATIONAL',  -- 'INTERNATIONAL' | 'CEDEAO'
```

Le Merchant Mode s'active via `POST /admin/v1/tenants/{id}/merchant/onboard` (super-admin après vérification KYC).

### 6. Table `tenant_settings` (nouveau)

```sql
CREATE TABLE tenant_settings (
    tenant_id           UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    logo_url            VARCHAR(500),
    primary_color       VARCHAR(7),   -- hex (#RRGGBB)
    locale              VARCHAR(10)  NOT NULL DEFAULT 'en',
    timezone            VARCHAR(50)  NOT NULL DEFAULT 'UTC',
    currency            VARCHAR(3)   NOT NULL DEFAULT 'USD',
    data_region         VARCHAR(10)  NOT NULL DEFAULT 'us'
                            CHECK (data_region IN ('us','eu','apac')),
    payment_region      VARCHAR(20)  NOT NULL DEFAULT 'INTERNATIONAL'
                            CHECK (payment_region IN ('INTERNATIONAL','CEDEAO')),
    merchant_mode       BOOLEAN NOT NULL DEFAULT FALSE,
    merchant_processor  VARCHAR(50),
    merchant_account_id VARCHAR(255),
    merchant_kyc_status VARCHAR(20)  NOT NULL DEFAULT 'NOT_STARTED',
    custom_domain       VARCHAR(255),
    b2b2c_enabled       BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 7. Schéma SQL Platform Mode

```sql
CREATE TABLE plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(50) NOT NULL UNIQUE,  -- 'free', 'starter', 'pro', 'enterprise'
    price_cents     BIGINT NOT NULL DEFAULT 0,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    interval        VARCHAR(10) NOT NULL DEFAULT 'MONTHLY'
                        CHECK (interval IN ('MONTHLY','YEARLY')),
    max_users       INTEGER,
    max_storage     BIGINT,
    max_api_calls   BIGINT,
    can_export      BOOLEAN NOT NULL DEFAULT FALSE,
    can_white_label BOOLEAN NOT NULL DEFAULT FALSE,
    can_merchant    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID UNIQUE NOT NULL REFERENCES tenants(id),
    plan_id                 UUID NOT NULL REFERENCES plans(id),
    status                  VARCHAR(20) NOT NULL DEFAULT 'TRIALING'
                                CHECK (status IN ('TRIALING','ACTIVE','PAST_DUE','CANCELED')),
    current_period_start    TIMESTAMPTZ NOT NULL,
    current_period_end      TIMESTAMPTZ NOT NULL,
    canceled_at             TIMESTAMPTZ,
    payment_method          VARCHAR(20) NOT NULL DEFAULT 'STRIPE',
    external_id             VARCHAR(255),  -- Stripe Subscription ID
    mobile_money_ref        VARCHAR(255),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 8. Schéma SQL Merchant Mode B2B2C

> Ces tables dépendent de `platform/end_customers` (RFC-0033). Elles sont dans les migrations de `platform/billing`.

```sql
CREATE TABLE tenant_invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    end_customer_id     UUID NOT NULL REFERENCES end_customers(id),
    amount_cents        BIGINT NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'XOF',
    status              VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                            CHECK (status IN ('DRAFT','SENT','PAID','OVERDUE','CANCELED')),
    payment_method      VARCHAR(30),
    transaction_ref     VARCHAR(255),
    platform_fee_cents  BIGINT NOT NULL DEFAULT 0,
    due_at              TIMESTAMPTZ,
    paid_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE invoice_line_items (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id       UUID NOT NULL REFERENCES tenant_invoices(id) ON DELETE CASCADE,
    description      VARCHAR(500) NOT NULL,
    quantity         INTEGER NOT NULL DEFAULT 1,
    unit_price_cents BIGINT NOT NULL,
    total_cents      BIGINT NOT NULL
);
```

### 9. Flux Merchant Mode B2B2C (exemple agence immo)

```
Locataire (end-customer)
    → POST /customer/v1/invoices/{id}/pay { method: "WAVE" }
        → WaveMerchantAdapter.CreateCustomerCheckout(tenant, endCustomer, 100000, "XOF")
            → Lien paiement Wave généré
                → Locataire confirme paiement sur téléphone
                    → Webhook Wave → POST /api/v1/billing/webhooks/wave-merchant
                        → Vérification signature HMAC Wave
                        → Idempotency check (processed_events)
                        → BEGIN TRANSACTION
                            → tenant_invoices.status = 'PAID', paid_at = NOW()
                            → INSERT event_outbox {type: "billing.invoice_paid", payload: {invoice_id, end_customer_id, amount}}
                        → COMMIT
                            → consumer outbox → customer_notifications → SMS quittance
                            → consumer outbox → audit_logs → INVOICE_PAID
```

### 10. Webhooks — idempotency

Tous les webhooks entrants (Stripe, Wave, Orange Money, Free Money) sont traités de manière idempotente via `processed_events` :

```sql
CREATE TABLE processed_events (
    event_id    VARCHAR(255) NOT NULL,
    source      VARCHAR(50)  NOT NULL,  -- 'stripe', 'wave', 'orange_money', 'free_money'
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, source)
);
```

Si `(event_id, source)` existe déjà → 200 OK sans retraitement.

### 11. Événements outbox émis par platform/billing

> Ajouté suite à la review RFC-0036 (Aziz) : ces types doivent être documentés comme contrat entre billing et customer_notifications.

| Type | Payload | Consommateurs |
|------|---------|---------------|
| `billing.invoice_paid` | `{invoice_id, end_customer_id, tenant_id, amount_cents, currency}` | customer_notifications (quittance SMS/email), audit |
| `billing.payment_reminder` | `{invoice_id, end_customer_id, tenant_id, due_at, amount_cents}` | customer_notifications (rappel) |
| `billing.subscription_renewed` | `{tenant_id, plan_id, next_period_end}` | notification staff |
| `billing.subscription_canceled` | `{tenant_id, canceled_at}` | notification staff, audit |

---

## **Drawbacks**

1. **KYC Mobile Money manuel** : Wave/Orange/Free n'ont pas d'API KYC automatique (contrairement à Stripe Identity). L'onboarding Merchant Mode Mobile Money nécessite un workflow super-admin avec upload de documents.
2. **Pas de renouvellement récurrent Mobile Money natif** : Mobile Money ne supporte pas les débits récurrents. Platform Mode CEDEAO = renouvellement manuel (notification J-7, J-1) avec re-paiement du lien.
3. **Commission plateforme Mobile Money** : Stripe Connect automatise la commission. Côté Mobile Money, la commission est gérée applicativement (à déduire avant payout), avec risque de réconciliation manuelle.

---

## **Alternatives**

### Agrégateur CinetPay / Julaya (unifie Wave + Orange + Free)

- ✅ Une seule intégration pour les 3 opérateurs CEDEAO
- ✅ Réduit la surface de maintenance
- ❌ Dépendance à un intermédiaire supplémentaire, frais additionnels
- **Verdict : évaluer en P1** — le port `PaymentProcessor` permet de switcher sans changer les use cases

---

## **Plan d'adoption**

1. **P0** : RFC rédigée (ce document)
2. **P1-2** : implémenter `platform/billing/` — Platform Mode (Stripe + Wave adapter) + tables plans/subscriptions/tenant_settings
3. **P1bis-8** : extension Merchant Mode (Stripe Connect + Wave Marchand) + tables invoices/line_items

---

## **References**

- [RFC-0012 — Platform Billing](0012-Platform-Billing-Abonnements.md) *(supersédé par ce RFC)*
- [RFC-0015 — Event Bus Outbox](0015-Event-Bus-Interne.md)
- [RFC-0019 — Platform Notification B2B](0019-Platform-Notification-B2B-Email-First.md)
- [RFC-0033 — Platform end_customers](0033-Platform-End-Customers.md) *(à créer)*
- [Stripe Connect Documentation](https://stripe.com/docs/connect)
- [Wave for Business API](https://www.wave.com/en/business/)
