#!/bin/bash
# Claude Code statusline. Renders this worktree's familiar from cached state only, then refreshes that cache out of band.
set -u
BUDDY_HOME="${BUDDY_HOME:-${BASH_SOURCE[0]%/*}}"
. "$BUDDY_HOME/lib.sh"

cwd=""
model=""
if [ ! -t 0 ]; then
  IFS=$'\t' read -r cwd model < <(jq -r '[(.workspace.current_dir // .cwd // ""), (.model.display_name // "")] | @tsv' 2>/dev/null)
fi
[ -n "$cwd" ] && [ -d "$cwd" ] || cwd="$PWD"

esc=$'\033'
dim="$esc[38;5;240m"
reset="$esc[0m"

if ! buddy_locate "$cwd"; then
  printf '%s%s %s%s\n' "$dim" '(-.-)' "${cwd##*/}" "$reset"
  exit 0
fi

buddy_species "$BUDDY_BRANCH"
key="$(buddy_key "$BUDDY_ROOT")"
BUDDY_NOW="$(date +%s)"

have_state=1
buddy_read_state "$key" || have_state=0
buddy_read_tests "$key"

if [ "$((BUDDY_NOW - BUDDY_TS))" -ge "$BUDDY_STALE_AFTER" ]; then
  "$BUDDY_HOME/probe.sh" "$BUDDY_ROOT" >/dev/null 2>&1 &
fi

label="$BUDDY_BRANCH"
[ "${#label}" -gt 26 ] && label="${label:0:25}…"

if [ "$have_state" -eq 0 ]; then
  gauge="${dim}·····${reset}"
  BUDDY_EYES='-.-'
  flags=""
else
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
  gauge="$esc[38;5;210m${filled}${dim}${empty}${reset}"

  flags=""
  [ "$BUDDY_DIRTY" -gt 0 ] && flags="$flags ${BUDDY_DIRTY}△"
  [ "$BUDDY_UNPUSHED" -gt 0 ] && flags="$flags ${BUDDY_UNPUSHED}↑"
  [ "$BUDDY_BEHIND" -gt 40 ] && flags="$flags ${BUDDY_BEHIND}↓"
  [ "$BUDDY_HEADS" -gt 1 ] && flags="$flags ⑂$BUDDY_HEADS"
  [ "$BUDDY_TESTS" = fail ] && flags="$flags ✗"
  [ -n "$flags" ] && flags="$esc[38;5;179m${flags# }$reset "
fi

hue="$esc[38;5;${BUDDY_COLOR}m"
sep="${dim}·${reset}"
printf '%s %s %s %s %s %s%s%s\n' \
  "${hue}${BUDDY_PRE}${BUDDY_EYES}${BUDDY_SUF}${reset}" \
  "${hue}${BUDDY_NAME}${reset}" \
  "$sep" \
  "$esc[38;5;252m${label}${reset}" \
  "$gauge" \
  "$flags" "$sep" " ${dim}${model}${reset}"
