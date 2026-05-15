# Schéma relationnel PostgreSQL — v0.1.0

Toutes les tables livrées par les modules `kernel/events` + `platform/{tenant,auth,membership,audit,notification}`. Les contraintes FK sont représentées au sens lecture business : `ON DELETE CASCADE` ou `SET NULL` selon la criticité.

```mermaid
erDiagram
    tenants ||--o| tenant_settings : "1 to 0..1"
    tenants ||--o{ memberships : "has"
    tenants ||--o{ refresh_tokens : "scopes"
    tenants ||--o{ invite_tokens : "issues"
    tenants ||--o{ audit_logs : "logs against"
    tenants ||--o{ notification_logs : "logs for"

    users ||--o{ memberships : "has"
    users ||--o{ refresh_tokens : "owns"
    users ||--o{ email_verification_tokens : "verifies"
    users ||--o{ audit_logs : "acts"
    users }o--|| invite_tokens : "invited by"

    tenants {
        uuid id PK
        varchar name UK
        varchar slug UK
        bool is_active
        timestamptz created_at
        timestamptz updated_at
    }

    tenant_settings {
        uuid tenant_id PK_FK
        text logo_url
        varchar primary_color
        varchar locale
        varchar timezone
        varchar currency
        varchar data_region "eu/us/apac"
        varchar custom_domain UK_partial
        varchar payment_region "INTERNATIONAL/CEDEAO"
        bool merchant_mode
        bool b2b2c_enabled
    }

    users {
        uuid id PK
        varchar email UK
        text password "bcrypt hash"
        user_role role "ROLE_ADMIN/AGENT/VIEWER"
        timestamptz email_verified_at
        timestamptz created_at
        timestamptz updated_at
    }

    memberships {
        uuid id PK
        uuid user_id FK
        uuid tenant_id FK
        member_role role "OWNER/ADMIN/AGENT/VIEWER"
        member_status status "PENDING/ACTIVE/REVOKED"
        uuid invited_by FK_nullable
        timestamptz joined_at
        timestamptz created_at
        timestamptz updated_at
    }

    refresh_tokens {
        uuid id PK
        uuid user_id FK
        uuid tenant_id FK_nullable "scopé au signup/invite, null au login"
        char token_hash UK "SHA-256 hex"
        timestamptz expires_at "default NOW+30d"
        timestamptz revoked_at
        timestamptz created_at
    }

    email_verification_tokens {
        uuid id PK
        uuid user_id FK
        varchar token UK "128 chars opaque"
        timestamptz expires_at "default NOW+24h"
        timestamptz consumed_at
        timestamptz created_at
    }

    invite_tokens {
        uuid id PK
        uuid tenant_id FK
        uuid invited_by FK
        varchar email
        member_role role
        varchar token UK "128 chars opaque"
        timestamptz expires_at "default NOW+7d"
        timestamptz consumed_at
        timestamptz revoked_at
        timestamptz created_at
    }

    audit_logs {
        uuid id PK
        uuid tenant_id FK_nullable
        uuid actor_id FK_nullable
        audit_action action "USER_REGISTERED/MEMBER_INVITED/..."
        varchar target_type
        uuid target_id
        jsonb metadata
        timestamptz occurred_at
    }

    notification_logs {
        uuid id PK
        uuid tenant_id_nullable
        uuid recipient_id_nullable
        varchar event_type
        varchar channel "EMAIL"
        varchar status "PENDING/SENT/FAILED"
        varchar external_id "ID retourné par EmailSender"
        text error_message
        jsonb metadata
        timestamptz created_at
        timestamptz updated_at
    }

    event_outbox {
        uuid id PK
        varchar event_type
        jsonb payload
        jsonb meta
        timestamptz created_at
        timestamptz published_at "NULL = pas encore traité"
        int attempts
        timestamptz next_attempt_at
        text last_error
    }

    processed_events {
        uuid event_id PK "idempotence côté handler"
        varchar event_type
        timestamptz processed_at
    }
```

## Tables annexes non liées au graphe

Deux tables vivent indépendamment du graphe FK :

- **`event_outbox`** (`kernel/events`) — la queue publish/dispatch. Aucune FK : un event peut référencer un tenant supprimé, le dispatcher s'en moque.
- **`processed_events`** (`kernel/events`) — table d'idempotence côté handler (le handler insère l'event_id avec `ON CONFLICT DO NOTHING` pour ne pas dupliquer son side-effect).

## Index importants

| Table | Index | Usage |
|---|---|---|
| `users` | `idx_users_email` | Lookup login. |
| `memberships` | `idx_memberships_status` + partial unique `(user_id, tenant_id) WHERE status != 'REVOKED'` | Empêche les duplicats, autorise réinvite. |
| `refresh_tokens` | `idx_refresh_tokens_active (token_hash) WHERE revoked_at IS NULL` | Refresh hot path. |
| `email_verification_tokens` | `idx_..._pending (token) WHERE consumed_at IS NULL` | Verify hot path. |
| `invite_tokens` | `idx_..._pending (token) WHERE consumed_at IS NULL AND revoked_at IS NULL` | Accept hot path. |
| `audit_logs` | `(tenant_id, occurred_at DESC, id DESC)` | Cursor pagination v2 (RFC-0006). |
| `event_outbox` | `idx_event_outbox_ready (next_attempt_at) WHERE published_at IS NULL` | Dispatcher poll. |

## Conventions de schéma

- **UUIDs primary keys** — `gen_random_uuid()` côté DB (`pgcrypto`).
- **`created_at` / `updated_at` TIMESTAMPTZ** sur toutes les entités business.
- **`ON DELETE CASCADE`** sur `tenants → memberships, refresh_tokens, invite_tokens` (la suppression d'un tenant emporte ses dépendants).
- **`ON DELETE SET NULL`** sur `audit_logs.tenant_id`, `audit_logs.actor_id` — on garde la trace même si l'acteur disparaît.
- **Enums Postgres** plutôt que CHECK constraints sur strings — typage strict côté pgx, et `ALTER TYPE ... ADD VALUE` pour étendre.
- **Versioning des migrations par-package** — `schema_migrations_<feature>` séparé (ADR-0006).

Sources : voir `*/migrations/*.up.sql` dans chaque module.
