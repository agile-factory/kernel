# _ADR #0005:_ Logging Fan-Out via `log/slog`

**date:** `2026 Apr 30`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

Un backend SaaS a des besoins de logging distincts selon le public :
- **Opérateurs** : logs structurés JSON pour ingestion par des outils (Loki, Datadog)
- **Développeurs** : logs lisibles en texte en développement local
- **Sécurité** : logs HTTP séparés pour l'accès aux endpoints (access log)
- **Incident response** : logs d'erreurs isolés pour triage rapide

Un seul handler syslog ne peut pas satisfaire ces besoins simultanément de manière propre.

---

## **Decision**

Implémenter un **fan-out logger** dans `kernel/logger/` utilisant `log/slog` (stdlib Go 1.21+) avec quatre destinations simultanées :

| Destination | Contenu | Format | Condition |
|---|---|---|---|
| `stdout` | Tous niveaux | Text (dev) / JSON (prod) | Toujours |
| `logs/app.log` | Tous niveaux, avec source | JSON | Toujours |
| `logs/error.log` | ERROR et au-dessus | JSON | Level >= ERROR |
| `logs/server.log` | Requêtes HTTP uniquement | JSON | Attribut `source=http` |

Le routage vers `server.log` repose sur un `filteredHandler` personnalisé qui inspecte l'attribut `source` de chaque record slog. Le middleware HTTP log avec `log.With("source", "http")`.

---

## **Alternatives Considered**

### Zap (uber-go/zap)
- ✅ Très performant (zéro allocation)
- ✅ Fan-out natif avec `zapcore.NewTee`
- ❌ Dépendance externe alors que `log/slog` stdlib couvre le besoin
- ❌ API différente de slog, migration vers slog complexe
- **Verdict : rejeté** — `slog` devient le standard Go, Zap aura un adaptateur slog

### Logrus
- ✅ Répandu, hooks pour fan-out
- ❌ En mode maintenance uniquement, auteur recommande migration vers slog
- **Verdict : rejeté**

### Log unique vers stdout, délégation à Docker/K8s
- ✅ 12-factor app strict
- ❌ Pas de séparation access log / error log sans middleware externe
- **Verdict : insuffisant** — les fichiers locaux accélèrent le développement et le debugging

---

## **Consequences**

### **Pros**
- Zéro dépendance externe : `log/slog` est stdlib depuis Go 1.21
- `slog.SetDefault()` permet à tout le code applicatif de logger sans importer `kernel/logger`
- Rotation externe possible (logrotate, Docker log drivers) sans modifier le code
- `server.log` dédié permet un grep rapide des access logs sans bruit

### **Cons**
- Les fichiers log doivent être montés en volume dans Docker (sinon perdus à chaque restart)
- `filteredHandler` est du code custom à maintenir
- En environnement lambda/FaaS, les fichiers log ne sont pas pertinents (stdout suffisant)
