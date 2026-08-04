#!/bin/bash
# Recompute one worktree's hygiene and write its cache. Cheap enough to run inline, backgrounded by the statusline.
set -u
BUDDY_HOME="${BUDDY_HOME:-${BASH_SOURCE[0]%/*}}"
. "$BUDDY_HOME/lib.sh"

root="${1:-}"
[ -n "$root" ] && [ -d "$root" ] || exit 0
key="$(buddy_key "$root")"

mkdir -p "$BUDDY_CACHE"
lock="$BUDDY_CACHE/$key.lock"
mkdir "$lock" 2>/dev/null || exit 0
trap 'rmdir "$lock" 2>/dev/null' EXIT

dirty="$(git -C "$root" status --porcelain 2>/dev/null | grep -c '^.')"

unpushed="$(git -C "$root" rev-list --count '@{upstream}..HEAD' 2>/dev/null)"
[ -n "$unpushed" ] || unpushed="$(git -C "$root" rev-list --count 'origin/main..HEAD' 2>/dev/null)"
[ -n "$unpushed" ] || unpushed=0

behind="$(git -C "$root" rev-list --count 'HEAD..origin/main' 2>/dev/null)"
[ -n "$behind" ] || behind=0

# Heads are the revisions nothing else declares as a parent. Each tree should have exactly one, so take the worst tree.
heads=0
for versions in "$root"/*/alembic/versions "$root"/*/*/migrations/versions; do
  [ -d "$versions" ] || continue
  revs="$(grep -hE '^revision' "$versions"/*.py 2>/dev/null | grep -oE "['\"][^'\"]+['\"]" | tr -d "\"'" | sort -u)"
  [ -n "$revs" ] || continue
  downs="$(grep -hE '^down_revision' "$versions"/*.py 2>/dev/null | grep -oE "['\"][^'\"]+['\"]" | tr -d "\"'" | sort -u)"
  found="$(comm -23 <(printf '%s\n' "$revs") <(printf '%s\n' "$downs") | grep -c '^.')"
  [ "$found" -gt "$heads" ] && heads="$found"
done

tmp="$BUDDY_CACHE/$key.state.$$"
{
  echo "dirty=$dirty"
  echo "unpushed=$unpushed"
  echo "behind=$behind"
  echo "heads=$heads"
  echo "ts=$(date +%s)"
} >"$tmp" && mv -f "$tmp" "$BUDDY_CACHE/$key.state"
