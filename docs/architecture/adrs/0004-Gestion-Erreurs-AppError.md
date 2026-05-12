# _ADR #0004:_ Gestion d'Erreurs Typées avec `AppError`

**date:** `2026 Apr 30`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

Un socle SaaS multi-produit a besoin d'une gestion d'erreurs qui satisfait plusieurs contraintes simultanément :
1. **Observabilité** : les erreurs doivent être traçables jusqu'à leur origine dans les logs
2. **Sécurité** : les détails internes (stack traces, causes DB) ne doivent jamais atteindre l'API publique
3. **Cohérence** : tous les handlers doivent retourner le même format JSON d'erreur
4. **Contrat API stable** : les codes d'erreur font partie du contrat API et doivent être versionnés

Les approches classiques (`fmt.Errorf`, erreurs génériques) ne permettent pas de lier une erreur domaine à un code HTTP et à un code stable machine-readable simultanément.

---

## **Decision**

Implémenter un type `AppError` centralisé dans `kernel/errors/` :

```go
type AppError struct {
    Code       string    // code stable machine-readable: "USER_NOT_FOUND"
    Message    string    // message human-readable pour l'API
    HTTPCode   int       // code HTTP correspondant
    Cause      error     // erreur originale (jamais sérialisée en API)
    Stack      []Frame   // jusqu'à 64 frames, capturées à la création
    OccurredAt time.Time
    TraceID    string
}
```

Des constructeurs sémantiques sont fournis : `errors.NotFound(msg)`, `errors.Unauthorized(msg)`, `errors.Forbidden(msg)`, `errors.BadRequest(msg)`, `errors.WrapInternal(err)`.

Des codes stables sont déclarés dans `codes.go` : `CODE_USER_NOT_FOUND`, `CODE_TENANT_NOT_FOUND`, etc.

En production/staging : seuls `code` et `message` sont exposés. En développement : un champ `debug` avec le premier frame de stack est ajouté.

---

## **Alternatives Considered**

### Erreurs stdlib + codes HTTP dans les handlers
```go
http.Error(w, "not found", http.StatusNotFound)
```
- ✅ Zéro infrastructure
- ❌ Pas de code machine-readable côté client
- ❌ Format d'erreur incohérent entre handlers
- **Verdict : rejeté**

### pkg/errors + sentinel errors
- ✅ Stack traces via `pkg/errors`
- ❌ Bibliothèque non maintenue (archivée)
- ❌ Pas de mapping HTTP ni de code stable
- **Verdict : rejeté**

### Middleware global de recovery seulement
- ✅ Simple
- ❌ Perd toute la sémantique domaine (auth, not found, validation)
- **Verdict : insuffisant seul**

---

## **Consequences**

### **Pros**
- Un seul format JSON d'erreur dans toute l'API : `{ "error": { "code": "...", "message": "..." } }`
- Stack traces capturées au moment de l'erreur, disponibles dans les logs uniquement
- Compatible avec `errors.Is` / `errors.As` stdlib (chaîne de wrapping préservée)
- Codes d'erreur versionnable et contractuels côté client

### **Cons**
- Type custom à maintenir (pas de bibliothèque derrière)
- Les développeurs doivent utiliser les constructeurs `kernel/errors` au lieu de `fmt.Errorf` natif
- Légère surcharge mémoire (stack de frames à chaque AppError)
