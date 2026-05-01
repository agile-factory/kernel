# _ADR #0002:_ Clean Architecture en Trois Niveaux (Kernel / Platform / Modules)

**date:** `2026 Apr 28`

---

## **Status:** `ACCEPTED`

---

## **Context**

Agile Factory a pour objectif de servir de socle à **plusieurs SaaS distincts**. Cette contrainte exige une séparation nette entre :
1. Les utilitaires purement techniques (config, logs, erreurs)
2. Les briques SaaS réutilisables (auth, tenant, billing)
3. La logique métier propre à chaque produit

Sans frontière explicite, les dépendances entre ces catégories dégénèrent en couplages qui empêchent la réutilisation.

---

## **Decision**

L'architecture est organisée en **trois niveaux hiérarchiques** avec une règle de dépendance stricte : chaque niveau ne peut dépendre **que** du niveau inférieur, jamais l'inverse.

```
MODULES (modules/)     ← change par SaaS
    ↓ dépend de
PLATFORM (platform/)   ← stable entre SaaS
    ↓ dépend de
KERNEL (kernel/)       ← rarement modifié
```

Au sein de chaque feature (`auth/`, `tenant/`, `audit/`…), la Clean Architecture est appliquée avec quatre sous-couches :

```
domain/         ← entités, interfaces (ports), erreurs sentinelles
usecase/        ← logique applicative, orchestre domain
infrastructure/ ← implémentations concrètes (Postgres, JWT, bcrypt)
delivery/       ← handlers HTTP, DTOs, routes
```

La **règle de dépendance** : `domain` ne connaît ni `infrastructure` ni `delivery`. Les interfaces sont définies dans `domain`, implémentées dans `infrastructure`.

---

## **Alternatives Considered**

### Architecture en flat packages (par feature uniquement)
```
internal/auth/ | internal/tenant/ | internal/property/
```
- ✅ Plus simple, idiomatique Go pour petits projets
- ❌ Pas de distinction code réutilisable vs métier-spécifique
- ❌ Impossible d'extraire le socle pour un nouveau SaaS
- **Verdict : rejeté**

### Architecture par couche horizontale
```
internal/handlers/ | internal/services/ | internal/repositories/
```
- ✅ Familier pour les devs venant d'Express/NestJS
- ❌ `handlers/auth.go` et `handlers/property.go` dans le même package → couplages cachés
- ❌ Difficile à découper en sous-modules indépendants
- **Verdict : rejeté**

---

## **Consequences**

### **Pros**
- Le `platform/` est copiable tel quel d'un SaaS à l'autre
- La logique métier (`usecase/`) est testable sans DB ni HTTP
- Substituabilité : passage de Postgres à MongoDB sans toucher au `domain`
- Nouveaux développeurs peuvent comprendre une feature isolément

### **Cons**
- Overhead de structure pour de petits SaaS (\<5 entités)
- Verbosité : chaque feature nécessite ~4 fichiers minimum
- Courbe d'apprentissage pour les devs habitués aux frameworks convention-over-configuration
