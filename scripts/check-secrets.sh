#!/usr/bin/env bash
# Refuse to commit secrets.
#
# Two independent checks, because the obvious one is not sufficient:
#
#   1. No .env file may be staged. They are gitignored, but `git add -f` and a
#      badly-scoped `git add` both defeat that.
#
#   2. No staged line may contain a value that appears in a local .env. This is
#      the check that matters: the realistic mistake is not committing .env
#      itself, it is pasting a working DSN into a test, a comment, or a doc.
#
# Exit 1 blocks the commit.
set -uo pipefail

fail=0
red() { printf '\033[31m%s\033[0m\n' "$1"; }

# --- 1. staged .env files -----------------------------------------------------
staged_env=$(git diff --cached --name-only --diff-filter=ACM \
  | grep -E '(^|/)\.env($|\.)' \
  | grep -v '\.env\.example$' || true)

if [ -n "$staged_env" ]; then
  red "Refusing to commit: environment files are staged"
  echo "$staged_env" | sed 's/^/    /'
  echo "    These hold production credentials. Unstage with: git restore --staged <file>"
  fail=1
fi

# --- 2. staged content matching a local secret --------------------------------
# Collect candidate values from every local .env. Short values are ignored: a
# port or a 'true' would match everything and make this check useless noise.
secrets=$(
  for f in .env .env.*; do
    [ -f "$f" ] || continue
    case "$f" in *.example) continue ;; esac
    # KEY=value -> value, plus the password field of any URL-style value.
    sed -nE 's/^[A-Z_0-9]+=(.*)$/\1/p' "$f"
    sed -nE 's#^[A-Z_0-9]+=[a-z]+://[^:/@]+:([^@]+)@.*#\1#p' "$f"
  done | sed 's/^"//; s/"$//' | awk 'length($0) >= 12' | sort -u
)

if [ -n "$secrets" ]; then
  diff_text=$(git diff --cached --diff-filter=ACM -U0 || true)
  while IFS= read -r secret; do
    [ -z "$secret" ] && continue
    # Fixed-string match against added lines only.
    if printf '%s' "$diff_text" | grep -F -- "$secret" >/dev/null 2>&1; then
      # Report once and stop. A DSN matches both as a whole value and as its
      # extracted password, so continuing would print the same finding twice.
      red "Refusing to commit: a value from a local .env appears in the staged diff"
      # Report where, never what.
      printf '%s' "$diff_text" | grep -nF -- "$secret" | head -3 \
        | sed -E 's/^([0-9]+):.*/    diff line \1 (value redacted)/'
      fail=1
      break
    fi
  done <<< "$secrets"
fi

if [ "$fail" -eq 0 ]; then
  exit 0
fi

echo
red "Commit blocked by the secret guard."
echo "    If this is a false positive, the offending value is genuinely present"
echo "    in a local .env — rotate it or remove it from the staged change."
exit 1
