---
name: buddy
description: "Show this worktree's familiar and the branch-hygiene signals behind its mood. Use when the user types /buddy."
argument-hint: "[show|why]"
allowed-tools: Bash(~/.claude/claude-buddy/card.sh:*)
---

# Buddy

Run the card and show it. Nothing else.

```sh
~/.claude/claude-buddy/card.sh "$(pwd)"
```

## Output rule

The script prints a pre-formatted card with ANSI colours and box characters.
**Output it exactly as returned, character for character**, in a fenced block so
the terminal renders the spacing. Do not summarise it, describe the creature in
prose, add commentary, or strip escape codes.

The creature is a pure function of the branch name, so it is not something the
user picks or renames. If they ask why it changed, the answer is that the branch
changed.

If `$ARGUMENTS` is `why`, additionally name the signals shown in warning colour
and, for each, the concrete command that clears it (`git commit`, `git push`,
`/fix-alembic-heads` for split alembic heads).
