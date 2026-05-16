# Roadmap seeding

Scripts qui peuplent le project board `agile-factory/projects/2` (template roadmap) avec les jalons + RFCs + ADRs du kernel.

## Fichiers

| Fichier | Rôle |
|---|---|
| `items.json` | Manifeste — 39 items (4 jalons + 25 RFCs + 10 ADRs) avec status, dates start/target, labels. |
| `seed.sh` | Itère le manifeste, crée les issues GitHub dans `AgileFactory/kernel`, les attache au project, et set les champs Status / Start / Target. Idempotent. |

## Prérequis

```bash
# Le token doit avoir le scope project (interactive, ouvre navigateur)
gh auth refresh -s project

# Vérifier
gh auth status
gh project view 2 --owner agile-factory
```

## Usage

```bash
# Dry-run (aucune écriture)
DRY_RUN=1 bash scripts/roadmap/seed.sh

# Run réel
bash scripts/roadmap/seed.sh
```

Le script :
1. Découvre l'ID du project + IDs des fields `Status`, `Start date`, `Target date`.
2. Récupère la liste des issues existantes (open + closed) pour idempotence par titre exact.
3. Pour chaque item du manifeste :
   - Skippe si une issue avec le même titre existe.
   - Sinon, crée l'issue GitHub avec labels.
   - Attache au project (`gh project item-add`).
   - Set Status (single-select), Start date, Target date.
   - Ferme l'issue si `status == done` (état visuel propre).

## Overrides

| Variable env | Défaut | Quand l'utiliser |
|---|---|---|
| `REPO` | `AgileFactory/kernel` | Repo cible des issues. |
| `ORG` | `agile-factory` | Org propriétaire du project. |
| `PROJECT_NUMBER` | `2` | Numéro du project board. |
| `STATUS_FIELD_NAME` | `Status` | Si le project utilise un nom différent. |
| `START_DATE_FIELD_NAME` | `Start date` | Idem. |
| `TARGET_DATE_FIELD_NAME` | `Target date` | Idem. |
| `DRY_RUN` | `0` | `1` = no-op, juste log. |

## Mapping status

| Manifest | → option GitHub |
|---|---|
| `done` | `Done` |
| `in_progress` | `In Progress` |
| `todo` | `Todo` |
| `deferred` | `Backlog` |

Si une option n'existe pas dans le project, l'item est créé mais le Status reste vide (warning loggué).

## Re-runs

`seed.sh` est idempotent **par titre exact d'issue**. Si tu modifies un titre dans `items.json` puis re-run, ça créera un doublon — supprime l'ancienne issue manuellement avant.

## Inventaire des items

```
4  milestones  (v0.1.0 ✅, v0.2.0, v0.3.0, v1.0.0)
10 ADRs        (0001-0010, tous done)
25 RFCs        (0002, 0003, 0004, 0005, 0006, 0007, 0008, 0009, 0010,
                0013, 0014, 0015, 0016, 0017, 0018, 0019, 0020, 0021,
                0022, 0023, 0024, 0025, 0026, 0028, 0038)
─────
39 items au total
```

Exclus volontairement : RFC-0001 (Flutter frontend, hors kernel), RFC-0011 + RFC-0012 (supersédées par 0019 + 0020).
