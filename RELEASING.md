# Releasing

```sh
git tag -a vX.Y.Z -m "…" && git push origin vX.Y.Z
```

That is the whole thing. The release workflow builds every platform and
publishes the archives and checksums.

## The Homebrew cask updates itself

[The tap](https://github.com/TevanB/homebrew-tap) carries a workflow that polls
this repository's latest release every six hours, and can be run on demand:

```sh
gh workflow run update-casks.yml --repo TevanB/homebrew-tap
```

It regenerates the cask from the release's `checksums.txt` and pushes.

**No token is involved.** A workflow in this repository could not write to the
tap without a personal access token, but a workflow in the tap can write to the
tap with its own `GITHUB_TOKEN`. Pulling from the release rather than pushing to
the tap removes the secret entirely, which is why `.goreleaser.yaml` keeps
`skip_upload` set on the cask and the scoop manifest.

It also makes the checksum rule structural rather than a thing to remember: the
updater has no local build to take hashes from, only the published
`checksums.txt`. That matters because the build is not byte-reproducible, so
locally generated hashes match nothing anyone downloads and every `brew install`
would fail verification.

## Platforms

Homebrew casks are macOS only. Linux and Windows users take a release archive or
`go install`. The Linux blocks GoReleaser writes into the cask are unreachable,
and harmless.

## Checks

The tap runs `brew audit --cask --strict` on a macOS runner on every push, then
installs the cask and runs the binary, so an audit cannot pass while the thing
it describes is unusable.
