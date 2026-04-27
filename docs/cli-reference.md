---
type: Note
belongs_to: "[[local-git-reviews]]"
_organized: true
---
# CLI Reference

Every command that produces output supports `--json` for machine-readable output.\
\
Exit codes: `0` success, `1` expected error (no session, etc), `2` unexpected error.

***

## Global Flags

```text
--json          Output as JSON instead of human-readable text
--repo PATH     Target repo (default: current directory)
--port PORT     Daemon port (default: 7080, overrides config)
```

***

## Daemon

### `review daemon start`

Start the review daemon in the background.

```bash
review daemon start
review daemon start --port 7080
```

Output:

```text
Daemon started on port 7080 (pid 12345)
```

### `review daemon stop`

Stop the running daemon.

### `review daemon status`

Show whether daemon is running.

```json
{ "running": true, "port": 7080, "pid": 12345 }
```

***

## Session

### `review open`

Open a review session for the current branch. Starts daemon if not running.

```bash
review open
review open --base develop
review open /path/to/repo
review open /path/to/repo --base develop
```

Output:

```text
Opened review session for my-feature → main
3 commits, 12 files changed
```

```json
{
  "id": "uuid",
  "branch": "my-feature",
  "base": "main",
  "commits": 3,
  "files_changed": 12,
  "status": "in_review"
}
```

### `review status`

Show current session status.

```bash
review status
```

Output:

```text
Branch:   my-feature → main
Status:   in_review
Commits:  3
Comments: 2 open, 1 resolved
```

```json
{
  "branch": "my-feature",
  "base": "main",
  "status": "in_review",
  "commits": 3,
  "comments": { "open": 2, "resolved": 1 }
}
```

### `review close`

Close and clean up the session without pushing.

***

## Diff

### `review diff`

Show full diff for the session.

```bash
review diff
review diff --file src/auth/login.go
review diff --commit abc1234
```

Output: standard diff format (colorized in terminal, plain in `--json`)

```json
{
  "files": [
    {
      "path": "src/auth/login.go",
      "additions": 12,
      "deletions": 3,
      "hunks": [...]
    }
  ]
}
```

### `review commits`

List commits in the review.

```bash
review commits
```

Output:

```text
abc1234  Add login endpoint         Claude  2 hours ago
def5678  Fix null check in parser   Claude  1 hour ago
```

```json
{
  "commits": [
    {
      "hash": "abc1234",
      "message": "Add login endpoint",
      "author": "Claude",
      "timestamp": "..."
    }
  ]
}
```

## Comments

### `review comment add`

Add an inline comment.

```bash
review comment add src/auth/login.go:42 "Use bcrypt here not md5"
review comment add src/auth/login.go:42 "Use bcrypt here not md5" --author agent
```

```json
{
  "id": "uuid",
  "file": "src/auth/login.go",
  "line": 42,
  "body": "Use bcrypt here not md5",
  "author": "human",
  "resolved": false
}
```

### `review comment list`

List all comments, optionally filtered.

```bash
review comment list
review comment list --unresolved
review comment list --file src/auth/login.go
review comment list --author agent
```

```json
{
  "comments": [
    {
      "id": "uuid",
      "file": "src/auth/login.go",
      "line": 42,
      "body": "Use bcrypt here not md5",
      "author": "human",
      "resolved": false,
      "created_at": "..."
    }
  ]
}
```

### `review comment resolve`

Mark a comment as resolved.

```bash
review comment resolve <id>
review comment resolve --all
```

### `review comment delete`

Delete a comment.

```bash
review comment delete <id>
```

***

## Description

### `review describe`

Generate an MR description from the diff using Claude API.\
\
Saves result to `.git/review/description.md`.

```bash
review describe
review describe --print     # print to stdout, don't save
```

Output:

```text
Generated MR description (saved to .git/review/description.md)

## Summary
Adds login endpoint with password hashing...
```

### `review description show`

Print the current saved description.

### `review description edit`

Open description in `$EDITOR`.

***

## Actions

### `review approve`

Push branch to origin and open browser with pre-filled MR/PR URL.

```bash
review approve
review approve --no-browser    # push only, skip browser
review approve --dry-run       # show what would happen
```

Output:

```text
Pushing my-feature to origin...
Opening MR creation page...
https://github.com/org/repo/compare/my-feature?body=...
```

### `review request-changes`

Mark session as changes requested. Does not push. Agents watching\
\
via `review watch` will receive a `changes_requested` event.

```bash
review request-changes
review request-changes --message "See comments on auth module"
```

***

## Watch

### `review watch`

Stream session events as newline-delimited JSON. Blocks until session closes.\
\
Intended for agents.

```bash
review watch
review watch --event comment_added      # filter to specific event type
```

Output (newline-delimited JSON):

```text
{"event":"comment_added","comment_id":"uuid","timestamp":"..."}
{"event":"changes_requested","timestamp":"..."}
{"event":"approved","timestamp":"..."}
```
