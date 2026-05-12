# _RFC #0023:_ Platform `webhooks_out` — Webhooks Sortants Signés

**date:** `2026 May 12`

> ⚠️ **DEFERRED — Implémentation P2+.** Ce RFC fixe la vision pour le profil BaaS futur. Le package `platform/webhooks_out/` existe en squelette (README uniquement). L'outbox transactionnel (RFC-0015) est déjà le terrain préparé : `webhooks_out` en sera simplement un consommateur.

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC définit la plateforme `webhooks_out`, qui permet aux tenants de **recevoir des notifications asynchrones signées HMAC** lorsque des événements se produisent sur la plateforme (paiements, end-customers, memberships…).

Le `WebhookDispatcher` consomme l'outbox transactionnel (RFC-0015) et POSTe les events vers les endpoints configurés par le tenant, avec signature HMAC-SHA256, retry exponentiel et dead-letter queue.

---

## **Motivation**

Les intégrations B2B modernes s'appuient sur les webhooks pour réagir en temps réel aux événements de la plateforme sans polling. Sans webhooks sortants :
- Les tenants ne peuvent pas synchroniser leurs systèmes (ERP, CRM, outils internes) avec la plateforme.
- Le profil BaaS (développeurs de tenants) manque de son canal de communication principal.

---

## **Proposal**

### 1. Modèle

```go
type WebhookEndpoint struct {
    ID          string
    TenantID    string
    URL         string
    Secret      string    // HMAC secret — chiffré en DB, jamais exposé via API
    EventTypes  []string  // filtre ["billing.invoice_paid", "auth.member_invited", "*"]
    IsActive    bool
    CreatedAt   time.Time
}

type WebhookDelivery struct {
    ID             string
    EndpointID     string
    EventType      string
    Payload        map[string]any
    Status         DeliveryStatus  // PENDING | SUCCESS | FAILED | DLQ
    AttemptCount   int
    LastAttemptAt  *time.Time
    NextAttemptAt  *time.Time
    ResponseStatus int
    ResponseBody   string
    CreatedAt      time.Time
}
```

### 2. Signature HMAC (style Stripe)

```
Header: X-Webhook-Signature: t=<unix_ts>,v1=<hex_hmac>

Payload signé = "<unix_ts>.<json_body>"
HMAC = HMAC-SHA256(secret, payload_signé)
```

Implémenté dans `kernel/webhook/signing.go` (helper partagé) :

```go
package webhook

func Sign(secret []byte, unixTs int64, body []byte) string
func Verify(secret []byte, header string, body []byte, tolerance time.Duration) error
```

### 3. WebhookDispatcher — consommateur de l'outbox

```go
// platform/webhooks_out/dispatcher.go
type WebhookDispatcher struct {
    endpointRepo  WebhookEndpointRepository
    deliveryRepo  WebhookDeliveryRepository
    httpClient    *http.Client
    signer        webhook.Signer
}

// Consume implémente events.EventHandler
func (d *WebhookDispatcher) Handle(ctx context.Context, event events.Event) error {
    endpoints, _ := d.endpointRepo.FindActiveByTenantAndType(ctx, event.TenantID, event.Type)
    for _, ep := range endpoints {
        delivery := d.createDelivery(ep, event)
        d.deliver(ctx, delivery, ep) // retry géré par le scheduler
    }
    return nil
}
```

### 4. Retry — backoff exponentiel

| Tentative | Délai |
|---|---|
| 1 | Immédiat |
| 2 | 30s |
| 3 | 5 min |
| 4 | 30 min |
| 5–10 | 1h, 2h, 4h, 8h, 12h, 24h |
| > 10 | DLQ (status = 'DLQ') |

Un endpoint en DLQ peut être rejoué via `POST /api/v1/tenants/{id}/webhooks/endpoints/{eid}/replay` (admin tenant).

### 5. API HTTP

```
# Configuration endpoints (JWT — ROLE_OWNER/ADMIN)
POST   /api/v1/tenants/{id}/webhooks/endpoints
GET    /api/v1/tenants/{id}/webhooks/endpoints
DELETE /api/v1/tenants/{id}/webhooks/endpoints/{eid}

# Monitoring livraisons
GET /api/v1/tenants/{id}/webhooks/deliveries?status=FAILED

# Replay DLQ
POST /api/v1/tenants/{id}/webhooks/endpoints/{eid}/replay
POST /api/v1/tenants/{id}/webhooks/deliveries/{did}/replay
```

### 6. Tables SQL

```sql
CREATE TABLE webhook_endpoints (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    url          VARCHAR(2000) NOT NULL,
    secret_enc   TEXT NOT NULL,      -- AES-256-GCM
    event_types  TEXT[] NOT NULL DEFAULT ARRAY['*'],
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id     UUID NOT NULL REFERENCES webhook_endpoints(id),
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempt_count   INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ,
    response_status INTEGER,
    response_body   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_wh_deliveries_next_attempt ON webhook_deliveries(next_attempt_at)
    WHERE status IN ('PENDING','FAILED');
```

---

## **References**

- [RFC-0015 — Event Bus Outbox](0015-Event-Bus-Interne.md) *(terrain préparé — webhooks_out en est un consommateur)*
- [RFC-0022 — API Keys](0022-Platform-APIKey-PAT-Scopes.md)
- [Stripe Webhooks — style de signature](https://stripe.com/docs/webhooks/signatures)
