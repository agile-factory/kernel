#!/usr/bin/env bash
# RFC-0038 conventions linter — runs in CI.
#
# Enforces:
#   - No `init()` functions in platform/ or kernel/ packages.
#   - No `os.Getenv` usage in platform/ packages (config flows via Config struct).
#   - No imports of any path containing `/cmd/` or `/examples/` from platform/.
#   - No `internal/` paths anywhere (kernel is a library — internal is forbidden).
set -euo pipefail

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
bold()  { printf '\033[1m%s\033[0m\n' "$*"; }

errs=0

bold "→ Checking: no init() in platform/ or kernel/"
# Match `func init()` at the start of a line. Exclude _test.go (init in tests is acceptable for fixtures).
if grep -rn --include='*.go' --exclude='*_test.go' '^func init()' platform/ kernel/ 2>/dev/null; then
  red "  ✗ found init() functions in platform/ or kernel/ — see RFC-0038 §5."
  errs=$((errs + 1))
else
  green "  ✓ none."
fi

bold "→ Checking: no os.Getenv in platform/"
if grep -rn --include='*.go' --exclude='*_test.go' 'os\.Getenv\b' platform/ 2>/dev/null; then
  red "  ✗ found os.Getenv in platform/ — config must flow through the Module Config struct."
  errs=$((errs + 1))
else
  green "  ✓ none."
fi

bold "→ Checking: no platform/ → cmd/ or examples/ imports"
# Detect import lines that point at cmd/ or examples/ from inside platform/.
if grep -rEn --include='*.go' '"github\.com/AgileFactory/kernel/(cmd|examples)/' platform/ 2>/dev/null; then
  red "  ✗ platform/ packages must be leaves; no import of cmd/ or examples/."
  errs=$((errs + 1))
else
  green "  ✓ none."
fi

bold "→ Checking: no internal/ paths"
# The repo IS the library — everything is exported. internal/ defeats that.
if find . -type d -name internal -not -path './.git/*' 2>/dev/null | grep -q .; then
  red "  ✗ found an internal/ directory — kernel is a public library, internal/ is forbidden."
  find . -type d -name internal -not -path './.git/*'
  errs=$((errs + 1))
else
  green "  ✓ none."
fi

if [[ $errs -gt 0 ]]; then
  red ""
  red "✗ $errs convention violation(s)."
  exit 1
fi

green ""
green "✓ All RFC-0038 conventions OK."
