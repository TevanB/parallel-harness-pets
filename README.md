# claude-buddy

A per-worktree familiar for Claude Code. It lives in the status line, its species
is a pure function of the branch name, and its mood tracks branch hygiene.

```
>(•ᴗ•)<  crab · agent-task-1785874237  ♥♥♥♥♥            ·  Opus 5
[@_@]    frog · DIL-3982-income-calculation  ♥♡♡♡♡  28△ 27↑  ·  Opus 5
```

## Why

Two problems, one status line.

**Telling sessions apart.** Running several Claude Code sessions across git
worktrees means several near-identical fullscreen terminals. The branch name is
long and the prefixes collide (`DIL-4118-...`, `DIL-4068-...`). A creature
silhouette in a distinct colour is recognisable pre-consciously, before you read
any text.

**Seeing state you would otherwise miss.** Uncommitted sprawl, unpushed
commits, and split alembic heads are all invisible until they bite. The pet is a
gauge for them that happens to have a face, which is what keeps it installed: a
purely cosmetic status line gets deleted the first time it is mildly annoying.

## Install

```sh
./install.sh
```

Copies the scripts to `~/.claude/claude-buddy`, the skill to
`~/.claude/skills/buddy`, and points `statusLine`, `PostToolUse`, and `Stop` at
them in `~/.claude/settings.json`. It backs up that file first and is safe to
re-run. Start a new session afterwards, since Claude Code reads the status line
and hook config at startup.

Requires `bash`, `git`, and `jq`. Deliberately no MCP server and no Node
dependency, so nothing has to boot for the pet to exist.

## Reading it

| Glyph | Meaning |
|---|---|
| `•ᴗ•` `•_•` `¬_¬` `>_<` `@_@` `x_x` | mood, five hearts down to zero |
| `7△` | 7 uncommitted files |
| `3↑` | 3 unpushed commits |
| `105↓` | commits behind `origin/main`, shown past 40, no penalty |
| `⑂2` | 2 alembic heads in one migration tree, costs 2 hearts |
| `✗` | last test or lint run failed |
| `-.-` with `·····` | first second of a session, before the probe lands |

Scoring starts at five hearts: minus one for any uncommitted file and another
past 15, minus one for any unpushed commit and another past 5, minus two for
split alembic heads, minus two for a failing test run.

`/buddy` prints the full card with the penalised signals in amber. `/buddy why`
adds the command that clears each one.

## Identity

Species and colour are independent hashes of the branch name, 16 creatures
across 12 colours. The same branch always summons the same creature, on any
machine, with nothing persisted. There is no naming or rerolling, which is the
point: the silhouette *is* the branch.

Where two live branches land on the same species the colours still differ, so
they read apart.

## How it works

Three seams in the Claude Code harness, and a cache between them.

| Piece | Seam | Job |
|---|---|---|
| `statusline.sh` | `statusLine`, 1s | Render from cache only. Never runs git. |
| `probe.sh` | backgrounded, 15s throttle | The git work. Writes `cache/<key>.state`. |
| `record.sh` | `PostToolUse(Bash)` | Record a test verdict when output states one. |
| `quip.sh` | `Stop` | Probe fresh, say one line about the worst signal. |
| `card.sh` | `/buddy` | Full readout with penalties called out. |

The split matters. A one second refresh across several sessions cannot afford
`git status`, so the renderer only reads two small files and the expensive work
happens out of band. Measured on an M-series Mac: 14ms per render, 127ms per
probe.

Test verdicts live in their own file so `probe.sh` and `record.sh` never race on
one another's writes.

## Notes and gotchas

**macOS bash is 3.2.** A `$var` immediately followed by a non-ASCII character
breaks, because bash folds the multibyte bytes into the variable name:
`"$dim·····"` fails with ``dim·: unbound variable``. Always brace it,
`"${dim}·····"`. This bites anything writing box-drawing characters or hearts
from a status line script.

**Alembic heads are per tree, not summed.** A repo with two migration trees
each having one head is healthy. Summing gives 2 and cries wolf on a clean
repo, so `probe.sh` takes the worst single tree.

**Test verdicts are never inferred.** `PostToolUse` does not hand a hook the
exit code, so `record.sh` only believes a runner that states its result outright
(`== 3 failed`, `Found 7 errors`, `All checks passed`). Anything else leaves the
previous verdict alone, and a verdict older than two hours decays to `unknown`
so a stale red cannot haunt the branch.

**Read the verdict from the output, never the command.** An early version
scanned the command string too, so any command that merely mentioned `3 failed`
in a heredoc, a grep pattern, or a commit message recorded a failure that never
happened. The command now only gates *whether this was a test run*; the verdict
comes from the response alone.

**A checkout and an install have separate caches**, because `BUDDY_HOME`
resolves relative to the script. Running `./card.sh` from the repo will not
update what an installed status line shows. Run `./install.sh` to deploy.

**Do not install it inside a working repo.** The predecessor to this lived in an
untracked directory inside a checkout and a `git clean -fd` removed it. The
status line then failed silently for months, because Claude Code does not
validate that a `statusLine` command exists.

## Layout

```
lib.sh          identity, cache reads, scoring, faces
statusline.sh   the status line renderer
probe.sh        git hygiene probe
record.sh       PostToolUse(Bash) test verdict recorder
quip.sh         Stop hook one-liner
card.sh         /buddy full readout
skill/SKILL.md  the /buddy skill
install.sh      copy into ~/.claude and patch settings.json
```

Scripts resolve `lib.sh` relative to themselves, so a checkout can be run and
tested in place without installing.
