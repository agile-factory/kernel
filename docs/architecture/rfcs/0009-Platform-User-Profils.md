# _RFC #0009:_ Platform `user` — Gestion des Profils Utilisateurs

**date:** `2026 May 01`

---

## **Status:** `PROPOSED`

---

## **Summary**

Ce RFC définit la plateforme `user`, brique SaaS qui gère le **profil public d'un utilisateur** au sein du socle Agile Factory, distinct de son identité d'authentification (`platform/auth`).

La séparation `auth` / `user` suit le principe de responsabilité unique :
- `auth` : identité, credentials, tokens (qui es-tu ?)
- `user` : profil, préférences, informations publiques (comment te présentes-tu ?)

---

## **Motivation**

La plateforme `auth` gère l'identité technique (email, mot de passe hashé, rôle plateforme). Elle n'a pas vocation à porter les données de profil (nom affiché, avatar, préférences) qui :
- Changent fréquemment (update de préférences)
- Sont publiques au sein d'un tenant (autres membres peuvent voir le profil)
- Sont spécifiques à chaque SaaS (un SaaS RH a besoin de `job_title`, un SaaS immobilier d'`agency_name`)

Sans plateforme `user` séparée, ces données s'accumulent dans la table `users` d'`auth`, créant une entité "god object" difficile à maintenir et à étendre.

---

## **Proposal**

### 1. Modèle de domaine

```go
type UserProfile struct {
    UserID      string    // FK → users.id (1:1)
    DisplayName string
    AvatarURL   string
    Bio         string
    Locale      string    // ex: "fr-SN", "en-US"
    Timezone    string    // ex: "Africa/Dakar"
    UpdatedAt   time.Time
}
```

La table `user_profiles` est liée à `users` par une relation **1:1 optionnelle**. Un profil est créé automatiquement lors de l'inscription (avec `DisplayName` extrait du préfixe email) et peut être mis à jour ultérieurement.

### 2. Use Cases

| Use Case | Description |
|---|---|
| `GetProfileUseCase` | Récupérer le profil d'un utilisateur par `userID` |
| `UpdateProfileUseCase` | Mettre à jour `displayName`, `avatarURL`, `bio`, `locale`, `timezone` |

### 3. API HTTP

```
GET  /api/v1/users/{userId}/profile
→ 200 OK { userId, displayName, avatarURL, bio, locale, timezone }

PATCH /api/v1/users/{userId}/profile
Body: { "displayName": "...", "locale": "fr-SN" }
→ 200 OK { ...profil mis à jour }
```

**Autorisation** : un utilisateur ne peut modifier que son propre profil (`X-User-ID == userId`). La lecture est accessible à tout utilisateur du même tenant.

### 4. Schéma SQL

```sql
CREATE TABLE user_profiles (
    user_id      UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(100),
    avatar_url   TEXT,
    bio          TEXT,
    locale       VARCHAR(10)  DEFAULT 'fr-SN',
    timezone     VARCHAR(50)  DEFAULT 'Africa/Dakar',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 5. Extensibilité par SaaS

Chaque SaaS peut étendre le profil via une table dédiée dans `modules/` :

```sql
-- modules/crm/
CREATE TABLE crm_user_extensions (
    user_id    UUID PRIMARY KEY REFERENCES users(id),
    job_title  VARCHAR(100),
    department VARCHAR(100)
);
```

Cette extension n'est pas portée par la plateforme `user` elle-même, respectant le découplage `platform/` vs `modules/`.

---

## **Drawbacks**

### Join nécessaire pour afficher profil + identité
Afficher "qui a fait cette action" nécessite un join `users` + `user_profiles`. La dénormalisation partielle (cacher `display_name` dans `audit_logs.metadata`) est une mitigation acceptable.

### Profil créé automatiquement = données minimales
Le profil auto-créé à l'inscription n'a qu'un `displayName` générique. L'utilisateur doit penser à le compléter.

---

## **Alternatives**

### Profil directement dans la table `users`
- ✅ Moins de joins
- ❌ `users` devient une "god table" avec des colonnes non liées à l'auth
- **Verdict : rejeté**

### Profil par tenant (un profil différent par workspace, comme Slack)
- ✅ Un utilisateur peut avoir un nom différent selon l'organisation
- ❌ Complexité élevée pour la v1
- **Verdict : différé** — à reconsidérer si des SaaS le nécessitent

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0007 : Platform Auth (identité de base)
- RFC-0003 : Multi-Tenancy (tenant context)
