# Contributing

Two things are easy to contribute and neither needs much Go.

## Adding a creature

Creatures live in one slice in `internal/identity/identity.go`. Adding one is a
single line:

```go
{"newt", "~[", "]~", Uncommon},
```

That is a name, the two fragments that bracket its face, and its rarity band.
`~[` and `]~` around the face `•ᴗ•` render as `~[•ᴗ•]~`.

Rules, all enforced by `go test ./internal/identity/`:

- The name must be unique.
- Neither fragment may be longer than three characters, or rows stop lining up.
- The band must be one of `Common`, `Uncommon`, `Rare`, `Legendary`, `Mythic`.
- The creature must be original. No third-party characters, however loosely.

Rarity odds come from a weight table, not from how many creatures sit in a band,
so adding ten legendaries does not make legendaries common. It only makes each
one rarer to draw.

Please check your creature reads at a glance in a terminal, not just in a diff.
Faces range from `•ᴗ•` down to `x_x`, so try it at both ends.

## Adding a signal

A signal is any executable that prints `key=value` lines on stdout. It receives
the worktree path as its first argument. Drop it in `~/.config/pets/signals/`.

```sh
#!/bin/sh
# ~/.config/pets/signals/cargo.sh
echo "clippy=$(cargo clippy --message-format=short 2>&1 | grep -c '^warning')"
```

Signals are user configuration rather than part of the tool, so there is nothing
to submit unless you want one documented in the README as an example.

## Working on the code

```sh
go test ./...
go build ./cmd/pets
```

The binary reads and writes three locations, all redirectable, so you can
exercise a build without touching your own collection or agent config:

```sh
export HOME=/tmp/pets-sandbox
export XDG_CONFIG_HOME=/tmp/pets-sandbox/config
export XDG_STATE_HOME=/tmp/pets-sandbox/state
```

`HOME` is what `pets install` writes into, and the two XDG paths hold config and
the collection. Nothing else on disk is touched.

Two things worth knowing before changing them:

**Identity must stay stable.** A branch is supposed to summon the same creature
forever, on any machine. `internal/identity` has a golden test pinning specific
branch names to specific creatures, and it is there to make an accidental
reshuffle loud. If you change identity deliberately, repin it and say so.

**The render path never runs git and never touches the network.** `pets render`
executes once a second in every open session, so it reads two small files and
nothing else. The expensive work happens in `pets probe`, backgrounded and
throttled. Update checks run only from `card`, `party` and `den`, which a person
types.
