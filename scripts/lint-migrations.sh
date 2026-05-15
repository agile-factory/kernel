#!/usr/bin/env bash
# Migration filename linter — enforces the format documented in RFC-0038 §4.
#
# Every *.sql file under {platform/*,kernel/*}/migrations/ must match
# `NNNN_<description>.{up,down}.sql` (4-digit numeric version, snake_case
# description, .up.sql or .down.sql extension).
#
# Additionally, every .up.sql must have a matching .down.sql.
set -euo pipefail

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
bold()  { printf '\033[1m%s\033[0m\n' "$*"; }

errs=0

bold "→ Locating migration directories"
mapfile -t dirs < <(find platform kernel -type d -name migrations 2>/dev/null | sort)
if [[ ${#dirs[@]} -eq 0 ]]; then
  red "  ✗ no migrations directory found — repo layout out-of-spec."
  exit 1
fi
for d in "${dirs[@]}"; do
  echo "  $d"
done

re='^[0-9]{4}_[a-z0-9_]+\.(up|down)\.sql$'

bold "→ Checking file naming (NNNN_<desc>.{up,down}.sql)"
for d in "${dirs[@]}"; do
  while IFS= read -r f; do
    base=$(basename "$f")
    if [[ ! $base =~ $re ]]; then
      red "  ✗ $f does not match NNNN_<desc>.{up,down}.sql"
      errs=$((errs + 1))
    fi
  done < <(find "$d" -maxdepth 1 -type f -name '*.sql' | sort)
done
if [[ $errs -eq 0 ]]; then
  green "  ✓ all migration filenames OK."
fi

bold "→ Checking every .up.sql has a matching .down.sql"
for d in "${dirs[@]}"; do
  while IFS= read -r up; do
    down="${up%.up.sql}.down.sql"
    if [[ ! -f $down ]]; then
      red "  ✗ missing rollback: $down (paired with $up)"
      errs=$((errs + 1))
    fi
  done < <(find "$d" -maxdepth 1 -type f -name '*.up.sql' | sort)
done

bold "→ Checking version numbers are unique per package"
for d in "${dirs[@]}"; do
  dupes=$(
    find "$d" -maxdepth 1 -type f -name '*.up.sql' \
      | xargs -n1 basename \
      | awk -F_ '{print $1}' \
      | sort | uniq -d
  )
  if [[ -n $dupes ]]; then
    red "  ✗ duplicate version numbers in $d: $dupes"
    errs=$((errs + 1))
  fi
done
if [[ $errs -eq 0 ]]; then
  green "  ✓ versions are unique per package."
fi

if [[ $errs -gt 0 ]]; then
  red ""
  red "✗ $errs migration filename violation(s)."
  exit 1
fi

green ""
green "✓ All migration files conform to RFC-0038."
