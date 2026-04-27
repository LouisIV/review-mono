# Project Structure

```
review/
├── main.go                    # entrypoint, wires cobra root command
├── go.mod
├── go.sum
│
├── cmd/                       # CLI commands (cobra)
│   ├── root.go                # root command, global flags, --json flag
│   ├── daemon.go              # daemon start/stop/status
│   ├── open.go                # review open
│   ├── status.go              # review status
│   ├── close.go               # review close
│   ├── diff.go                # review diff
│   ├── commits.go             # review commits
│   ├── comment.go             # review comment add/list/resolve/delete
│   ├── describe.go            # review describe / description show / edit
│   ├── approve.go             # review approve
│   ├── request_changes.go     # review request-changes
│   └── watch.go               # review watch
│
├── internal/
│   │
│   ├── daemon/                # HTTP server
│   │   ├── server.go          # server setup, route registration, start/stop
│   │   ├── handlers/
│   │   │   ├── session.go     # GET/POST/DELETE /session
│   │   │   ├── commits.go     # GET /commits
│   │   │   ├── diff.go        # GET /diff
│   │   │   ├── comments.go    # GET/POST/PATCH/DELETE /comments
│   │   │   ├── description.go # GET /description, POST /description/generate
│   │   │   ├── actions.go     # POST /approve, POST /request-changes
│   │   │   ├── events.go      # GET /events (SSE)
│   │   │   └── health.go      # GET /health
│   │   └── middleware.go      # logging, error handling
│   │
│   ├── git/                   # git operations (shell out to git binary)
│   │   ├── repo.go            # open repo, validate, get current branch
│   │   ├── diff.go            # parse diff output into DiffFile structs
│   │   ├── commits.go         # git log parsing
│   │   └── push.go            # git push origin HEAD
│   │
│   ├── store/                 # persistence (.git/review/)
│   │   ├── store.go           # open/init store for a repo path
│   │   ├── session.go         # read/write session.json
│   │   ├── comments.go        # read/write comments.json
│   │   └── description.go     # read/write description.md
│   │
│   ├── ai/                    # Claude API integration
│   │   ├── client.go          # Anthropic API client setup
│   │   └── describe.go        # build prompt from diff, stream response
│   │
│   ├── events/                # SSE event bus
│   │   └── bus.go             # in-memory pub/sub, fan-out to SSE clients
│   │
│   ├── models/                # shared data structs
│   │   ├── session.go
│   │   ├── comment.go
│   │   ├── commit.go
│   │   ├── diff.go
│   │   └── event.go
│   │
│   ├── client/                # HTTP client for CLI → daemon
│   │   └── client.go          # typed methods matching daemon API surface
│   │
│   └── config/                # user config
│       └── config.go          # load ~/.config/review/config.json + env vars
│
└── docs/
    ├── SCOPE.md
    ├── DATA_MODELS.md
    ├── CLI_REFERENCE.md
    ├── DAEMON_API.md
    └── PROJECT_STRUCTURE.md
```

---

## Key Dependencies

```
github.com/spf13/cobra          # CLI framework
github.com/go-chi/chi           # HTTP router (lightweight)
github.com/anthropics/anthropic-sdk-go  # Claude API
```

No ORM, no database driver, no heavy frameworks. State is plain JSON files.

---

## Design Decisions

**Shell out to git binary** rather than use go-git. The git binary is always present,
handles edge cases well, and parsing its output is straightforward for the operations
we need. go-git adds complexity without much benefit at this scope.

**SSE not WebSockets** for events. Simpler, works with `curl`, no special client
library needed. One-directional (server → client) which is all we need.

**chi not gorilla/mux** — lighter, idiomatic, good middleware story.

**Single daemon per machine** — the port is global.

**No TUI in this repo (yet)** — the TUI is a separate binary that imports `internal/client`.
Could live in `cmd/tui/` or a separate repo. The daemon API is the contract.

---

## Startup Flow

```
review open
  → check if daemon running (GET /health)
  → if not: fork daemon process, wait for /health to respond
  → POST /session {repo, base}
  → print status summary
```

```
review daemon start
  → write pid to ~/.config/review/daemon.pid
  → start HTTP server on configured port
  → log to ~/.config/review/daemon.log
```
