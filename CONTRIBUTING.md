# Contributing

## Setup

**Requirements:** Go 1.26.2+, [Task](https://taskfile.dev), `git`

```bash
git clone https://github.com/louisiv/review-mono
cd review-mono
go mod download
task hooks-install
```

## Development workflow

```bash
task build          # build to ./bin/review
task run -- open    # run without installing (pass args after --)
task daemon         # run daemon in foreground (useful for dev)
task test           # run test suite
task lint           # run golangci-lint (installs pinned version automatically)
task hooks-install  # install the pre-push hook
```

## Project layout

```
cmd/          CLI commands (cobra)
internal/
  daemon/     HTTP server — all business logic lives here
  git/        Thin wrappers around the git binary
  store/      JSON persistence under .git/review/
  ai/         Anthropic API integration
  client/     HTTP client used by CLI commands to talk to the daemon
  config/     User config (~/.config/review/)
  tui/        BubbleTea terminal UI and widgets
  events/     SSE pub/sub event bus
main.go       Wires the cobra root command
```

See [docs/structure.md](docs/structure.md) for a deeper walkthrough and design rationale.

## Architecture notes

- **CLI → daemon pattern:** CLI commands are thin HTTP clients. All meaningful work happens in the daemon (`internal/daemon/`). Add a new feature by adding a handler there, then a command in `cmd/` that calls it.
- **State is plain JSON** in `.git/review/` — no database, no ORM. See [docs/data-models.md](docs/data-models.md).
- **Events** flow over SSE, not WebSockets. This keeps `review watch` curl-friendly and easy to test.
- **No go-git** — the project shells out to the `git` binary to keep things simple and avoid version drift.

## Code standards

These are enforced by CI (golangci-lint):

- Max file length: 500 lines
- Max line length: 120 characters
- Max cyclomatic complexity: 50
- Formatting: `gofmt` + `goimports`

Run `task lint` before pushing. CI will fail on lint errors.

The tracked pre-push hook runs `task test` and `task lint` before every push. Enable it once with:

```bash
task hooks-install
```

## Tests

```bash
task test
```

TUI widgets have visual snapshot tests. When you change a widget, CI will render before/after screenshots and post a diff comment on your PR automatically — no action needed from you.

## CI

| Workflow | Trigger | What it does |
|---|---|---|
| `golangci-lint` | PR, push to main | Lints the whole codebase |
| `widget-snapshots-pr` | PR | Renders widget snapshots, posts visual diff as PR comment |

## Architecture Decision Records

Significant design decisions are recorded in [`docs/adr/`](docs/adr/). If your change involves a meaningful architectural choice, add an ADR.

## Pull requests

1. Branch off `main`
2. `task test && task lint` must pass locally
3. Keep PRs focused — one concern per PR
4. Reference any related issue in the PR description
