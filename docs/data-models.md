---
type: Note
belongs_to: "[[local-git-reviews]]"
_organized: true
---
# Data Models

All state is persisted under `.git/reviews/` in the target repository.

## Directory Layout

```text
.git/reviews/
  sessions.json         # all active review sessions
  sessions/
    [id]/
      comments.json  # all comments for this session
      description.md # MR description (if any)
```

***

## Sessions

**File:** `.git/reviews/sessions.json`

```json
[{
  "id": "uuid",
  "repo": "/absolute/path/to/repo",
  "branch": "my-feature",
  "base": "main",
  "status": "in_review",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "approved_at": null,
}, ...]
```

**Status values:**

- `in_review` — open, awaiting approval
- `changes_requested` — rejected, back to agent
- `approved` — pushed to origin

***

## Comment

**File:** `.git/reviews/sessions/:id/comments.json` — array of Comment objects

```json
[
  {
    "id": "uuid",
    "file": "src/auth/login.go",
    "line": 42,
    "body": "Use bcrypt here, not md5",
    "author": "human",
    "resolved": false,
    "created_at": "2026-01-01T00:00:00Z",
    "resolved_at": null
  }
]
```

**Author values:**

- `human` — left by a human via TUI or CLI
- `agent` — left by an AI agent via CLI
  - Could also be Claude, Codex, Amp, etc.

***

## Commit (in-memory, not persisted)

Derived from git log at request time.

```json
{
  "hash": "abc1234",
  "hash_full": "abc1234def5678...",
  "author": "Claude",
  "message": "Add login endpoint",
  "timestamp": "2026-01-01T00:00:00Z"
}
```

***

## DiffFile (in-memory, not persisted)

Derived from git diff at request time.

```json
{
  "path": "src/auth/login.go",
  "additions": 12,
  "deletions": 3,
  "hunks": [
    {
      "header": "@@ -10,6 +10,12 @@",
      "lines": [
        { "type": "context", "number": 10, "content": "func Login() {" },
        { "type": "add",     "number": 11, "content": "  hash := md5(pass)" },
        { "type": "remove",  "number": null, "content": "  hash := plain(pass)" }
      ]
    }
  ],
  "comments": []
}
```

Line types: `add`, `remove`, `context`

***

## Event (streamed, not persisted)

Emitted by `review watch` as newline-delimited JSON.

```json
{ "event": "comment_added",       "comment_id": "uuid", "timestamp": "..." }
{ "event": "comment_resolved",    "comment_id": "uuid", "timestamp": "..." }
{ "event": "changes_requested",   "timestamp": "..." }
{ "event": "approved",            "timestamp": "..." }
{ "event": "description_updated", "timestamp": "..." }
{ "event": "session_opened",      "timestamp": "..." }
```

***

## Config

**File:** `~/.config/review/config.json` — global user config

```json
{
  "anthropic_api_key": "sk-...",
  "default_base_branch": "main",
  "daemon_port": 7080,
  "open_browser_on_approve": true,
  "github_host": "github.com",
  "gitlab_host": "gitlab.example.com"
}
```

API key can also be set via `ANTHROPIC_API_KEY` env var (takes precedence).
