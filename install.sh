#!/bin/bash
# Install into ~/.claude and point Claude Code's statusline and hooks at it. Safe to re-run.
set -eu

src="$(cd "$(dirname "$0")" && pwd)"
home_dir="$HOME/.claude/claude-buddy"
skill_dir="$HOME/.claude/skills/buddy"
settings="$HOME/.claude/settings.json"

mkdir -p "$home_dir/cache" "$skill_dir"
for f in lib.sh statusline.sh probe.sh quip.sh record.sh card.sh; do
  install -m 755 "$src/$f" "$home_dir/$f"
done
install -m 644 "$src/skill/SKILL.md" "$skill_dir/SKILL.md"
echo "installed scripts -> $home_dir"

if [ ! -f "$settings" ]; then
  echo '{}' >"$settings"
fi
cp "$settings" "$settings.bak-buddy-$(date +%Y%m%d%H%M%S)"

BUDDY_TARGET="$home_dir" python3 - "$settings" <<'PY'
import json, os, sys

path = sys.argv[1]
target = os.environ["BUDDY_TARGET"]
with open(path) as handle:
    settings = json.load(handle)

settings["statusLine"] = {
    "type": "command",
    "command": f"{target}/statusline.sh",
    "padding": 1,
    "refreshInterval": 1,
}

hooks = settings.setdefault("hooks", {})


def replace(event, command, matcher=None):
    """Drop any existing claude-buddy entry for this event, then add ours."""
    kept = []
    for entry in hooks.get(event, []):
        inner = [h for h in entry.get("hooks", []) if "claude-buddy" not in h.get("command", "")]
        if inner:
            entry = dict(entry, hooks=inner)
            kept.append(entry)
    ours = {"hooks": [{"type": "command", "command": command}]}
    if matcher:
        ours["matcher"] = matcher
    hooks[event] = kept + [ours]


replace("PostToolUse", f"{target}/record.sh", matcher="Bash")
replace("Stop", f"{target}/quip.sh")

with open(path, "w") as handle:
    json.dump(settings, handle, indent=2)
    handle.write("\n")
print("patched", path)
PY

echo
echo "Done. Start a new Claude Code session to see it."
echo "Check it now with: $home_dir/card.sh \"\$(pwd)\""
