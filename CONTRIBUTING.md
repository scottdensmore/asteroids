# Contributing to Asteroids

Thank you for helping improve Asteroids.

## Development environment

Install the Go version pinned in [go.mod](go.mod). Ebitengine needs a graphics
stack. On Linux, install the development packages listed in the
`Install Linux build dependencies` step of the
[CI workflow](.github/workflows/ci.yml) and run graphical commands
under `xvfb-run -a`. macOS and Windows need no additional packages.

Run the local quality suite before opening a pull request:

```bash
go mod download
gofmt -l .
go vet ./...
go build ./...
go test ./...
```

`gofmt -l .` must produce no output. Format changed Go files with
`gofmt -w <files>`.

## Project structure

| Path | Purpose |
| --- | --- |
| `cmd/asteroids/main.go` | Executable entry point and Ebitengine window setup |
| `internal/game/game.go` | Game setup, `Update`, spawning, collisions, and input |
| `internal/game/draw.go` | `Draw`, `Layout`, and vector rendering helpers |
| `internal/game/types.go` | Game state and entity types |
| `internal/game/ui.go` | Scaled retro text measurement, caching, and rendering helpers |
| `internal/game/sound.go` | Procedural audio synthesis and playback |
| `internal/game/version.go` | Build metadata injected with `-ldflags` |
| `internal/game/*_test.go` | Tests kept beside the game code they exercise |
| `README.md` | Player-facing game overview, controls, and play instructions |
| `CONTRIBUTING.md` | Development setup, repository structure, and release automation |
| `ACKNOWLEDGMENTS.md` | Inspiration, foundational tools, and automated assistance |
| `LICENSE` | MIT license terms |
| `.github/workflows/ci.yml` | Formatting, vet, build, and tests without publishing |
| `.github/workflows/package-artifacts.yml` | PR/manual build, test, and artifact packaging; no release publication |

## Build and test CI

The CI workflow checks formatting, runs vet, builds packages, and runs tests on
Linux, macOS, and Windows for pull requests and pushes to `main`. It can also be
run manually without inputs. Linux tests run under Xvfb. CI has read-only
repository permissions, does not retain Git credentials, and does not call the
release workflow, create tags, upload packages, or publish releases.

## Release automation

Release builds use Semantic Version tags such as `v1.2.3`, with optional
prerelease or build metadata. The GitHub Actions workflow tests Linux, macOS,
and Windows, embeds version metadata, packages each binary, and publishes a
release only after every platform build succeeds.

Dependabot keeps Go modules and GitHub Actions current. Pull requests run the
separate cross-platform CI workflow without publishing a release.
