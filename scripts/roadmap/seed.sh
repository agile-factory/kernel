#!/usr/bin/env bash
# Seed the agile-factory/kernel roadmap project (#2) with the items
# listed in items.json. Each item maps to one GitHub issue in
# AgileFactory/kernel + one project item attached to it, with Status
# and Start/Target date fields set.
#
# Idempotent: re-runs skip issues whose exact title already exists.
#
# Prerequisites:
#   - gh CLI authenticated with `project` scope:
#       gh auth refresh -s project
#   - jq installed
#   - Run from the repo root or anywhere — the script resolves its own dir.
#
# Usage:
#   bash scripts/roadmap/seed.sh           # full run
#   DRY_RUN=1 bash scripts/roadmap/seed.sh # show what would happen, no writes
#
set -euo pipefail

REPO="${REPO:-AgileFactory/kernel}"
ORG="${ORG:-agile-factory}"
PROJECT_NUMBER="${PROJECT_NUMBER:-2}"
DRY_RUN="${DRY_RUN:-0}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$SCRIPT_DIR/items.json"

bold()  { printf '\033[1m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
red()   { printf '\033[31m%s\033[0m\n' "$*" >&2; }
yellow(){ printf '\033[33m%s\033[0m\n' "$*"; }
dim()   { printf '\033[2m%s\033[0m\n' "$*"; }

[[ -f $MANIFEST ]] || { red "manifest not found: $MANIFEST"; exit 1; }
command -v jq >/dev/null || { red "jq required"; exit 1; }
command -v gh >/dev/null || { red "gh CLI required"; exit 1; }

# ─── Step 1: discover project + field IDs ──────────────────────────────────

bold "→ Discovering project $PROJECT_NUMBER under org $ORG"
PROJECT_JSON=$(gh project view "$PROJECT_NUMBER" --owner "$ORG" --format json 2>&1) || {
  red "Failed to read project. If you see a scope error, run:"
  red "  gh auth refresh -s project"
  exit 1
}
PROJECT_ID=$(echo "$PROJECT_JSON" | jq -r '.id')
dim "  project id: $PROJECT_ID"

bold "→ Discovering fields"
FIELDS_JSON=$(gh project field-list "$PROJECT_NUMBER" --owner "$ORG" --format json --limit 50)

# Pull common roadmap-template fields. Names default to GitHub's roadmap
# template: "Status", "Start date", "Target date". Override via env if
# the project uses different names.
STATUS_FIELD_NAME="${STATUS_FIELD_NAME:-Status}"
START_DATE_FIELD_NAME="${START_DATE_FIELD_NAME:-Start date}"
TARGET_DATE_FIELD_NAME="${TARGET_DATE_FIELD_NAME:-Target date}"

field_id() {
  local name="$1"
  echo "$FIELDS_JSON" | jq -r --arg n "$name" '.fields[] | select(.name == $n) | .id'
}

# Single-select option IDs (Status field options like "Todo", "In Progress", "Done"...)
status_option_id() {
  local opt_name="$1"
  echo "$FIELDS_JSON" | jq -r --arg field "$STATUS_FIELD_NAME" --arg opt "$opt_name" \
    '.fields[] | select(.name == $field) | .options[]? | select(.name == $opt) | .id'
}

STATUS_FIELD_ID=$(field_id "$STATUS_FIELD_NAME")
START_FIELD_ID=$(field_id "$START_DATE_FIELD_NAME")
TARGET_FIELD_ID=$(field_id "$TARGET_DATE_FIELD_NAME")

[[ -z $STATUS_FIELD_ID ]] && red "  ⚠ field '$STATUS_FIELD_NAME' not found — status will be skipped"
[[ -z $START_FIELD_ID  ]] && red "  ⚠ field '$START_DATE_FIELD_NAME' not found — start date will be skipped"
[[ -z $TARGET_FIELD_ID ]] && red "  ⚠ field '$TARGET_DATE_FIELD_NAME' not found — target date will be skipped"

dim "  Status field id     : ${STATUS_FIELD_ID:-(missing)}"
dim "  Start date field id : ${START_FIELD_ID:-(missing)}"
dim "  Target date field id: ${TARGET_FIELD_ID:-(missing)}"

# Map our manifest status keys to GitHub status option names. Adjust if
# the project uses different option names (e.g. "In progress" vs "In Progress").
declare -A STATUS_OPT_MAP=(
  [done]="Done"
  [in_progress]="In progress"
  [todo]="Todo"
  [deferred]=""
)

# Pre-resolve option ids; warn if any are missing.
declare -A STATUS_OPT_ID=()
for key in "${!STATUS_OPT_MAP[@]}"; do
  opt_name="${STATUS_OPT_MAP[$key]}"
  id=$(status_option_id "$opt_name")
  if [[ -z $id ]]; then
    yellow "  ⚠ status option '$opt_name' (for $key) not found in project — items with status=$key will be skipped on status assignment"
  fi
  STATUS_OPT_ID[$key]="$id"
done

# ─── Step 2: look up existing issues for idempotence ──────────────────────

bold "→ Loading existing issues (idempotence check)"
# Cache issue titles + URLs into a tmpfile. We page through all issues
# (open + closed) since done items may be closed already.
EXISTING_TMP=$(mktemp)
trap 'rm -f "$EXISTING_TMP"' EXIT

gh issue list --repo "$REPO" --state all --limit 500 --json title,url \
  > "$EXISTING_TMP" 2>/dev/null

issue_url_for_title() {
  local title="$1"
  jq -r --arg t "$title" '.[] | select(.title == $t) | .url' "$EXISTING_TMP" | head -1
}

# ─── Step 3: iterate manifest ─────────────────────────────────────────────

TOTAL=$(jq '. | length' "$MANIFEST")
bold "→ Processing $TOTAL roadmap items"

created=0
skipped=0
errors=0
i=0

while IFS= read -r row; do
  i=$((i + 1))
  title=$(jq -r '.title'       <<<"$row")
  body=$( jq -r '.body'        <<<"$row")
  status=$(jq -r '.status'     <<<"$row")
  start=$(jq -r '.start_date // ""' <<<"$row")
  target=$(jq -r '.target_date // ""' <<<"$row")
  labels=$(jq -r '.labels // [] | join(",")' <<<"$row")
  category=$(jq -r '.category' <<<"$row")

  printf '\n[%2d/%d] %s\n' "$i" "$TOTAL" "$title"

  existing_url=$(issue_url_for_title "$title")

  if [[ -n $existing_url ]]; then
    dim "    ↻ issue exists: $existing_url (skipping create)"
    issue_url="$existing_url"
    skipped=$((skipped + 1))
  else
    if [[ $DRY_RUN == 1 ]]; then
      dim "    [DRY] would create issue + labels: $labels"
      continue
    fi
    if ! issue_url=$(gh issue create --repo "$REPO" \
        --title "$title" \
        --body  "$body" \
        ${labels:+--label "$labels"} 2>&1); then
      red "    ✗ failed to create issue: $issue_url"
      errors=$((errors + 1))
      continue
    fi
    # gh issue create echoes the URL on success
    issue_url=$(echo "$issue_url" | grep -oE 'https://github.com/[^[:space:]]+' | head -1)
    green "    ✓ created $issue_url"
    created=$((created + 1))
  fi

  if [[ $DRY_RUN == 1 ]]; then
    dim "    [DRY] would attach to project + set status=$status start=$start target=$target"
    continue
  fi

  # Attach to project. gh project item-add is idempotent at the server level.
  item_json=$(gh project item-add "$PROJECT_NUMBER" --owner "$ORG" --url "$issue_url" --format json 2>&1) || {
    red "    ✗ failed to attach to project: $item_json"
    errors=$((errors + 1))
    continue
  }
  item_id=$(echo "$item_json" | jq -r '.id')
  dim "    ↳ project item id: $item_id"

  # Set Status (single-select)
  opt_id="${STATUS_OPT_ID[$status]:-}"
  if [[ -n $STATUS_FIELD_ID && -n $opt_id ]]; then
    gh project item-edit \
      --id "$item_id" \
      --field-id "$STATUS_FIELD_ID" \
      --project-id "$PROJECT_ID" \
      --single-select-option-id "$opt_id" >/dev/null
    dim "    ↳ status set to ${STATUS_OPT_MAP[$status]}"
  fi

  # Set Start date
  if [[ -n $START_FIELD_ID && -n $start ]]; then
    gh project item-edit \
      --id "$item_id" \
      --field-id "$START_FIELD_ID" \
      --project-id "$PROJECT_ID" \
      --date "$start" >/dev/null
    dim "    ↳ start: $start"
  fi

  # Set Target date
  if [[ -n $TARGET_FIELD_ID && -n $target ]]; then
    gh project item-edit \
      --id "$item_id" \
      --field-id "$TARGET_FIELD_ID" \
      --project-id "$PROJECT_ID" \
      --date "$target" >/dev/null
    dim "    ↳ target: $target"
  fi

  # Close the issue if status is Done (clean visual state in GitHub)
  if [[ $status == done ]]; then
    gh issue close "$issue_url" --reason completed >/dev/null 2>&1 || true
  fi

done < <(jq -c '.[]' "$MANIFEST")

# ─── Summary ──────────────────────────────────────────────────────────────

bold ""
bold "──────────────────────────────────────────────"
green "  ✓ created : $created"
yellow "  ↻ skipped: $skipped (already existed)"
[[ $errors -gt 0 ]] && red "  ✗ errors  : $errors" || dim   "  ✗ errors  : 0"
bold "──────────────────────────────────────────────"
