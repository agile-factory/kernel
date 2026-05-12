# _ADR #0008:_ Injection de Dépendances Manuelle (Container Explicite)

**date:** `2026 Apr 28`

---

## **Status:** `ACCEPTED`

---

## **Context**

Toute application Clean Architecture nécessite un mécanisme pour assembler les dépendances : repos → use cases → handlers. En Go, plusieurs approches existent : DI manuelle (code explicite), génération de code (`google/wire`), ou des conteneurs réflectifs (`uber-go/dig`, `samber/do`).

Le RFC-0002 identifie ce choix comme une **question ouverte** à trancher après expérience. Après avoir implémenté les plateformes `auth`, `tenant`, `membership` et `audit`, la complexité réelle du wiring peut être évaluée.

---

## **Decision**

Utiliser un **conteneur DI manuel** dans `internal/bootstrap/container.go` :

```go
func (c *Container) setupRoutes(db *pgxpool.Pool) {
    // Audit
    auditRepo := auditinfra.NewPostgresAuditRepository(db)
    logEventUC := auditusecase.NewLogEventUseCase(auditRepo)

    // Membership
    membershipRepo := memberinfra.NewPostgresMembershipRepository(db)
    inviteUC := memberusecase.NewInviteMemberUseCase(membershipRepo, tenantRepo, logEventUC)
    // ...
}
```

Toutes les dépendances sont construites dans l'ordre topologique manuellement. Aucune réflexion, aucune génération de code.

---

## **Alternatives Considered**

### `google/wire` (génération de code)
- ✅ Graphe de dépendances vérifié à la compilation
- ✅ Élimine les erreurs d'ordre d'initialisation
- ❌ Outil externe à installer (`go generate`)
- ❌ Courbe d'apprentissage pour les nouveaux développeurs
- ❌ Fichiers générés dans le dépôt (bruit)
- **Verdict : différé** — à réévaluer si le conteneur dépasse ~150 lignes

### `uber-go/dig` ou `samber/do` (réflexion)
- ✅ Enregistrement déclaratif des fournisseurs
- ❌ Erreurs de câblage détectées au runtime, pas à la compilation
- ❌ Dépendance externe
- ❌ Debug difficile (`dig` produit des traces complexes)
- **Verdict : rejeté** — la sécurité à la compilation prime sur l'ergonomie

---

## **Consequences**

### **Pros**
- Aucune dépendance externe pour le DI
- Le câblage est entièrement visible et debuggable dans un seul fichier
- Erreurs de type détectées à la compilation Go
- Onboarding simplifié : un développeur comprend `container.go` sans outil tiers

### **Cons**
- Le fichier `container.go` peut devenir long à mesure que les plateformes et modules s'accumulent
- L'ordre d'initialisation est à la charge du développeur (pas de résolution automatique de graphe)
- Toute nouvelle dépendance nécessite une modification manuelle du conteneur
