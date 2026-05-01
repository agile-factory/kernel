# _RFC #0012:_ Platform `billing` — Abonnements et Facturation

**date:** `2026 May 01`

---

## **Status:** `PROPOSED`

---

## **Summary**

Ce RFC définit la plateforme `billing`, brique SaaS qui gère les **abonnements, les plans tarifaires et la facturation** des tenants dans le socle Agile Factory.

La plateforme couvre le cycle de vie d'un abonnement (essai gratuit → plan payant → annulation), les quotas d'usage par plan, et l'intégration avec un processeur de paiement externe (Stripe recommandé).

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
    ExternalID    string // Stripe Subscription ID
}
```

### 2. Use Cases

| Use Case | Description |
|---|---|
| `StartTrialUseCase` | Démarre une période d'essai (14 jours) à la création du tenant |
| `UpgradePlanUseCase` | Passage à un plan supérieur via Stripe Checkout |
| `CancelSubscriptionUseCase` | Annulation (fin de période en cours) |
| `CheckQuotaUseCase` | Vérifie si le tenant peut effectuer une action selon son plan |
| `GetCurrentPlanUseCase` | Retourne le plan actif d'un tenant |

### 3. Intégration Stripe

```
Tenant crée compte → StartTrial → Stripe Customer créé
Tenant upgrade → Stripe Checkout Session → Webhook stripe → UpgradePlan
Stripe renouvellement → Webhook stripe → UpdateSubscription
Stripe impayé → Webhook stripe → Subscription PAST_DUE → accès limité
```

Les webhooks Stripe sont vérifiés par signature (`Stripe-Signature` header).

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
    stripe_subscription_id VARCHAR(255),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## **Drawbacks**

### Dépendance forte à Stripe
Le processeur de paiement est abstrait via le port `PaymentProcessor`, mais Stripe reste la seule implémentation prévue. Migration difficile.

### Webhooks asynchrones = état éventuel
Un upgrade initié à 10h00 peut ne pas être reflété en DB avant 10h00:05 (délai webhook). Comportement à gérer avec soin dans l'UI.

### Pas de billing pour le marché sénégalais en v1
Stripe ne supporte pas Orange Money ou Wave nativement. Un adaptateur dédié (RFC séparé) sera nécessaire pour les SaaS ciblant le marché CEDEAO.

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

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0003 : Multi-Tenancy (un abonnement par tenant)
- RFC-0011 : Platform Notification (emails de facturation)
- [Stripe Webhooks Documentation](https://stripe.com/docs/webhooks)
- [Stripe Billing Documentation](https://stripe.com/docs/billing)
