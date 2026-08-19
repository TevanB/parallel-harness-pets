#!/bin/bash
# Build the demo environment the README's recordings are made in.
#
#   ./scripts/demo.sh          seed a throwaway world, then open a shell in it
#   ./scripts/demo.sh --print  seed it, print party/den/card, exit
#   ./scripts/demo.sh --seed   seed it and exit, for docs/demo.tape to record against
#
# This lives in the repo on purpose. Every recording has to be remade whenever the
# views change, and the previous one-off script was deleted, which is how docs/demo.gif
# came to be two releases and one data-model rewrite out of date.
#
# It also exists because the only other impressive screenshot is a real machine's, and
# a real machine's worktrees are somebody's actual work. Nothing here is anybody's.
#
# HOME, config, state and the repos all live under one throwaway directory, so your own
# collection is never read or written.

set -euo pipefail

here="$(cd "$(dirname "$0")/.." && pwd)"
sandbox="${PETS_SANDBOX:-${TMPDIR:-/tmp}/pets-demo}"
print=false
seed_only=false
[ "${1:-}" = "--print" ] && print=true
[ "${1:-}" = "--seed" ] && seed_only=true

rm -rf "$sandbox"
mkdir -p "$sandbox/home" "$sandbox/repos"

if [ -n "${PETS_BIN:-}" ]; then
	bin="$PETS_BIN"
else
	go build -o "$sandbox/pets" "$here/cmd/pets"
	bin="$sandbox/pets"
fi

export HOME="$sandbox/home"
export XDG_CONFIG_HOME="$sandbox/home/.config"
export XDG_STATE_HOME="$sandbox/home/.local/state"
export PATH="$(dirname "$bin"):$PATH"

git_quiet() { git -C "$1" -c user.email=demo@example.com -c user.name=Demo "${@:2}"; }

# An agent, not just a worktree. The creature belongs to the session, so a den with
# nobody in it can only ever show the branch's fallback pet - which is not the thing
# worth recording.
join_den() {
	local dir="$1" session="$2" name="$3"
	printf '{"session_id":"%s","session_name":"%s","cwd":"%s","tool_input":{"command":"ls"},"tool_response":{"stdout":""}}' \
		"$session" "$name" "$dir" | "$bin" record >/dev/null 2>&1 || true
}

seed() {
	local branch="$1" dirty="$2" commits="$3"
	local dir="$sandbox/repos/${branch//\//-}"
	git clone -q "$sandbox/repos/origin" "$dir"
	git_quiet "$dir" checkout -qb "$branch"
	echo work >"$dir/work.txt"
	git_quiet "$dir" add -A
	git_quiet "$dir" commit -qm "start $branch"
	local index=1
	while [ "$index" -lt "$commits" ]; do
		echo "$index" >>"$dir/work.txt"
		git_quiet "$dir" commit -qam "change $index"
		index=$((index + 1))
	done
	index=0
	while [ "$index" -lt "$dirty" ]; do
		echo scratch >"$dir/scratch$index.txt"
		index=$((index + 1))
	done
	"$bin" probe "$dir" >/dev/null 2>&1 || true
	echo "$dir"
}

git init -q "$sandbox/repos/origin"
git_quiet "$sandbox/repos/origin" commit -q --allow-empty -m init
git_quiet "$sandbox/repos/origin" branch -M main

# A spread of health, so the party view has something to sort, and enough branches that
# the collection shows progress across bands and cities. The branch names matter as much
# as the sessions: recordIfEarned keys the collection on the branch, so feat/token is
# what puts a legendary in the bands and spike/upload is what puts a shiny there.
one=$(seed feat/checkout-flow 0 1)
two=$(seed fix/session-leak 2 1)
three=$(seed chore/bump-deps 0 3)
four=$(seed spike/wasm-build 22 9)
five=$(seed spike/upload 1 1)
six=$(seed refactor/auth-guard 41 13)
seven=$(seed feat/token 3 2)
eight=$(seed perf/index-scan 0 5)

# Two of these session ids were picked because their hash rolls something worth
# showing: ...073 is a shiny, ...2500 is a legendary. A mythic was deliberately left
# out - at 0.5% one across eight dens would imply they are common.
join_den "$one" 7f3a91c2-0000-4000-8000-000000000073 "Wire up the checkout flow"
join_den "$two" 7f3a91c2-0000-4000-8000-000000000002 "Track down the session leak"
join_den "$three" 7f3a91c2-0000-4000-8000-000000000003 "Bump dependencies"
join_den "$four" 7f3a91c2-0000-4000-8000-000000000004 "Get the wasm build green"
join_den "$seven" 7f3a91c2-0000-4000-8000-000000000005 "Rotate the signing tokens"
join_den "$eight" 7f3a91c2-0000-4000-8000-000000002500 "Speed up the index scan"

# Two agents in one den, which is the thing no worktree-shaped tool can show and the
# single most important frame in any recording.
join_den "$four" 7f3a91c2-0000-4000-8000-000000000007 "Second pass on the wasm build"

# One failing run, so a red mark appears somewhere.
printf '{"cwd":"%s","tool_input":{"command":"go test ./..."},"tool_response":{"stdout":"--- FAIL: TestThing\\nFAIL\\n"}}' \
	"$four" | "$bin" record >/dev/null 2>&1 || true

if [ "$seed_only" = true ]; then
	echo "$sandbox"
	exit 0
fi

if [ "$print" = true ]; then
	"$bin" party
	"$bin" den
	(cd "$four" && "$bin" card)
	exit 0
fi

cat <<EOF

demo world: $sandbox

  pets party
  pets den
  cd $four && pets card      # the den with two agents in it

Exit the shell to leave it. Remove it with: rm -rf $sandbox

EOF

exec "${SHELL:-/bin/sh}"
