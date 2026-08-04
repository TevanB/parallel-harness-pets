#!/bin/bash
# Full readout for /buddy: the creature plus every signal behind its mood, including the ones costing it hearts.
set -u
BUDDY_HOME="${BUDDY_HOME:-${BASH_SOURCE[0]%/*}}"
. "$BUDDY_HOME/lib.sh"

cwd="${1:-$PWD}"
esc=$'\033'
dim="$esc[38;5;240m"
reset="$esc[0m"
warn="$esc[38;5;179m"

if ! buddy_locate "$cwd"; then
  printf '%s(-.-) nothing here but %s%s\n' "$dim" "$cwd" "$reset"
  exit 0
fi

buddy_species "$BUDDY_BRANCH"
key="$(buddy_key "$BUDDY_ROOT")"
"$BUDDY_HOME/probe.sh" "$BUDDY_ROOT" >/dev/null 2>&1
BUDDY_NOW="$(date +%s)"
if ! buddy_read_state "$key"; then
  printf '%s(-.-) %s could not read its state. Is %s writable?%s\n' \
    "$dim" "$BUDDY_NAME" "$BUDDY_CACHE" "$reset"
  exit 1
fi
buddy_read_tests "$key"
buddy_score
buddy_face

filled=""
empty=""
i=0
while [ "$i" -lt "$BUDDY_SCORE" ]; do
  filled="${filled}♥"
  i=$((i + 1))
done
while [ "$i" -lt 5 ]; do
  empty="${empty}♡"
  i=$((i + 1))
done

hue="$esc[38;5;${BUDDY_COLOR}m"
row() { printf '  %s%-14s%s %s\n' "$dim" "$1" "$reset" "$2"; }

printf '\n  %s%s%s%s%s  %s%s%s\n' "$hue" "$BUDDY_PRE" "$BUDDY_EYES" "$BUDDY_SUF" "$reset" "$hue" "$BUDDY_NAME" "$reset"
printf '  %s%s%s\n' "$dim" '──────────────────────────────' "$reset"
row branch "$BUDDY_BRANCH"
row worktree "$BUDDY_ROOT"
row mood "$esc[38;5;210m${filled}${dim}${empty}${reset}  $BUDDY_SCORE/5"
printf '\n'

note() { # value, label, penalised
  if [ "$3" = yes ]; then
    printf '  %s%-14s%s %s%s%s\n' "$dim" "$2" "$reset" "$warn" "$1" "$reset"
  else
    row "$2" "$1"
  fi
}
[ "$BUDDY_DIRTY" -gt 0 ] && p=yes || p=no
note "$BUDDY_DIRTY" uncommitted "$p"
[ "$BUDDY_UNPUSHED" -gt 0 ] && p=yes || p=no
note "$BUDDY_UNPUSHED" unpushed "$p"
row "behind main" "$BUDDY_BEHIND"
[ "$BUDDY_HEADS" -gt 1 ] && p=yes || p=no
note "$BUDDY_HEADS" "alembic heads" "$p"
[ "$BUDDY_TESTS" = fail ] && p=yes || p=no
note "$BUDDY_TESTS" "last tests" "$p"
printf '\n'
