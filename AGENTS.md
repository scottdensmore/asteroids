# AGENTS.md

Working agreement for any coding agent (Claude Code, Codex, Copilot, Gemini CLI,
Cursor, or a human) contributing to this repository. This file is the single
source of truth for the development workflow. Tool-specific files such as
`CLAUDE.md` point here rather than duplicating these rules.

## Project overview

An Asteroids arcade clone written in Go using
[Ebitengine](https://ebitengine.org/). Vector-style graphics, ship inertia,
screen wrapping, splitting asteroids, UFOs, and procedurally generated retro
audio.

- Module: `github.com/scottdensmore/asteroids` (see `go.mod`)
- Go version: pinned by `go.mod` (currently 1.24.10 or newer)
- Only external dependency: `github.com/hajimehoshi/ebiten/v2`
- The executable is intentionally thin; gameplay lives in an internal package.

The maintained file-by-file map lives in the
[contributing guide](CONTRIBUTING.md#project-structure).

`SPEC.md` describes intended game behavior. When a change affects gameplay,
reconcile it with `SPEC.md` and update the spec in the same pull request if the
intended behavior itself is changing.

## Environment and commands

Follow the canonical [development environment](CONTRIBUTING.md#development-environment)
setup and command suite. The build binary `asteroids` is git-ignored; never
commit it.

## Workflow rules

These rules are mandatory. Follow them in order.

### 1. Inspect before changing anything

Inspect the repository, current Git state, and all applicable instruction files
before making changes. Preserve unrelated staged, unstaged, and untracked work.
Never stash, reset, or discard changes you did not create.

### 2. Create a branch first

Create a dedicated branch before making code changes. Never commit directly to
`main`, and create the branch from the latest appropriate `main` state
(`git fetch origin main` first).

Branch naming: `<owner>/<type>/<short-description>`, for example
`scottdensmore/fix/release-publish-main`. Use one of the types `feat`, `fix`,
`refactor`, `chore`, `test`, or `docs`.

### 3. Choose a thin vertical slice

Before implementing a tracked issue or feature, define the smallest end-to-end
slice that can be reviewed, tested, shipped, and merged independently. Prefer
one coherent user-visible or operational outcome over a broad horizontal layer.
If the next issue is too large for one pull request, split it into ordered
slices and complete only the current slice. Keep pull requests small enough for
thorough review, reliable verification, and quick rollback.

### 4. Use test-driven development when behavior or structure is testable

- Add or update a focused test before implementation.
- Run it and confirm it fails for the expected reason.
- Implement the smallest appropriate change.
- Run focused tests while iterating (`go test -run TestName ./...`).
- Refactor only while the relevant tests remain green.

Pure logic — vector math, scoring, level progression, version metadata, sound
buffer generation — is testable and should be covered. Frame-by-frame rendering
and audio playback are not unit-testable here; verify those by running the game
and say so explicitly instead of skipping verification silently.

### 5. Inspect the complete diff

Review the branch diff plus all staged, unstaged, and untracked files
(`git diff origin/main...HEAD`, `git status --porcelain`). Remove accidental or
unrelated changes while preserving work that belongs to the user.

### 6. Run `ui-review` before verification

After the main agent completes an implementation pass, invoke the `ui-review`
sub-agent, acting as an expert in design, usability, responsiveness, and
accessibility. Address every actionable finding before running the `verifier`.

For UI-affecting changes, exercise the changed journey in the rendered
application, inspect interaction, loading, empty, error, focus, keyboard,
contrast, and responsive states as applicable, and capture screenshots or
equivalent visual evidence. This project is a fixed-viewport desktop game rather
than a web application: the equivalent review is running the game, exercising
the changed journey, and checking on-screen readability, contrast, input
responsiveness, and audio behavior. Phone and tablet viewports do not apply.

For changes with no UI impact, explicitly record that rendered UI review is not
applicable. If a finding is not applicable, record the concrete reason rather
than silently ignoring it.

### 7. Run `verifier` before code review

Invoke the `verifier` sub-agent to run the builds, static checks, tests, and
journey coverage appropriate for the change — at minimum `gofmt -l .`,
`go vet ./...`, `go build ./...`, and `go test ./...`. The verifier must report
failures, flakes, missing coverage, and environment issues. Fix or explicitly
resolve every actionable finding before starting code review. If a verifier
finding requires a code change, rerun the verifier after addressing it.

### 8. Run `code-review` before every commit

Invoke the `code-review` sub-agent against the current branch diff and every
staged, unstaged, and untracked file. The reviewer must act as an expert in Go
and Ebitengine. Address every actionable finding before committing. If review
findings cause changes, rerun the appropriate tests and the `verifier`, then
obtain a fresh `code-review` approval for the changed state.

If a named sub-agent is unavailable in the current tool, perform the same review
inline with equivalent rigor and state in the summary which agent was
substituted for.

### 9. Commit after approval

Commit only after verification and code review are complete. Use Conventional
Commits:

```text
<type>(<scope>): <imperative summary>
```

Keep the subject at 72 characters or fewer, describe why in the body when
useful, and do not combine unrelated work. Stage files explicitly; never
`git add -A` when unrelated work is present in the worktree.

### 10. Create pull requests from the reviewed state

- Confirm that local verification remains valid.
- Rerun `code-review` only if the reviewed state changed after the pre-commit
  review. A changed state includes code, tests, documentation, generated files,
  conflict resolution, or any other staged, unstaged, or untracked content.
- Do not repeat code review when the already-reviewed diff and worktree remain
  unchanged.
- Push and create the pull request only after local verification and any
  required code review are complete.
- Open a normal, ready-for-review pull request by default. Do not open draft
  pull requests unless the user explicitly asks for a draft.

### 11. Merge only clean, passing pull requests

Merge only after GitHub reports a clean merge state and every configured check
passes. Never bypass a failing or pending required check. Self-merges are
allowed when these conditions are met. Use squash merge for short-lived
development branches to keep `main` linear, then delete the merged branch.

## GitHub operations

Use the GitHub CLI (`gh`) for all GitHub operations — pull requests, issues, CI
status, releases. Do not use the web UI or raw API calls.

```bash
gh pr create --title "<type>(<scope>): <summary>" --body "<why>"
gh pr checks --watch
gh pr merge --squash --delete-branch
```

## Go conventions

- Keep `cmd/asteroids` limited to process startup and window configuration.
  Gameplay, rendering, audio, and build metadata belong in `internal/game`.
- Add another package only when it represents a cohesive boundary and reduces
  coupling; do not split the game into packages solely to create directories.
- Methods that mutate game state hang off `*Game`; audio behavior belongs on
  `*SoundManager`.
- Prefer small, focused helpers over growing `Update` and `Draw`.
- Comment density matches the surrounding file: short comments for non-obvious
  math and timing constants, none for self-evident code.
- Handle errors explicitly; the game loop returns `error` from `Update` for
  termination.
- Magic numbers that tune gameplay belong in the `const` blocks at the top of
  the relevant file, not inline.

## Release process

Release behavior and version metadata are documented in the canonical
[release automation guide](CONTRIBUTING.md#release-automation) and implemented
by `.github/workflows/release.yml`.

## Documentation conventions

- Keep `README.md` focused on the game: its presentation, features, controls,
  play instructions, releases, and concise links to supporting documents.
- Put development setup, repository structure, and release automation in
  `CONTRIBUTING.md`; keep mandatory agent workflow and Go conventions here.
  Put intended gameplay behavior in `SPEC.md`, human attribution in
  `CONTRIBUTORS.md`, credits in `ACKNOWLEDGMENTS.md`, and license terms in
  `LICENSE`.
- Update the owning document instead of duplicating detailed guidance across
  files that can drift apart.

## Repository tooling

`.claude/` and `.codex/` hold agent hook configuration for the
[Entire](https://docs.entire.io/) CLI, which records checkpoints and transcripts.
The hooks no-op when `entire` is not installed. `.entire/metadata/` is not
readable by agents by design — do not attempt to work around that restriction.
