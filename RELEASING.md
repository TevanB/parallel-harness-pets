# Releasing

```sh
git tag -a vX.Y.Z -m "…" && git push origin vX.Y.Z
```

The release workflow builds every platform and publishes the archives.

Homebrew casks are macOS only, so the cask serves Mac users and everyone else
takes an archive or `go install`. The Linux blocks GoReleaser writes into the
cask are unreachable, and harmless.

## Publishing the Homebrew cask

The cask is currently pushed by hand, because CI has no token with write access
to `TevanB/homebrew-tap`. `.goreleaser.yaml` therefore sets `skip_upload` on the
cask and the scoop manifest, so a missing token can never fail the whole release
and leave users with nothing.

After the release finishes, update
[the tap](https://github.com/TevanB/homebrew-tap)'s
`Casks/parallel-harness-pets.rb`: bump `version`, and replace each `sha256` with
the matching line from the release's `checksums.txt`.

**Take the checksums from the published `checksums.txt`, never from a local
rebuild.** The build is not byte-reproducible, so a locally generated cask
carries hashes that no downloaded artifact matches and every `brew install`
fails verification.

To automate it later, create a fine-grained personal access token with contents
write access to `homebrew-tap` only, add it as the `TAP_GITHUB_TOKEN` secret,
and drop the two `skip_upload` lines. Prefer a fine-grained token over a
general-purpose one: this repository is public, and a broad token in its secrets
would be reachable from any workflow in it.
