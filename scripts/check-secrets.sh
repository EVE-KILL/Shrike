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
#
# Two lists, scanned against different diffs, because .env.example needs
# different treatment from everything else. Its whole job is to document the
# same keys, so it legitimately repeats non-credential values — hostnames,
# paths, a Memgraph bolt:// URL. Scanning it for those made every edit to it
# unlandable, which is a check that trains you to reach for --no-verify.
#
#   values      — every private KEY=value. Not scanned in .env.example.
#   credentials — the password field of a URL-style value. Scanned everywhere,
#                 including .env.example, because a real password there is
#                 wrong no matter what the file is for.
#
# A value already present in .env.example is public configuration. Remove those
# values from the private list so documentation can name public buckets, hosts,
# and paths without weakening the credential scan.
env_files=()
for f in .env .env.*; do
  [ -f "$f" ] || continue
  case "$f" in *.example) continue ;; esac
  env_files+=("$f")
done

collect() {
  # $1: sed expression selecting the part of KEY=value to extract.
  [ ${#env_files[@]} -gt 0 ] || return 0
  sed -nE "$1" "${env_files[@]}" | sed 's/^"//; s/"$//' | awk 'length($0) >= 12' | sort -u
}

values=$(collect 's/^[A-Z_0-9]+=(.*)$/\1/p')
if [ -f .env.example ]; then
  documented_values=$(
    sed -nE 's/^[# ]*[A-Z_0-9]+=(.*)$/\1/p' .env.example \
      | sed 's/^"//; s/"$//' \
      | awk 'length($0) >= 12' \
      | sort -u
  )
  values=$(comm -23 \
    <(printf '%s\n' "$values" | sort -u) \
    <(printf '%s\n' "$documented_values" | sort -u))
fi
credentials=$(collect 's#^[A-Z_0-9]+=[a-z]+://[^:/@]+:([^@]+)@.*#\1#p')

diff_all=$(git diff --cached --diff-filter=ACM -U0 || true)
diff_code=$(git diff --cached --diff-filter=ACM -U0 -- . ':(exclude).env.example' || true)

# report_match <secret-list> <diff> — flags the first hit and stops. A DSN
# matches both as a whole value and as its extracted password, so continuing
# would print the same finding twice.
report_match() {
  local list="$1" text="$2" secret
  [ -n "$list" ] || return 0
  while IFS= read -r secret; do
    [ -z "$secret" ] && continue
    if printf '%s' "$text" | grep -F -- "$secret" >/dev/null 2>&1; then
      red "Refusing to commit: a value from a local .env appears in the staged diff"
      # Report where, never what.
      printf '%s' "$text" | grep -nF -- "$secret" | head -3 \
        | sed -E 's/^([0-9]+):.*/    diff line \1 (value redacted)/'
      fail=1
      return 0
    fi
  done <<< "$list"
}

report_match "$credentials" "$diff_all"
if [ "$fail" -eq 0 ]; then
  report_match "$values" "$diff_code"
fi

if [ "$fail" -eq 0 ]; then
  exit 0
fi

echo
red "Commit blocked by the secret guard."
echo "    If this is a false positive, the offending value is genuinely present"
echo "    in a local .env — rotate it or remove it from the staged change."
exit 1
