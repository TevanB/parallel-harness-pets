#!/bin/bash
# Shared identity, cache, and rendering helpers. Sourced by the statusline, probe, and hooks.

BUDDY_HOME="${BUDDY_HOME:-$HOME/.claude/claude-buddy}"
BUDDY_CACHE="$BUDDY_HOME/cache"
BUDDY_STALE_AFTER=15

# name|frame prefix|frame suffix
BUDDY_SPECIES=(
  'otter|(|)'
  'cat|/|\'
  'fox|{|}'
  'frog|[|]'
  'bat|^|^'
  'rabbit|(\|/)'
  'moth|<|>'
  'axolotl|~(|)~'
  'gecko|-(|)-'
  'owl|((|))'
  'mouse|.(|).'
  'squid|*(|)*'
  'crab|>(|)<'
  'bear|o(|)o'
  'beetle|+(|)+'
  'koi|=(|)='
)

# Hashed independently of species, so two live sessions on the same creature still read apart at a glance.
BUDDY_PALETTE=(173 213 208 114 141 203 187 218 79 179 81 156)

# Char-fold hash, kept pure bash so identity never costs a subprocess.
buddy_hash() {
  local text="$1" i=0 code
  BUDDY_HASH=7
  while [ "$i" -lt "${#text}" ]; do
    code=0
    printf -v code '%d' "'${text:$i:1}" 2>/dev/null
    BUDDY_HASH=$(((BUDDY_HASH * 31 + code) % 1000003))
    i=$((i + 1))
  done
}

# Walk up for .git, honouring the gitdir: pointer file a worktree uses.
buddy_locate() {
  local dir="$1" line
  BUDDY_ROOT=""
  BUDDY_GITDIR=""
  BUDDY_BRANCH=""
  while [ -n "$dir" ] && [ "$dir" != "/" ]; do
    if [ -d "$dir/.git" ]; then
      BUDDY_ROOT="$dir"
      BUDDY_GITDIR="$dir/.git"
      break
    fi
    if [ -f "$dir/.git" ]; then
      IFS= read -r line <"$dir/.git"
      BUDDY_ROOT="$dir"
      BUDDY_GITDIR="${line#gitdir: }"
      break
    fi
    dir="${dir%/*}"
  done
  [ -n "$BUDDY_GITDIR" ] && [ -r "$BUDDY_GITDIR/HEAD" ] || return 1
  IFS= read -r line <"$BUDDY_GITDIR/HEAD"
  case "$line" in
  'ref: refs/heads/'*) BUDDY_BRANCH="${line#ref: refs/heads/}" ;;
  'ref: '*) BUDDY_BRANCH="${line#ref: }" ;;
  *) BUDDY_BRANCH="${line:0:7}" ;;
  esac
  [ -n "$BUDDY_BRANCH" ]
}

buddy_key() {
  local key="${1#/}"
  printf '%s' "${key//\//_}"
}

# Branch name decides the creature, so a branch always summons the same one.
buddy_species() {
  buddy_hash "$1"
  local entry="${BUDDY_SPECIES[$((BUDDY_HASH % ${#BUDDY_SPECIES[@]}))]}"
  IFS='|' read -r BUDDY_NAME BUDDY_PRE BUDDY_SUF <<<"$entry"
  buddy_hash "hue:$1"
  BUDDY_COLOR="${BUDDY_PALETTE[$((BUDDY_HASH % ${#BUDDY_PALETTE[@]}))]}"
}

# A half-written or hand-edited cache must degrade to zero, not leak "integer expression expected" into the status bar.
buddy_num() {
  case "$1" in
  '' | *[!0-9]*) printf '0' ;;
  *) printf '%s' "$1" ;;
  esac
}

buddy_read_state() {
  BUDDY_DIRTY=0
  BUDDY_UNPUSHED=0
  BUDDY_HEADS=0
  BUDDY_BEHIND=0
  BUDDY_TS=0
  local file="$BUDDY_CACHE/$1.state" key value
  [ -r "$file" ] || return 1
  while IFS='=' read -r key value; do
    case "$key" in
    dirty) BUDDY_DIRTY="$(buddy_num "$value")" ;;
    unpushed) BUDDY_UNPUSHED="$(buddy_num "$value")" ;;
    heads) BUDDY_HEADS="$(buddy_num "$value")" ;;
    behind) BUDDY_BEHIND="$(buddy_num "$value")" ;;
    ts) BUDDY_TS="$(buddy_num "$value")" ;;
    esac
  done <"$file"
}

# Test verdicts live in their own file so the probe and the Bash hook never race.
buddy_read_tests() {
  BUDDY_TESTS=unknown
  local file="$BUDDY_CACHE/$1.tests" key value now
  [ -r "$file" ] || return 0
  while IFS='=' read -r key value; do
    case "$key" in
    result) BUDDY_TESTS="$value" ;;
    ts) now="$value" ;;
    esac
  done <"$file"
  # A verdict older than two hours says nothing about the code as it stands now.
  if [ -n "${now:-}" ] && [ "$((${BUDDY_NOW:-0} - now))" -gt 7200 ]; then
    BUDDY_TESTS=unknown
  fi
}

buddy_score() {
  BUDDY_SCORE=5
  [ "$BUDDY_DIRTY" -gt 0 ] && BUDDY_SCORE=$((BUDDY_SCORE - 1))
  [ "$BUDDY_DIRTY" -gt 15 ] && BUDDY_SCORE=$((BUDDY_SCORE - 1))
  [ "$BUDDY_UNPUSHED" -gt 0 ] && BUDDY_SCORE=$((BUDDY_SCORE - 1))
  [ "$BUDDY_UNPUSHED" -gt 5 ] && BUDDY_SCORE=$((BUDDY_SCORE - 1))
  [ "$BUDDY_HEADS" -gt 1 ] && BUDDY_SCORE=$((BUDDY_SCORE - 2))
  [ "$BUDDY_TESTS" = fail ] && BUDDY_SCORE=$((BUDDY_SCORE - 2))
  [ "$BUDDY_SCORE" -lt 0 ] && BUDDY_SCORE=0
  return 0
}

buddy_face() {
  case "$BUDDY_SCORE" in
  5) BUDDY_EYES='•ᴗ•' ;;
  4) BUDDY_EYES='•_•' ;;
  3) BUDDY_EYES='¬_¬' ;;
  2) BUDDY_EYES='>_<' ;;
  1) BUDDY_EYES='@_@' ;;
  *) BUDDY_EYES='x_x' ;;
  esac
}
