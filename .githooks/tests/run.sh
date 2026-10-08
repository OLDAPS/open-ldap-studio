#!/usr/bin/env bash
# Tests for the hook helpers and the commit-msg hook. Plain bash so it runs
# anywhere the hooks do; CI runs it in the `hooks` job.
set -uo pipefail

here=$(cd "$(dirname "$0")" && pwd)
hooks=$(dirname "$here")
root=$(dirname "$hooks")
. "$hooks/lib.sh"

pass=0; failed=0
ok()  { pass=$((pass+1)); }
bad() { failed=$((failed+1)); printf 'FAIL: %s\n' "$*" >&2; }
expect() { # expect <0|1> <description> <command...>
  local want="$1" desc="$2"; shift 2
  "$@" >/dev/null 2>&1; local got=$?
  if [ "$got" -eq "$want" ]; then ok; else bad "$desc (exit $got, want $want)"; fi
}

# --- valid_subject -----------------------------------------------------------
for s in 'feat: add search' 'fix(secrets): copy on get' 'ci: gate lint' 'feat!: drop v1' \
         'refactor(bridge)!: rename' 'chore: bump' 'Merge branch main' 'Revert "feat: x"' 'fixup! fix: y'; do
  expect 0 "accepts: $s" valid_subject "$s"
done
for s in '' 'bad message' 'Feat: capital' 'feat:no space' 'feat(): empty scope' 'feature: wrong type' 'fix(Scope): upper' 'feat: '; do
  expect 1 "rejects: '$s'" valid_subject "$s"
done

# --- commit-msg hook ---------------------------------------------------------
msg=$(mktemp)
printf 'fix(x): ok\n\nbody\n' >"$msg";                 expect 0 "hook accepts a good message" "$hooks/commit-msg" "$msg"
printf '# comment\nnot conventional\n' >"$msg";        expect 1 "hook rejects a bad message"  "$hooks/commit-msg" "$msg"
printf '# comment first\nci: after a comment\n' >"$msg"; expect 0 "hook skips comment lines"   "$hooks/commit-msg" "$msg"
rm -f "$msg"

# --- is_zero_sha -------------------------------------------------------------
expect 0 "zero sha" is_zero_sha 0000000000000000000000000000000000000000
expect 1 "real sha" is_zero_sha 1234567890abcdef1234567890abcdef12345678

# --- push_changed_files against a throw-away repo ----------------------------
tmp=$(mktemp -d)
(
  cd "$tmp" && git init -q -b main . \
    && git config user.email t@example.com && git config user.name t \
    && echo a >a.txt && git add . && git commit -qm "chore: a" \
    && git checkout -qb feature \
    && echo b >b.txt && echo c >>a.txt && git add . && git commit -qm "feat: b" \
    && git rm -q a.txt && git commit -qm "chore: drop a"
) >/dev/null 2>&1
zero=0000000000000000000000000000000000000000
tip=$(git -C "$tmp" rev-parse HEAD)
first=$(git -C "$tmp" rev-parse HEAD~1)

new_branch=$(cd "$tmp" && push_changed_files "$tip" "$zero" | sort | tr '\n' ' ')
[ "$new_branch" = "a.txt b.txt " ] && ok || bad "new branch lists files since the fork point (got: $new_branch)"
only_last=$(cd "$tmp" && push_changed_files "$tip" "$first" | sort | tr '\n' ' ')
[ "$only_last" = "a.txt " ] && ok || bad "existing branch lists only the new commits, deletions included (got: $only_last)"
unknown_remote=$(cd "$tmp" && push_changed_files "$tip" 1111111111111111111111111111111111111111 | sort | tr '\n' ' ')
[ "$unknown_remote" = "a.txt b.txt " ] && ok || bad "an unknown remote tip falls back to the fork point (got: $unknown_remote)"
rm -rf "$tmp"

# --- the lint version is pinned in one place and CI matches it ---------------
make_ver=$(sed -n 's/^GOLANGCI_LINT_VERSION[[:space:]]*:\{0,1\}=[[:space:]]*//p' "$root/Makefile")
ci_ver=$(sed -n 's/^[[:space:]]*version: \(v2\.[0-9.]*\)$/\1/p' "$root/.github/workflows/ci.yml" | sort -u)
if [ -n "$make_ver" ] && [ "$make_ver" = "$ci_ver" ]; then ok; else bad "Makefile GOLANGCI_LINT_VERSION ($make_ver) and ci.yml ($ci_ver) differ"; fi

printf '%d passed, %d failed\n' "$pass" "$failed"
[ "$failed" -eq 0 ]
