#!/bin/bash
# Find out whether Codex actually fires the hooks pets installs.
#
#   ./scripts/codex-hook-probe.sh install   point ~/.codex/hooks.json at a logging wrapper
#   ./scripts/codex-hook-probe.sh report    show which hooks fired, and the payload keys
#   ./scripts/codex-hook-probe.sh restore   put your real hooks.json back
#
# Why a wrapper rather than reading pets' cache: a hook that fires and then bails for an
# unrelated reason leaves no trace, so the cache cannot distinguish "never fired" from
# "fired and did nothing". The wrapper records the invocation before pets sees it.
#
# It also captures the payloads, which is what issue #14 wants. Sanitise before committing
# any of them: a real payload carries your username in absolute paths, your repo owner, the
# session name and cost figures. See CONTRIBUTING.

set -euo pipefail

hooks="$HOME/.codex/hooks.json"
saved="$HOME/.codex/hooks.json.before-probe"
dir="${PETS_PROBE_DIR:-$HOME/.codex/pets-probe}"
wrapper="$dir/tee-hook"

case "${1:-}" in
install)
	[ -f "$hooks" ] || { echo "no $hooks - run 'pets install --harness=codex' first"; exit 1; }
	[ -f "$saved" ] && { echo "already installed; run restore first"; exit 1; }
	cp "$hooks" "$saved"
	mkdir -p "$dir"
	cat >"$wrapper" <<'INNER'
#!/bin/sh
# Records that this hook ran, and what it was handed, then behaves exactly as before.
event="$1"
shift
dir="$(dirname "$0")"
stamp="$(date +%Y%m%d-%H%M%S)"
payload="$dir/$event-$stamp.json"
tee "$payload" | "$@"
status=$?
printf '%s %s exit=%s payload=%s\n' "$stamp" "$event" "$status" "$payload" >>"$dir/fired.log"
exit $status
INNER
	chmod +x "$wrapper"

	pets_bin="$(command -v pets || echo /opt/homebrew/bin/pets)"
	python3 - "$hooks" "$wrapper" "$pets_bin" <<'PY'
import json, sys
path, wrapper, pets = sys.argv[1], sys.argv[2], sys.argv[3]
data = json.load(open(path))
for event, group in data.get("hooks", {}).items():
    for entry in group:
        for hook in entry.get("hooks", []):
            command = hook.get("command", "")
            if "pets" not in command:
                continue
            verb = command.split()[-1]
            hook["command"] = f'{wrapper} {event} {pets} {verb}'
json.dump(data, open(path, "w"), indent=2)
print("rewrote", path)
PY
	echo
	echo "Now, in a NEW terminal:"
	echo "  1. /opt/homebrew/bin/codex        # the real binary; a cmux shim shadows 'codex' on PATH"
	echo "  2. cd into a git worktree first, or the hooks bail before doing anything"
	echo "  3. ask it to run one shell command, so PostToolUse has a chance"
	echo "  4. end the turn, so Stop has a chance"
	echo "  5. come back and run: $0 report"
	;;
report)
	[ -f "$dir/fired.log" ] || { echo "nothing fired. Either Codex never ran, or it never called the hooks."; exit 0; }
	echo "=== hooks that fired ==="
	cat "$dir/fired.log"
	echo
	echo "=== payload keys, per capture ==="
	for f in "$dir"/*.json; do
		[ -f "$f" ] || continue
		printf '%s: ' "$(basename "$f")"
		python3 -c 'import json,sys; print(", ".join(sorted(json.load(open(sys.argv[1])).keys())))' "$f" 2>/dev/null || echo "(not JSON - that is itself the answer)"
	done
	echo
	echo "The question issue #15 asks: is there a per-session identifier in there?"
	;;
restore)
	[ -f "$saved" ] || { echo "no $saved to restore"; exit 1; }
	mv "$saved" "$hooks"
	echo "restored $hooks (captures left in $dir)"
	;;
*)
	sed -n '2,12p' "$0"
	exit 1
	;;
esac
