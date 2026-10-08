#!/usr/bin/env bash
# Helpers shared by the hooks and their tests. Sourced, never executed.

if [ -t 2 ]; then
  _bold=$'\033[1m'; _yellow=$'\033[33m'; _red=$'\033[31m'; _reset=$'\033[0m'
else
  _bold=''; _yellow=''; _red=''; _reset=''
fi

say()  { printf '%s[hooks]%s %s\n' "$_bold" "$_reset" "$*" >&2; }
warn() { printf '%s[hooks] warning:%s %s\n' "$_yellow" "$_reset" "$*" >&2; }
fail() { printf '%s[hooks] %s%s\n' "$_red" "$*" "$_reset" >&2; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

# Commit types accepted by the pr-title job in .github/workflows/ci.yml.
CONVENTIONAL_TYPES='feat|fix|perf|revert|refactor|docs|build|ci|test|style|chore'

is_zero_sha() { [[ "$1" =~ ^0+$ ]]; }

# Succeeds when a commit subject is acceptable: a conventional commit, or one
# of the subjects git itself generates.
valid_subject() {
  local subject="$1"
  case "$subject" in
    "Merge "*|'Revert "'*|"fixup! "*|"squash! "*|"amend! "*) return 0 ;;
  esac
  [[ "$subject" =~ ^($CONVENTIONAL_TYPES)(\([a-z0-9._/-]+\))?!?:\ .+ ]]
}

# The commit a new branch forked from, so a first push is judged by the whole
# branch rather than by nothing. Falls back to the empty tree when there is no
# main to compare against.
fork_base() {
  local head="$1" ref
  for ref in origin/main origin/master main master; do
    if git rev-parse --verify --quiet "$ref^{commit}" >/dev/null; then
      git merge-base "$ref" "$head" 2>/dev/null && return 0
    fi
  done
  git hash-object -t tree /dev/null
}

# Files a push would add or change: local_sha against what the remote already
# has, or against the fork point when the remote has never seen the branch (or
# its tip is not in this clone).
push_changed_files() {
  local local_sha="$1" remote_sha="$2" base
  if is_zero_sha "$remote_sha" || ! git cat-file -e "$remote_sha^{commit}" 2>/dev/null; then
    base=$(fork_base "$local_sha")
  else
    base="$remote_sha"
  fi
  git diff --name-only --diff-filter=ACMRD "$base" "$local_sha"
}

# The pinned linter is v2; CI and .golangci.yml (version: "2") need it, and v1
# fails on that config with an unhelpful schema error.
require_golangci_v2() {
  have golangci-lint || fail "golangci-lint is not installed. Run: make tools"
  local out major
  out=$(golangci-lint --version 2>&1 | head -1)
  if [[ "$out" =~ version\ v?([0-9]+) ]]; then
    major="${BASH_REMATCH[1]}"
  else
    fail "cannot read the golangci-lint version from: $out"
  fi
  [ "$major" -ge 2 ] || fail "golangci-lint v$major found, CI uses v2. Run: make tools"
}
