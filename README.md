# parallel-harness-pets

A creature for every git worktree, living in your coding agent's status line.
Its species comes from the branch name, and its mood tracks how tidy that branch is.

```
<•_•>    moth · feat/checkout-flow  ♥♥♥♥♡  2△        ·  Opus 5
o[@_@]o  seal · fix/session-leak    ♥♡♡♡♡  28△ 27↑  ·  Opus 5
```

Most terminal pets belong to *you*: one creature, nurtured over time. This one
belongs to a **worktree**. Six worktrees means six creatures alive at once, each
recognisable at a glance, so you always know which session you are looking at
and which one is in trouble.

## Install

**Linux and macOS**

```sh
curl -fsSL https://raw.githubusercontent.com/TevvvB/parallel-harness-pets/main/install.sh | sh
pets install
```

**macOS with Homebrew**

```sh
brew install TevvvB/tap/parallel-harness-pets
pets install
```

**Windows** - grab an archive from
[Releases](https://github.com/TevvvB/parallel-harness-pets/releases), put `pets`
on your `PATH`, then run `pets install`.

**With Go** - `go install github.com/TevvvB/parallel-harness-pets/cmd/pets@latest`

`pets install` finds the agents you have and wires itself in, backing up any
config it touches and leaving your other settings alone. Start a new session
afterwards. `pets uninstall` reverses it.

If an agent is installed but has never been run, it has no config directory yet,
so name it directly: `pets install --harness=claude`.

## Updating

| Installed with | Update with |
|---|---|
| Homebrew | `brew upgrade parallel-harness-pets` |
| The install script | re-run it, it replaces the binary in place |
| Go | `go install github.com/TevvvB/parallel-harness-pets/cmd/pets@latest` |
| A release archive | download the new one |

`pets card`, `pets party` and `pets den` check once a day whether a newer
release exists and print a line if so. Nothing else ever does: no hook and
nothing on the status line path touches the network. Turn it off with
`check = false` under `[update]`.

## Supported agents

| Agent | What you get |
|---|---|
| Claude Code | Full: live status line, hatch, quips |
| Codex CLI | Hooks are written, but untested against a real Codex. Use the shell snippet below for the pet itself |
| tmux or your shell prompt | Full pet, works with **any** agent |
| Editors | `pets render --format=json` to build your own |

The tmux and shell options need nothing from the agent, so they work with Aider,
Amp, Gemini CLI, or anything else. `pets install` prints the snippets.

Claude Code is the only agent this has been verified against end to end. If you
run something else and it works, or does not, please open an issue.

## Reading it

| Glyph | Meaning |
|---|---|
| `•ᴗ•` `•_•` `¬_¬` `>_<` `@_@` `x_x` | mood, five hearts down to zero |
| `7△` | 7 uncommitted files |
| `3↑` | 3 unpushed commits |
| `105↓` | commits behind trunk, shown past 40, costs nothing |
| `⑂2` | 2 heads in one migration tree |
| `✗` | last test or lint run failed |
| `-.-` with `·····` | still waking up, in the first second of a session |

You start at five hearts and lose one for any uncommitted file, another past 15,
one for any unpushed commit, another past 5, and two for a failing test run. All
of it is configurable.

Test results are only recorded when a runner says outright how it went, and a
result older than two hours is forgotten, so a red mark never haunts a branch
you already fixed.

## Seeing every worktree at once

```
  pets party                                  6 alive

  /x_x\     cat  ✦     spike/wasm-build     ♡♡♡♡♡  22△ 9↑ ✗
  <@_@>     moth ✦     refactor/auth-guard  ♥♡♡♡♡  41△ 13↑
  \(¬_¬)/   crow ✦✦✦   docs/api-reference   ♥♥♥♡♡  1△ 1↑
  o[¬_¬]o   seal ✦✦✦   fix/session-leak     ♥♥♥♡♡  2△ 1↑
  {•_•}     fox  ✦     chore/bump-deps      ♥♥♥♥♡  3↑
  <•_•>     moth ✦     feat/checkout-flow   ♥♥♥♥♡  1↑

  worst: cat · uncommitted, unpushed, tests
```

Worst first, so whatever needs attention is at the top. Long lists are trimmed;
`pets party --all` shows everything.

## Collecting them

Open a branch you have never opened before and its creature hatches. Species are
banded by rarity:

| Band | Odds | |
|---|---|---|
| common | 60% | `✦` |
| uncommon | 25% | `✦✦` |
| rare | 11% | `✦✦✦` |
| legendary | 3.5% | `✦✦✦✦` |
| mythic | 0.5% | `✦✦✦✦✦` |

Shiny is a separate 1-in-128 roll, so even a common can be a find. `pets den`
shows your collection.

The same branch name always summons the same creature, on any machine, so a
silhouette is a reliable way to recognise a session. The only way to roll a new
one is to open a new worktree, and a creature joins your den once that worktree
has a commit.

## Commands

| | |
|---|---|
| `pets party [--all]` | every live worktree, worst first |
| `pets den` | your collection |
| `pets card [path]` | full readout for one worktree |
| `pets render [--format=…]` | `statusline`, `tmux`, `title` or `json` |
| `pets install` / `pets uninstall` | wire into your agents |
| `pets version` | |

## Configuration

Optional, in `~/.config/pets/config.toml`.

```toml
[branch]
# Detected from your remote; set only to override.
# default = "origin/trunk"

[update]
# Once a day, and only from card, party and den.
check = true

[score]
dirty_many_at = 15
unpushed_many_at = 5
tests_failing = 2

[display]
party_limit = 12

[signals]
enabled = ["dirty", "unpushed", "behind", "tests"]

# Off by default. Turn on if your project uses Alembic or Django migrations,
# and the pet will warn you when a tree has split into two heads.
[signals.migrations]
enabled = false
penalty = 2
```

### Teaching it your stack

Any executable that prints `key=value` lines is a signal. Drop it in
`~/.config/pets/signals/` and it is scored with the built-in ones. It gets the
worktree path as its first argument.

```sh
#!/bin/sh
# ~/.config/pets/signals/cargo.sh
echo "clippy=$(cargo clippy --message-format=short 2>&1 | grep -c '^warning')"
```

## Contributing

Two ways in, and only one needs Go. **Creatures** are data files, so adding one
means writing six lines and no code. **Signals** are any executable that prints
`key=value`. Issues and pull requests welcome.

## Licence

MIT. See [LICENSE](LICENSE).
