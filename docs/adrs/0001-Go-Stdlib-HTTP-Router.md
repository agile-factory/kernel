# _ADR #0001:_ Go Standard Library pour le Routing HTTP

**date:** `2026 Apr 28`

---

## **Status:** `ACCEPTED`

---

## **Context**

Le socle Agile Factory doit exposer une API REST. Le choix du routeur HTTP est structurant sur le long terme car il impacte toutes les couches delivery. Les frameworks populaires de l'écosystème Go (Gin, Echo, Fiber, Chi) offrent un démarrage plus rapide mais introduisent du couplage.

Depuis **Go 1.22**, le package standard `net/http` supporte nativement le **routing par pattern** avec paramètres de chemin (`GET /users/{id}`) et la différentiation par méthode HTTP, ce qui comble le principal manque historique de la stdlib.

---

## **Decision**

Utiliser **exclusivement `net/http`** de la stdlib Go pour le routing et le serveur HTTP. Aucun framework tiers n'est introduit dans la couche delivery.

Les routes sont enregistrées via `mux.HandleFunc("METHOD /path/{param}", handler)`. Les middlewares sont construits manuellement sous forme de `func(http.Handler) http.Handler`, compatibles avec n'importe quel routeur Go.

---

## **Alternatives Considered**

### Gin / Echo / Fiber
- ✅ Démarrage plus rapide (~1 semaine de gain)
- ✅ Helpers de binding/validation intégrés
- ❌ `*gin.Context` s'infiltre dans toutes les couches → brise la Clean Architecture
- ❌ Lock-in fort : changement = réécriture de toute la couche delivery
- ❌ Fiber utilise `fasthttp`, incompatible avec l'écosystème stdlib

**Verdict : rejeté**

### Chi Router
- ✅ 100% compatible stdlib (`func(w, r)`)
- ✅ Sub-routers et composition de middleware plus ergonomiques
- ❌ Dépendance externe supplémentaire pour un gain marginal depuis Go 1.22
- **Verdict : différé** — migration triviale si les besoins l'exigent

---

## **Consequences**

### **Pros**
- Aucun lock-in framework : le code delivery est portable vers n'importe quel routeur Go
- Zéro dépendance transitive liée au routing
- Compatibilité garantie à long terme (stdlib Go 1.x)
- Signatures `http.HandlerFunc` standard, documentées et stables

### **Cons**
- Validation et binding HTTP doivent être écrits manuellement (atténué par `kernel/validator` et `kernel/response`)
- Code delivery légèrement plus verbeux qu'avec un framework
- Pas de sub-routers natifs (acceptable pour la v1 avec le préfixe `/api/v1/` en dur)
