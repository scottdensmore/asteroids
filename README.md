# Asteroids

[![Build and Release](https://github.com/scottdensmore/asteroids/actions/workflows/release.yml/badge.svg)](https://github.com/scottdensmore/asteroids/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/release/scottdensmore/asteroids?sort=semver)](https://github.com/scottdensmore/asteroids/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/scottdensmore/asteroids)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Ebitengine](https://img.shields.io/badge/powered%20by-Ebitengine-00a4de)](https://ebitengine.org/)

A vector-style recreation of the 1979 arcade classic, built in Go with
[Ebitengine](https://ebitengine.org/). Drift through a wrapping playfield,
split asteroids, dodge increasingly accurate saucers, and chase a high score
against a procedurally generated retro soundtrack.

![Asteroids-inspired arcade cabinet](img/gemini.jpeg)

## Features

- Momentum-based ship movement with screen wrapping
- Large, medium, and small asteroids with progressive splitting
- Big and small flying saucers with distinct movement, firing, and scoring
- Score-based difficulty, level progression, extra ships, and respawning
- Monochrome vector-style rendering with particles and invincibility effects
- Procedurally generated stereo audio with no external sound assets
- Version, commit, and build metadata embedded in release binaries
- Cross-platform release packages for Linux, macOS, and Windows

## Quick start

Install [Go 1.24.10 or newer](https://go.dev/doc/install), then clone and run
the project:

```bash
git clone https://github.com/scottdensmore/asteroids.git
cd asteroids
go run ./cmd/asteroids
```

Ebitengine opens the game in an 800 x 600 desktop window. Linux systems also
need the graphics and audio development packages documented in the
[release workflow](.github/workflows/release.yml).

## Controls

| Key | Action |
| --- | --- |
| <kbd>Enter</kbd> | Start or restart the game |
| <kbd>Left Arrow</kbd> / <kbd>Right Arrow</kbd> | Rotate the ship |
| <kbd>Up Arrow</kbd> | Apply thrust |
| <kbd>Space</kbd> | Fire |

## Releases

Download ready-to-run archives from the
[latest GitHub release](https://github.com/scottdensmore/asteroids/releases/latest).
Each release contains a platform-specific binary:

- Linux: `asteroids-linux-amd64-<version>.tar.gz`
- macOS: `asteroids-macos-<version>.tar.gz`
- Windows: `asteroids-windows-amd64-<version>.zip`

Release builds are created from Semantic Version tags such as `v1.2.3`, with
optional prerelease or build metadata supported. The GitHub Actions workflow
tests all three platforms, embeds version metadata, packages the binaries, and
publishes the release only after every platform build succeeds.

## Development

Run the complete local quality suite before opening a pull request:

```bash
gofmt -l .
go vet ./...
go build ./...
go test ./...
```

`gofmt -l .` must produce no output. On Linux, install the dependencies listed
in the workflow and run graphical tests with `xvfb-run -a go test ./...`.

The repository uses Dependabot for Go module and GitHub Actions updates. Pull
requests are checked on Linux, macOS, and Windows by the same workflow used to
produce releases.

## Project structure

| Path | Purpose |
| --- | --- |
| `cmd/asteroids/main.go` | Thin executable entry point and window setup |
| `internal/game/game.go` | Game loop, input, movement, collisions, and spawning |
| `internal/game/draw.go` | Vector rendering and Ebitengine layout |
| `internal/game/types.go` | Game state and entity types |
| `internal/game/ui.go` | Scaled retro text measurement, caching, and rendering |
| `internal/game/sound.go` | Procedural audio synthesis and playback |
| `internal/game/version.go` | Build metadata displayed in the game |
| `internal/game/*_test.go` | Gameplay, audio, UI, and version unit tests |
| `SPEC.md` | Intended gameplay behavior |
| `CONTRIBUTORS.md` | Project maintainers and contributors |
| `.github/workflows/release.yml` | CI, packaging, and release automation |

## Contributors

See [CONTRIBUTORS.md](CONTRIBUTORS.md) for the people who maintain and improve
the project.

## Acknowledgments

This independent learning project is inspired by Atari's original *Asteroids*
and was initially developed interactively with the Gemini CLI. Ebitengine and
Go make the cross-platform game loop, rendering, input, and audio possible.

## License

Licensed under the [MIT License](LICENSE).
