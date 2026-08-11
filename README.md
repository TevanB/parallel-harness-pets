# parallel-harness-pets

A creature for every worktree. It lives in your agent's status line, its species
is a pure function of the branch name, and its mood tracks branch hygiene.

```
((•ᴗ•))  owl  · feat/checkout-flow  ♥♥♥♥♥             ·  Opus 5
[@_@]    frog · fix/session-leak  ♥♡♡♡♡  28△ 27↑      ·  Opus 5
```

Most terminal pets belong to *you*: one creature, nurtured over time, with
affection to decay and XP to grind. This one belongs to a **worktree**. Six
worktrees means six creatures alive at once, which is the whole point.

## The party

One screen, every live worktree, and the worst thing wrong across all of them.

```
  pets party                                  6 alive

  /<•_•>\   cat  ✦     spike/wasm-build      ♥♥♡♡♡  22△ 1↑
  {•_•}     fox  ✦     chore/bump-deps       ♥♥♥♥♡  1↑
  o[•_•]o   seal ✦✦✦   docs/install-rewrite  ♥♥♥♥♡  1↑
  <•_•>     moth ✦     feat/checkout-flow    ♥♥♥♥♡  1↑
  o[•_•]o   seal ✦✦✦   fix/session-leak      ♥♥♥♥♡  1↑
  <@_@>     moth ✦     refactor/auth-guard   ♥♡♡♡♡  41△ 13↑

  worst: moth · uncommitted, unpushed
```

Two seals and two moths, at different colours, still read apart.

## The collection

Opening a branch you have never opened before hatches its creature. Species are
banded by rarity, and the bands hold no matter how many creatures exist:

| Band | Odds | |
|---|---|---|
| common | 60% | `✦` |
| uncommon | 25% | `✦✦` |
| rare | 11% | `✦✦✦` |
| legendary | 3.5% | `✦✦✦✦` |
| mythic | 0.5% | `✦✦✦✦✦` |

Shiny is an independent roll at 1 in 128, so a shiny common is its own find.

`pets den` shows what you have. The only way to roll is to open a new worktree,
and a creature only enters the den once that worktree has a commit, so the
collection stays a record of branches where work actually happened.

## Why

Two problems, one status line.

**Telling sessions apart.** Running several agent sessions across git worktrees
means several near-identical fullscreen terminals. Branch names are long and
their prefixes collide (`feat/checkout-...`, `feat/checkout-v2-...`). A creature
silhouette in a distinct colour is recognisable pre-consciously, before you read
any text.

**Seeing state you would otherwise miss.** Uncommitted sprawl, unpushed commits
and split migration heads are all invisible until they bite. The pet is a gauge
for them that happens to have a face, which is what keeps it installed: a purely
cosmetic status line gets deleted the first time it is mildly annoying.

## Install

```sh
brew install TevanB/tap/parallel-harness-pets
pets install
```

Or grab an archive from
[Releases](https://github.com/TevanB/parallel-harness-pets/releases) and put
`pets` on your `PATH`, or build it with
`go install github.com/TevanB/parallel-harness-pets/cmd/pets@latest`.

One static binary. No bash, no jq, no Python, no Node, nothing to boot. `pets
install` detects which agents you actually have and wires itself into them, then
prints the snippets for tmux and your shell. It backs up any config it touches
and leaves your other settings and hooks alone. `pets uninstall` reverses it.

If an agent is installed but has never been run, its config directory does not
exist yet and there is nothing to detect. Name it directly in that case:

```sh
pets install --harness=claude
```

Start a new session afterwards, since agents read status line and hook config at
startup.

## Surfaces

| Surface | How | Notes |
|---|---|---|
| Claude Code | `statusLine` + `PostToolUse`/`Stop` hooks | The richest surface: per-second refresh *and* hooks |
| tmux / shell prompt | `pets render --format=tmux` in `status-right` | Works with **any** agent. No harness support needed |
| Codex CLI | hooks + terminal title | Same hook event schema as Claude Code. Its status line is a fixed item list, so the pet lives in the title |
| Editors | `pets render --format=json` | The contract for anything not yet invented |

The tmux and shell surfaces are the interesting ones: they need nothing from the
agent, so they work with Aider, Amp, Gemini CLI, Crush, or whatever ships next.

## Reading it

| Glyph | Meaning |
|---|---|
| `•ᴗ•` `•_•` `¬_¬` `>_<` `@_@` `x_x` | mood, five hearts down to zero |
| `7△` | 7 uncommitted files |
| `3↑` | 3 unpushed commits |
| `105↓` | commits behind trunk, shown past 40, no penalty |
| `⑂2` | 2 heads in one migration tree |
| `✗` | last test or lint run failed |
| `-.-` with `·····` | first second of a session, before the probe lands |

Scoring starts at five hearts: minus one for any uncommitted file and another
past 15, minus one for any unpushed commit and another past 5, minus two for a
failing test run. All of it is configurable.

## Identity

Species and colour are independent hashes of the branch name. The same branch
always summons the same creature, on any machine, with nothing persisted. There
is no rerolling *within* a branch, which is the point: the silhouette **is** the
branch.

Where two live branches land on the same species the colours still differ, so
they read apart.

## Configuration

Everything lives in `~/.config/pets/config.toml`, and nothing is required.

```toml
[branch]
# Detected from origin/HEAD; set only to override.
# default = "origin/trunk"

[score]
dirty_many_at = 15
unpushed_many_at = 5
tests_failing = 2

[signals]
enabled = ["dirty", "unpushed", "behind", "tests"]

# Off by default: it is a large penalty for a signal most repos cannot produce.
[signals.migrations]
enabled = false
penalty = 2
paths = ["alembic/versions", "migrations/versions"]
```

### Teaching it to read your stack

Any executable that prints `key=value` lines on stdout is a signal. Drop it in
`~/.config/pets/signals/` and it is scored alongside the builtins. It receives
the worktree path as its first argument.

```sh
#!/bin/sh
# ~/.config/pets/signals/cargo.sh
echo "clippy=$(cargo clippy --message-format=short 2>&1 | grep -c '^warning')"
```

## How it works

Three seams, and a cache between them.

| Piece | Runs | Job |
|---|---|---|
| `pets render` | status line, ~1s | Render from cache only. Never runs git |
| `pets probe` | detached, 15s throttle | The git work. Writes the cache |
| `pets record` | `PostToolUse` hook | Record a test verdict when output states one |
| `pets quip` | `Stop` hook | Say one line about the worst signal |
| `pets card` | on demand | Full readout with penalties called out |
| `pets hatch` | `SessionStart` hook | Greet a worktree opened for the first time |
| `pets party` | on demand | Every live worktree at once, worst first (`--all` to skip the cap) |
| `pets den` | on demand | The collection |

The split matters. A one second refresh across several sessions cannot afford
`git status`, so the renderer only reads two small files and the expensive work
happens out of band.

Test verdicts live in their own file so the probe and the tool hook never race
on one another's writes.

## Notes and gotchas

**Test verdicts are never inferred.** A hook is not handed the exit code, so
`pets record` only believes a runner that states its result outright. Anything
else leaves the previous verdict alone, and a verdict older than two hours
decays to `unknown` so a stale red cannot haunt the branch.

**Read the verdict from the output, never the command.** An early version
scanned the command string too, so any command that merely mentioned `3 failed`
in a heredoc, a grep pattern, or a commit message recorded a failure that never
happened. The command now only gates *whether this was a test run*.

**Counted failures need a non-zero count.** `cargo test` prints `test result:
ok. 14 passed; 0 failed` on success, so matching the bare word ` failed` reads a
clean run as red.

**Migration heads are per tree, not summed.** A repo with two migration trees
each having one head is healthy. Summing gives 2 and cries wolf, so the worst
single tree wins.

## Contributing

Two ways in, and only one of them needs Go.

**Creatures** are data. **Signals** are any executable that prints `key=value`.
Neither requires touching the core.

## Licence

MIT. See [LICENSE](LICENSE).

## Coffee

If the pet earned its keep, the developer takes payment in coffee.

[![Buy me a coffee](https://img.shields.io/badge/buy_me_a_coffee-ffdd00?style=flat&logo=buymeacoffee&logoColor=black)](https://buymeacoffee.com/tevanb)
