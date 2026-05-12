# _ADR #0006:_ Migrations SQL Versionnées avec `golang-migrate`

**date:** `2026 Apr 30`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

Le schéma de base de données évolue avec chaque feature. Sans système de migrations versionnées, les environnements (dev, staging, prod) divergent et les rollbacks sont dangereux. Le projet a besoin d'une solution qui :
- Versionne chaque changement de schéma sous forme de fichiers SQL lisibles
- S'exécute automatiquement au démarrage en développement
- Peut être exécutée manuellement en production (contrôle explicite)
- Supporte les rollbacks via des fichiers `.down.sql`

---

## **Decision**

Utiliser **`golang-migrate`** comme outil de migration avec des fichiers SQL bruts versionnés dans `migrations/`.

Convention de nommage : `YYYYMMDDHHMI_description.up.sql` / `.down.sql`.

```
migrations/
├── 202604301917_create_tenants_table.up.sql
├── 202604301917_create_tenants_table.down.sql
├── 202604301918_create_users_table.up.sql
├── 202605010001_create_memberships_table.up.sql
├── 202605010002_drop_users_tenant_id.up.sql
└── 202605010003_create_audit_logs_table.up.sql
```

Le `kernel/migrator/` encapsule l'exécution de `golang-migrate` et est appelé au démarrage de l'application en développement.

---

## **Alternatives Considered**

### SQL brut sans outil (scripts manuels)
- ✅ Aucune dépendance
- ❌ Pas de tracking de version (quelle migration a été appliquée ?)
- ❌ Pas de rollback automatisé
- **Verdict : rejeté**

### GORM Auto-Migrate
- ✅ Génération automatique depuis les structs Go
- ❌ Migrations non lisibles, difficiles à auditer
- ❌ GORM est un ORM complet, incompatible avec la philosophie stdlib du projet
- ❌ Pas de rollback
- **Verdict : rejeté**

### Atlas (ariga.io/atlas)
- ✅ Déclaratif, puissant, drift detection
- ❌ Outil plus récent, moins mûr pour la prod
- ❌ Dépendance plus lourde
- **Verdict : différé** — à réévaluer pour la v2 si besoin de drift detection

---

## **Consequences**

### **Pros**
- Fichiers SQL lisibles par n'importe quel DBA sans connaissance Go
- Rollback propre via `.down.sql`
- Historique Git complet des changements de schéma
- Intégrable dans les pipelines CI/CD

### **Cons**
- Les fichiers `.down.sql` doivent être maintenus manuellement (pas de génération automatique)
- Migrations destructives (DROP COLUMN, DROP TYPE) nécessitent attention particulière en prod
- `golang-migrate` est une dépendance externe (bien que mature et largement adoptée)
