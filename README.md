# review

A lightweight CLI tool and local daemon for reviewing AI-generated code before pushing. Browse commits and diffs, leave inline comments, generate MR descriptions with Claude, then approve and push — all before touching your remote.

## Install

**Prerequisites:** Go 1.26.2+, [Task](https://taskfile.dev)

```bash
git clone https://github.com/louisiv/review-mono
cd review-mono
task install          # installs to $HOME/.local/bin
```

To install to a custom location:

```bash
INSTALL_DIR=/usr/local/bin task install
```

## Usage

Start a daemon in your repo, then run review commands against it:

```bash
# Start the background daemon (one per machine, persists across sessions)
review daemon start

# Open a review session for the current branch
review open

# Browse and comment
review diff
review commits
review comment add path/to/file.go:42 "this looks off"
review comment list

# Generate an MR description with Claude (requires ANTHROPIC_API_KEY)
review describe

# Ship it or send it back
review approve          # pushes to origin
review request-changes  # flags for rework, stays local
```

Launch the interactive TUI instead:

```bash
review tui
```

Every command accepts `--json` for script/agent consumption, and `review watch` streams live events as newline-delimited JSON.

## Configuration

`review` reads `~/.config/review/config.json`. The Anthropic API key can also be set via `ANTHROPIC_API_KEY` in your environment.

## Docs

- [CLI reference](docs/cli-reference.md)
- [TUI guide](docs/tui-review.md)
- [Daemon HTTP API](docs/daemon-api.md)
- [Data models](docs/data-models.md)
