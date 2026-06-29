# _ADR #0014:_ Platform `billing` — Adapter pattern Stripe primary, Wave stub

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

RFC-0020 spécifiait un billing hybride : Stripe (international) + Wave / Orange Money / Free Money (CEDEAO) via adapters. L'implémentation P1-4 livre le squelette complet mais différencie volontairement les deux adapters :

- **Stripe** : implémenté avec HMAC-SHA256 webhook verification, CreateCheckoutSession, et gestion des events `checkout.session.completed` / `customer.subscription.deleted`
- **Wave** : interface satisfaite (`var _ domain.PaymentProcessor = (*WaveAdapter)(nil)`) mais toutes les méthodes retournent `errors.New("wave: ... not yet implemented — Wave API access required")`

---

## **Decision**

**Stripe est le processeur de paiement primaire pour P1.** Wave est une interface stub documentée qui permet :
1. De compiler et tester la couche d'abstraction sans credentials Wave
2. D'avoir un point d'entrée clair pour l'implémentation Wave quand les credentials sont disponibles
3. D'activer Wave sans modifier la signature du domaine

**Critère de déclenchement de l'implémentation Wave** : accès à un compte Wave Business API + environnement sandbox disponible pour les tests.

---

## **Alternatives Considered**

### Différer toute la billing à v2 (pas de wave.go)
- ✅ Moins de code à maintenir
- ❌ Laisse l'adapter pattern non prouvé ; les consommateurs ne savent pas qu'ils pourront brancher Wave
- **Verdict : rejeté** — le stub avec interface compile-time guard est la meilleure documentation

### Implémenter Wave maintenant avec Mock API
- ✅ Complet
- ❌ Sans sandbox Wave réelle, les tests seraient des mocks qui ne testent pas le vrai comportement
- **Verdict : différé** — YAGNI jusqu'à avoir les credentials

---

## **Consequences**

### **Pros**
- `platform/billing` est entièrement testable avec un `stubProcessor` in-memory
- La couche d'abstraction `domain.PaymentProcessor` est validée end-to-end via Stripe
- Les SaaS consommateurs CEDEAO peuvent démarrer avec Stripe et migrer vers Wave sans changer leur code

### **Cons**
- Wave n'est pas encore opérationnel — un SaaS ciblant exclusivement Wave/OM ne peut pas démarrer P1
- `platform/billing` n'a pas encore d'infrastructure Postgres (SubRepo, PlanRepo) — le wiring dans main.go est différé avec un commentaire explicatif
