# _RFC #0026:_ Platform `usage_metering` — Comptage d'Usage par Tenant

**date:** `2026 May 12`

> ⚠️ **DEFERRED — Implémentation P2+.** Ce RFC fixe la vision pour le billing usage-based. Le package `platform/usage_metering/` existe en squelette (README uniquement).

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC définit la plateforme `usage_metering`, qui compte l'usage par tenant et par métrique pour alimenter le billing usage-based (Stripe Metered Billing, RFC-0020).

---

## **Motivation**

Le billing per-seat (subscriptions fixes) couvre le Platform Mode (RFC-0020). Pour le BaaS et certains profils B2B2C, un billing **usage-based** est requis :
- BaaS : facturation par requête API, par bandwidth, par storage utilisé
- B2B2C : facturation par end-customer actif, par transaction Merchant Mode

---

## **Proposal**

### 1. Port abstrait `Meter`

```go
// platform/usage_metering/domain/meter.go
package usage_metering

type Meter interface {
    // Record enregistre une unité d'usage.
    // metric = identifiant de la métrique ("api_requests", "end_customers_active", "storage_bytes")
    // value  = quantité (généralement 1, mais peut être bytes pour storage)
    Record(ctx context.Context, tenantID, metric string, value int64) error

    // Aggregate retourne l'usage cumulé sur une période.
    Aggregate(ctx context.Context, tenantID, metric string, from, to time.Time) (int64, error)
}
```

### 2. Tables SQL

```sql
CREATE TABLE usage_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    metric      VARCHAR(100) NOT NULL,  -- 'api_requests', 'storage_bytes', etc.
    value       BIGINT NOT NULL DEFAULT 1,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (recorded_at);

-- Partition mensuelle (ex: usage_events_2026_05)
CREATE TABLE usage_events_2026_05 PARTITION OF usage_events
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE INDEX idx_usage_events_tenant_metric ON usage_events(tenant_id, metric, recorded_at DESC);

-- Table agrégée quotidiennement (job cron)
CREATE TABLE usage_aggregates_daily (
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    metric      VARCHAR(100) NOT NULL,
    date        DATE NOT NULL,
    total       BIGINT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, metric, date)
);
```

### 3. Agrégation quotidienne

Un worker cron (quotidien, 00:05 UTC) agrège `usage_events` → `usage_aggregates_daily` pour chaque couple `(tenant_id, metric)` du jour précédent.

```sql
INSERT INTO usage_aggregates_daily (tenant_id, metric, date, total)
SELECT tenant_id, metric, DATE(recorded_at), SUM(value)
FROM usage_events
WHERE recorded_at >= $1 AND recorded_at < $2
GROUP BY tenant_id, metric, DATE(recorded_at)
ON CONFLICT (tenant_id, metric, date) DO UPDATE
    SET total = EXCLUDED.total, updated_at = NOW();
```

### 4. Couplage Stripe Metered Billing (P2+)

```go
// platform/usage_metering/infrastructure/stripe_reporter.go

// ReportToStripe pousse les agrégats vers Stripe Metered Billing (une fois par mois, avant facturation)
func (r *StripeReporter) ReportToStripe(ctx context.Context, tenantID, metric string, period time.Time) error {
    total, _ := r.meter.Aggregate(ctx, tenantID, metric, periodStart, periodEnd)
    _, err := r.stripeClient.UsageRecords.New(&stripe.UsageRecordParams{
        Quantity:         stripe.Int64(total),
        Timestamp:        stripe.Int64(period.Unix()),
        SubscriptionItem: stripe.String(tenant.StripeMeteredItemID),
    })
    return err
}
```

### 5. Métriques exposées (exemples)

| Métrique | Unité | Consommateur |
|---|---|---|
| `api_requests` | count | BaaS — billing par requête |
| `end_customers_active_monthly` | count | B2B2C — billing par end-customer actif |
| `storage_bytes` | bytes | RFC-0017 — billing par storage |
| `webhook_deliveries` | count | BaaS — monitoring |
| `invoice_transactions` | count | Merchant Mode — frais variables |

---

## **Références**

- [RFC-0020 — Platform Billing Hybride](0020-Platform-Billing-Hybride-Merchant-Mode.md)
- [RFC-0022 — Platform API Keys](0022-Platform-APIKey-PAT-Scopes.md)
- [Stripe Metered Billing](https://stripe.com/docs/billing/subscriptions/metered-billing)
