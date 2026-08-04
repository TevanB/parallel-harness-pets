#!/bin/bash
# PostToolUse(Bash). Records a test or lint verdict when the output states one outright, and stays silent otherwise.
set -u
BUDDY_HOME="${BUDDY_HOME:-${BASH_SOURCE[0]%/*}}"
. "$BUDDY_HOME/lib.sh"

payload="$(cat 2>/dev/null)"
[ -n "$payload" ] || exit 0

cwd="$(printf '%s' "$payload" | jq -r '.cwd // ""' 2>/dev/null)"
[ -n "$cwd" ] && [ -d "$cwd" ] || exit 0
buddy_locate "$cwd" || exit 0

cmd="$(printf '%s' "$payload" | jq -r '.tool_input.command // ""' 2>/dev/null)"
out="$(printf '%s' "$payload" | jq -r '.tool_response | tostring' 2>/dev/null)"
[ -n "$out" ] || exit 0

# The command only decides whether this was a test run. Reading a verdict out of it would let any
# command that merely mentions "3 failed" in a string or heredoc record a failure that never happened.
case "$cmd" in
*pytest* | *vitest* | *jest* | *" test"* | *ruff* | *eslint* | *tsc* | *mypy* | *pyright*) ;;
*) exit 0 ;;
esac

# Only verdicts a runner spells out count, so the pet never infers a result it cannot see.
result=""
case "$out" in
*' failed'*) result=fail ;;
*'error TS'*) result=fail ;;
*'Found '*' error'*) result=fail ;;
*' problems ('*) result=fail ;;
esac
if [ -z "$result" ]; then
  case "$out" in
  *' passed'*) result=pass ;;
  *'All checks passed'*) result=pass ;;
  esac
fi
[ -n "$result" ] || exit 0

key="$(buddy_key "$BUDDY_ROOT")"
tmp="$BUDDY_CACHE/$key.tests.$$"
{
  echo "result=$result"
  echo "ts=$(date +%s)"
} >"$tmp" && mv -f "$tmp" "$BUDDY_CACHE/$key.tests"
exit 0
