#!/bin/bash
# Stop hook. Probes fresh, then says one short thing about the worst signal it can see.
set -u
BUDDY_HOME="${BUDDY_HOME:-${BASH_SOURCE[0]%/*}}"
. "$BUDDY_HOME/lib.sh"

payload="$(cat 2>/dev/null)"
cwd="$(printf '%s' "$payload" | jq -r '.cwd // ""' 2>/dev/null)"
[ -n "$cwd" ] && [ -d "$cwd" ] || cwd="$PWD"
buddy_locate "$cwd" || exit 0

buddy_species "$BUDDY_BRANCH"
key="$(buddy_key "$BUDDY_ROOT")"
"$BUDDY_HOME/probe.sh" "$BUDDY_ROOT" >/dev/null 2>&1
BUDDY_NOW="$(date +%s)"
buddy_read_state "$key" || exit 0
buddy_read_tests "$key"
buddy_score
buddy_face

pick() {
  eval "printf '%s' \"\${$((RANDOM % $# + 1))}\""
}

if [ "$BUDDY_HEADS" -gt 1 ]; then
  line="$(pick "$BUDDY_HEADS migration heads. that one never fixes itself." \
    "$BUDDY_HEADS heads in versions/. something is going to refuse to migrate.")"
elif [ "$BUDDY_TESTS" = fail ]; then
  line="$(pick "tests are red." "last run failed. i saw it.")"
elif [ "$BUDDY_UNPUSHED" -gt 5 ]; then
  line="$(pick "$BUDDY_UNPUSHED commits stacked up unpushed." \
    "$BUDDY_UNPUSHED unpushed. this branch only exists on your laptop.")"
elif [ "$BUDDY_DIRTY" -gt 15 ]; then
  line="$(pick "$BUDDY_DIRTY files uncommitted. that is a lot of uncommitted." \
    "$BUDDY_DIRTY dirty files. commit something.")"
elif [ "$BUDDY_UNPUSHED" -gt 0 ] || [ "$BUDDY_DIRTY" -gt 0 ]; then
  line="$(pick "${BUDDY_DIRTY}△ ${BUDDY_UNPUSHED}↑, nothing alarming." \
    "in progress, looks fine." "${BUDDY_DIRTY} dirty, ${BUDDY_UNPUSHED} unpushed.")"
else
  line="$(pick "clean and pushed." "nothing to report. rare." "all quiet.")"
fi

jq -cn --arg msg "$BUDDY_PRE$BUDDY_EYES$BUDDY_SUF $BUDDY_NAME: $line" '{systemMessage: $msg}'
