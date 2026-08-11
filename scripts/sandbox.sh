#!/bin/bash
# Run pets in a throwaway environment that cannot touch your real setup.
#
#   ./scripts/sandbox.sh          build from source, seed worktrees, open a shell
#   ./scripts/sandbox.sh --demo   same, but print the views and exit
#   PETS_BIN=$(which pets) ./scripts/sandbox.sh    test an installed build instead
#
# Everything lives under one directory: HOME, config, state, and the repos. Your
# own config, den and agent settings are never read or written.

set -euo pipefail

here="$(cd "$(dirname "$0")/.." && pwd)"
sandbox="${PETS_SANDBOX:-${TMPDIR:-/tmp}/pets-sandbox}"
demo=false
[ "${1:-}" = "--demo" ] && demo=true

rm -rf "$sandbox"
mkdir -p "$sandbox/home" "$sandbox/repos"

if [ -n "${PETS_BIN:-}" ]; then
  bin="$PETS_BIN"
else
  echo "building from source"
  go build -o "$sandbox/pets" "$here/cmd/pets"
  bin="$sandbox/pets"
fi

export HOME="$sandbox/home"
export XDG_CONFIG_HOME="$sandbox/home/.config"
export XDG_STATE_HOME="$sandbox/home/.local/state"
export PATH="$(dirname "$bin"):$PATH"

git_quiet() { git -C "$1" -c user.email=pets@example.com -c user.name=Pets "${@:2}"; }

seed() {
  local name="$1" dirty="$2" commits="$3"
  local dir="$sandbox/repos/${name//\//-}"
  git clone -q "$sandbox/repos/origin" "$dir"
  git_quiet "$dir" checkout -qb "$name"
  echo "work" >"$dir/work.txt"
  git_quiet "$dir" add -A
  git_quiet "$dir" commit -qm "start $name"
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
  printf '{"cwd":"%s"}' "$dir" | "$bin" hatch >/dev/null 2>&1 || true
}

echo "seeding worktrees"
git init -q "$sandbox/repos/origin"
git_quiet "$sandbox/repos/origin" commit -q --allow-empty -m init
git_quiet "$sandbox/repos/origin" branch -M main

# A spread of health, so the party view has something to sort.
seed feat/checkout-flow 0 1
seed fix/session-leak 2 1
seed chore/bump-deps 0 3
seed spike/wasm-build 22 9
seed docs/api-reference 1 1
seed refactor/auth-guard 41 13

# One failing test run, so a red mark shows up somewhere.
printf '{"cwd":"%s","tool_input":{"command":"go test ./..."},"tool_response":{"stdout":"--- FAIL: TestThing\\nFAIL\\n"}}' \
  "$sandbox/repos/spike-wasm-build" | "$bin" record >/dev/null 2>&1 || true

echo
echo "sandbox: $sandbox"
echo "binary:  $bin"
echo

if [ "$demo" = true ]; then
  "$bin" party
  "$bin" den
  exit 0
fi

cat <<EOF
Nothing here touches your real config. Try:

  pets party
  pets den
  cd $sandbox/repos/spike-wasm-build && pets card

Exit the shell to leave the sandbox. Delete it with:
  rm -rf $sandbox

EOF

exec "${SHELL:-/bin/sh}"
