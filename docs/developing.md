# Developing
This document describes the basics of hacking on Substrate. If you are only interested in using Substrate to create and manage infrastructure, see the [Getting Started guide](getting-started.md) instead.

### Install Dependencies

Developing on Substrate requires a couple of dependencies:

 - `go` -- the Go compiler (`brew install go` on OS X).

 - `glide` -- a Go package manager (`brew install glide` on OS X).

 - `shellcheck` -- a static analyzer for shell scripts (`brew install shellcheck` on OS X).

### Build

 - To build for your current OS/architecture, run `make`. Your output binary should end up in `./bin/substrate`.

 - To build for all target architectures, run `make build-release`. To build for some specific target, set `RELEASE_TARGET`, e.g., `make build-release RELEASE_TARGETS=openbsd-386`.

 - To run a suite of lint tools, run `make lint`. This will also complain about a bunch of style/formatting problems. Many of these can be fixed automatically by running `make fmt`.

 - To re-pin Go package versions, optionally edit `glide.yaml` then run `make update-deps`.

### Release

Substrate is released as a binary package to ensure that everyone is using exactly the same code. Official (internal) releases are built and tested in a special dedicated Jenkins cluster (internal only).

Every package carries a version number and a description of the Git commit from which it was built. At build time the version number comes from a special `VERSION` file.

If the version in `VERSION` does not match a Git tag, it has `-snapshot` appended to its name to indicate that it's a pre-release build. If the binary is built from a modified working directory, the Git commit hash will have `-dirty` appended to indicate that local modifications are included in the build.

To cut a release:
 - Edit `CHANGELOG.md` to reflect major changes since the last release.
 - Ensure that the `VERSION` file matches the intended release version (following SemVer as mentioned above).
 - Tag the release (`git tag $(cat VERSION)`) and push your tag (`git push origin $(cat VERSION)`).
 - Create a GitHub release by visiting https://github.com/SimpleFinance/substrate/tags and clicking `Add release notes`.
 - Bump the version number in `VERSION` to the next expected version and push that to `master`. This will affect the version number of `*-snapshot` builds until the next release.
