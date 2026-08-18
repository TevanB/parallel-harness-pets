<!-- Delete anything that does not apply. A one-line PR is welcome. -->

## What this changes

## Why

## Checks

- [ ] `go test ./...` passes
- [ ] `gofmt -l .` prints nothing

If the change touches `internal/identity`, the golden test pins branch names to
creatures on purpose. Repinning it is fine when the reshuffle is deliberate —
say so here, so a reviewer knows it was not an accident.

If it adds a creature, please confirm it reads at a glance in a real terminal at
both ends of the mood range, not just in the diff.
