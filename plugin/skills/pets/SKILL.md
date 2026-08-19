---
name: pets
description: "Show this worktree's creature and the branch-hygiene signals behind its mood. Use when the user types /pets."
argument-hint: "[show|why]"
allowed-tools: Bash(pets:*)
---

# Pets

Run the card and show it. Nothing else.

```sh
pets card
```

## Output rule

The script prints a pre-formatted card with ANSI colours and box characters.
**Output it exactly as returned, character for character**, in a fenced block so
the terminal renders the spacing. Do not summarise it, describe the creature in
prose, add commentary, or strip escape codes.

`pets card` is typed by a person, so there is no agent session to key on and the
creature comes from the branch. A live agent's status line keys on its session id
instead, which is why one worktree can show several different creatures at once.
Either way it is derived, never picked or renamed.

If `$ARGUMENTS` is `why`, additionally name the signals shown in warning colour
and, for each, the concrete command that clears it: `git commit` for
uncommitted files, `git push` for unpushed commits, and for split migration
heads, whatever that project uses to merge them.
