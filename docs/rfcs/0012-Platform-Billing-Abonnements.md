# _RFC #0012:_ Platform `billing` — Abonnements et Facturation

**date:** `2026 May 01`

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC définit la plateforme `billing`, brique SaaS qui gère **deux types de flux financiers** dans le socle Agile Factory :

| Mode | Sens du paiement | Description |
|---|---|---|
| **Platform Mode** | Plateforme → Tenant | Abonnements SaaS : le tenant paie la plateforme pour utiliser le socle |
| **Merchant Mode** | Tenant → Client final | Facturation B2B2C : le tenant encaisse ses propres clients via la plateforme |

La plateforme couvre le cycle de vie d'un abonnement (essai gratuit → plan payant → annulation), les quotas d'usage par plan, l'intégration avec des processeurs de paiement (Stripe, Wave, Orange Money), et l'infrastructure de facturation que les tenants peuvent activer pour leurs propres clients.

---

## **Motivation**

Tout SaaS B2B monétise ses services via des abonnements. Sans plateforme billing commune :
- Chaque SaaS réimplémente son intégration Stripe
- Les plans et quotas sont hardcodés dans le code métier
- La gestion des impayés et des downgrades est ad hoc

Le billing étant transversal (il affecte toutes les fonctionnalités selon le plan), il doit être une brique plateforme, pas un module métier.

---

## **Proposal**

### 1. Modèle de domaine

```go
type Plan struct {
    ID       string
    Name     string       // "free", "starter", "pro", "enterprise"
    Price    int64        // en centimes (100 = 1.00 USD)
    Currency string       // "USD", "XOF"
    Interval BillingInterval // MONTHLY | YEARLY
    Quotas   PlanQuotas
}

type PlanQuotas struct {
    MaxUsers      int
    MaxStorage    int64 // en bytes
    MaxAPICallsPerMonth int64
    CanExport     bool
    CanWhiteLabel bool
}

type Subscription struct {
    ID            string
    TenantID      string
    PlanID        string
    Status        SubscriptionStatus // TRIALING | ACTIVE | PAST_DUE | CANCELED
    CurrentPeriodStart time.Time
    CurrentPeriodEnd   time.Time
    CanceledAt    *time.Time
    ExternalID    string  // Stripe Subscription ID (vide si paiement Mobile Money)
    PaymentMethod PaymentMethod // STRIPE | MOBILE_MONEY
    MobileMoneyRef string  // référence transaction Wave/Orange Money (optionnel)
}
```

### 2. Use Cases

| Use Case | Description |
|---|---|
| `StartTrialUseCase` | Démarre une période d'essai (14 jours) à la création du tenant |
| `UpgradePlanUseCase` | Passage à un plan supérieur via Stripe Checkout ou Mobile Money |
| `CancelSubscriptionUseCase` | Annulation (fin de période en cours) |
| `CheckQuotaUseCase` | Vérifie si le tenant peut effectuer une action selon son plan |
| `GetCurrentPlanUseCase` | Retourne le plan actif d'un tenant |
| `RecordMobileMoneyPaymentUseCase` | Enregistre un paiement Mobile Money confirmé (Wave, Orange Money, Free Money) |

### 3. Intégration Stripe

```
Tenant crée compte → StartTrial → Stripe Customer créé
Tenant upgrade → Stripe Checkout Session → Webhook stripe → UpgradePlan
Stripe renouvellement → Webhook stripe → UpdateSubscription
Stripe impayé → Webhook stripe → Subscription PAST_DUE → accès limité
```

Les webhooks Stripe sont vérifiés par signature (`Stripe-Signature` header).

### 3b. Intégration Mobile Money (marché CEDEAO)

Le port `PaymentProcessor` est implémenté par plusieurs adaptateurs :

| Adaptateur | Fournisseur | Marché | Mécanique |
|---|---|---|---|
| `StripeAdapter` | Stripe | International | Checkout Session + Webhook |
| `WaveAdapter` | Wave | Sénégal, Côte d'Ivoire | Wave Checkout Link + webhook |
| `OrangeMoneyAdapter` | Orange Money | CEDEAO (15 pays) | API Orange Money B2C + callback |
| `FreeMoneyAdapter` | Free Money (Expresso) | Sénégal | API Expresso + callback |

**Alternative recommandée :** un agrégateur comme **CinetPay** ou **Julaya** unifie Wave, Orange Money et Free Money sous une seule API — à trancher en Open Question #5.

**Flux Mobile Money :**
```
Tenant upgrade → Génération lien paiement Wave/Orange → Redirect utilisateur
→ Callback webhook → RecordMobileMoneyPaymentUseCase → Subscription ACTIVE
```

Le renouvellement est **manuel en v1** (pas de débit récurrent natif) : le tenant est notifié avant expiration via RFC-0011 WhatsApp/Email et renouvelle lui-même.

### 4. Garde d'accès par quota

Le `CheckQuotaUseCase` est injecté dans les use cases métier des modules :

```go
// modules/crm/usecase/create_contact.go
if err := uc.quota.Check(ctx, tenantID, "contacts"); err != nil {
    return nil, billing.ErrQuotaExceeded
}
```

### 5. Schéma SQL

```sql
CREATE TABLE plans (
    id            UUID PRIMARY KEY,
    name          VARCHAR(50) NOT NULL UNIQUE,
    price_cents   INTEGER NOT NULL DEFAULT 0,
    currency      VARCHAR(3) NOT NULL DEFAULT 'USD',
    interval      VARCHAR(10) NOT NULL DEFAULT 'MONTHLY',
    max_users     INTEGER,
    max_storage   BIGINT,
    can_export    BOOLEAN NOT NULL DEFAULT FALSE,
    can_white_label BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE subscriptions (
    id                   UUID PRIMARY KEY,
    tenant_id            UUID UNIQUE NOT NULL REFERENCES tenants(id),
    plan_id              UUID NOT NULL REFERENCES plans(id),
    status               VARCHAR(20) NOT NULL DEFAULT 'TRIALING',
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    canceled_at          TIMESTAMPTZ,
    payment_method       VARCHAR(20) NOT NULL DEFAULT 'STRIPE', -- 'STRIPE' | 'WAVE' | 'ORANGE_MONEY' | 'FREE_MONEY'
    stripe_subscription_id VARCHAR(255),
    mobile_money_ref     VARCHAR(255), -- référence transaction Mobile Money
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## **Drawbacks**

### Dépendance forte à Stripe
Le processeur de paiement est abstrait via le port `PaymentProcessor`, mais Stripe reste la seule implémentation prévue. Migration difficile.

### Webhooks asynchrones = état éventuel
Un upgrade initié à 10h00 peut ne pas être reflété en DB avant 10h00:05 (délai webhook). Comportement à gérer avec soin dans l'UI.

### Pas de renouvellement automatique Mobile Money en v1
Wave, Orange Money et Free Money ne supportent pas les débits récurrents natifs. En v1, le renouvellement Mobile Money est **manuel** : le tenant reçoit une notification (RFC-0011) avant expiration et renouvelle manuellement. L'automatisation (via paiement récurrent Wave Business) est différée à v2.

### KYC et conformité Merchant Mode
Activer le Merchant Mode implique un processus de KYC (Know Your Customer) pour vérifier l'identité du tenant marchand. Stripe Connect gère ce KYC automatiquement. Pour les Mobile Money marchands, l'inscription au service B2B (Wave Business, Orange Money PME) est manuelle et varie par opérateur et par pays.

---

## **Proposal (suite)**

### 6. Merchant Mode — Le tenant encaisse ses propres clients

Le **Merchant Mode** permet aux tenants d'utiliser la plateforme `billing` comme infrastructure de paiement pour leurs propres clients finaux (B2B2C). Le tenant devient **marchand** ; ses clients sont ses acheteurs.

#### 6a. Modèle de domaine complémentaire

```go
// Représente un client final du tenant (pas un utilisateur de la plateforme)
type TenantCustomer struct {
    ID          string
    TenantID    string   // le tenant qui gère ce client
    Name        string
    Email       string
    Phone       string   // nécessaire pour Mobile Money
    ExternalID  string   // Stripe Customer ID du sub-account (Stripe Connect)
    CreatedAt   time.Time
}

// Facture émise par un tenant à son client
type TenantInvoice struct {
    ID             string
    TenantID       string
    CustomerID     string
    AmountCents    int64
    Currency       string          // "XOF", "USD"
    Status         InvoiceStatus   // DRAFT | SENT | PAID | OVERDUE | CANCELED
    PaymentMethod  PaymentMethod   // STRIPE | WAVE | ORANGE_MONEY | FREE_MONEY
    TransactionRef string          // référence paiement externe
    DueAt          time.Time
    PaidAt         *time.Time
    LineItems      []InvoiceLineItem
    CreatedAt      time.Time
}
```

#### 6b. Use Cases Merchant Mode

| Use Case | Description |
|---|---|
| `OnboardMerchantUseCase` | Active le Merchant Mode pour un tenant (KYC, compte bancaire ou Mobile Money) |
| `CreateTenantCustomerUseCase` | Crée un client final dans le carnet du tenant |
| `CreateInvoiceUseCase` | Génère une facture pour un client du tenant |
| `SendInvoiceUseCase` | Envoie la facture par email/WhatsApp (via RFC-0011) avec un lien de paiement |
| `RecordInvoicePaymentUseCase` | Enregistre un paiement reçu (webhook Stripe/Mobile Money) |
| `ListTenantInvoicesUseCase` | Liste les factures d'un tenant (paginé, cursor-based via RFC-0006 v2) |

#### 6c. Infrastructure de paiement Merchant Mode

**Stripe Connect** pour les paiements carte internationaux :
```
Tenant s'onboarde → Stripe Connect Account créé
Tenant crée facture → Stripe Payment Link généré
Client paie → Webhook Stripe Connect → RecordInvoicePayment
Fonds virés → Compte Stripe Connect du tenant (moins la commission plateforme)
```

**Mobile Money** pour les paiements locaux (CEDEAO) :
```
Tenant crée facture → Lien paiement Wave/Orange Money généré
Client paie sur son téléphone → Callback webhook → RecordInvoicePayment
Fonds crédités → Compte Wave/Orange Money du tenant marchand
```

La plateforme peut prélever une **commission** (`platform_fee_cents`) sur chaque transaction Merchant Mode (modèle marketplace).

#### 6d. Schéma SQL Merchant Mode

```sql
CREATE TABLE tenant_customers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    email       VARCHAR(255),
    phone       VARCHAR(30),   -- format international (+221...)
    external_id VARCHAR(255),  -- Stripe Customer ID
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenant_invoices (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    customer_id      UUID NOT NULL REFERENCES tenant_customers(id),
    amount_cents     BIGINT NOT NULL,
    currency         VARCHAR(3) NOT NULL DEFAULT 'XOF',
    status           VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    payment_method   VARCHAR(20),  -- 'STRIPE' | 'WAVE' | 'ORANGE_MONEY' | 'FREE_MONEY'
    transaction_ref  VARCHAR(255), -- référence paiement externe
    platform_fee_cents BIGINT NOT NULL DEFAULT 0,
    due_at           TIMESTAMPTZ,
    paid_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE invoice_line_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id  UUID NOT NULL REFERENCES tenant_invoices(id) ON DELETE CASCADE,
    description VARCHAR(500) NOT NULL,
    quantity    INTEGER NOT NULL DEFAULT 1,
    unit_price_cents BIGINT NOT NULL,
    total_cents BIGINT NOT NULL
);
```

---


## **Alternatives**

### Billing dans chaque module
- ✅ Plus simple initialement
- ❌ Duplication des intégrations Stripe
- **Verdict : rejeté**

### Paddle (alternative à Stripe)
- ✅ Gestion de la TVA/taxes plus simple (Merchant of Record)
- ❌ Moins répandu, API moins documentée
- **Verdict : possible** — le port `PaymentProcessor` permet de switcher

### Billing open source (Lago, Killbill)
- ✅ Auto-hébergeable, pas de commission
- ❌ Complexité opérationnelle élevée
- **Verdict : différé** — pour les volumes importants

---

## **Open Questions**

1. Devise principale : USD ou XOF ? Peut-on supporter les deux ?
2. Faut-il un portail de facturation self-service (Stripe Customer Portal) dès la v1 ?
3. Gestion des remises et des coupons de réduction ?
4. Comment gérer le downgrade (suppression des utilisateurs excédentaires) ?
5. **Mobile Money** : Wave Business API ou agrégateur (Julaya, CinetPay) pour mutualiser Orange Money + Wave + Free Money sur une seule intégration ?
6. **Mobile Money** : comment gérer les remboursements en cas d'annulation anticipée (les APIs Mobile Money ne supportent pas tous le refund automatique) ?
7. **Merchant Mode** : quel pourcentage de commission plateforme sur les transactions Merchant ? Fixé dans le plan ou configurable par tenant ?
8. **Merchant Mode** : doit-on exposer un endpoint public (`GET /pay/{invoice_id}`) permettant au client final de payer sans compte sur la plateforme ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0003 : Multi-Tenancy (un abonnement par tenant)
- RFC-0011 : Platform Notification (emails de facturation)
- [Stripe Webhooks Documentation](https://stripe.com/docs/webhooks)
- [Stripe Billing Documentation](https://stripe.com/docs/billing)
