# _ADR #0013:_ Platform `user` — Lazy profile creation à la première lecture

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

RFC-0009 spécifiait que le profil utilisateur est "créé automatiquement lors de l'inscription". Deux approches étaient possibles :

1. **Eager** : créer la ligne `user_profiles` dans `SignupOwnerUseCase` (immédiatement à l'inscription).
2. **Lazy** : créer la ligne `user_profiles` lors du premier `GET /api/v1/users/{userId}/profile`.

L'approche eager crée un couplage entre `platform/auth` et `platform/user` : auth devrait importer user, mais user dépend de permission qui dépend de membership, et tout ce graphe doit être wired dans auth — ou un événement outbox doit être défini.

---

## **Decision**

**Lazy creation** : `GetProfileUseCase` tente `repo.FindByUserID`. Si `ErrProfileNotFound`, et que `Input.Email` est fourni, il appelle `domain.NewFromEmail` et `repo.Upsert` avant de retourner. `Input.Email` est optionnel ; si absent et profil absent → `ErrProfileNotFound` propagé (HTTP 404).

L'email est fourni par le handler HTTP qui le lit depuis l'auth context (JWT claim). Le module `user` s'abonne aussi à l'event `auth.signup_owner` via le bus d'événements pour créer le profil eagerly en arrière-plan sans coupler auth ↔ user à la compilation.

---

## **Alternatives Considered**

### Création eager dans SignupOwnerUseCase
- ✅ Profil garanti présent dès le premier GET
- ❌ Couplage compile-time auth → user → permission → membership
- **Verdict : rejeté — viole la séparation des modules**

### Création via outbox uniquement (pas de lazy fallback)
- ✅ Découplage maximal
- ❌ Race condition : si l'event outbox est retardé, le premier GET retourne 404
- **Verdict : rejeté — UX dégradée au démarrage**

---

## **Consequences**

### **Pros**
- Zéro couplage compile-time entre `platform/auth` et `platform/user`
- `platform/user` fonctionne en standalone (pas besoin du bus d'events pour les tests)
- L'event `auth.signup_owner` crée le profil en background pour les workflows batch

### **Cons**
- Si le handler n'injecte pas l'email (bug de wiring), le premier GET retourne 404 au lieu de créer le profil
- Un upsert supplémentaire à la première lecture (négligeable)
