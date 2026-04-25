---
type: Note
belongs_to: "[[local-git-reviews]]"
_organized: true
---
# Daemon API

Local HTTP server, default port `7080`. All responses are JSON.\
\
No authentication — localhost only.

Base URL: `http://localhost:7080`

***

## Session

### `GET /sessions`

Lists all active sessions. Supports filters via query parameters:

|                  |            |                                                                                        |
| ---------------- | ---------- | -------------------------------------------------------------------------------------- |
| Query Param      | Type       | Description                                                                            |
| `repo_uri`       | `string`   | One of the following: - The absolute path of the repo on disk - The git remote URL     |
| `branch`         | `string`   | The branch the review is on (eg: `my-feature`)                                         |
| `base`           | `string`   | The branch the review is against (eg: `main`)                                          |
| `status`         |            | Status of the session                                                                  |
| `created_at__gt` | `DateTime` | Return sessions where the `created_at` datetime is greater than the provided date time |
| `created_at__lt` | `DateTime` | Return sessions where the `created_at` datetime is less than the provided date time    |
| `updated_at__gt` | `DateTime` | Return sessions where the `updated_at` datetime is greater than the provided date time |
| `updated_at__lt` | `DateTime` | Return sessions where the `updated_at` datetime is less than the provided date time    |

> Will be exanded as sessions support more data

Session fillters are an implied `AND`.

### `GET /session/:id`

Get a session's state by it’s ID.

**Response:**

```json
{
  "id": "uuid",
  "repo": "/path/to/repo",
  "branch": "my-feature",
  "base": "main",
  "status": "in_review",
  "created_at": "...",
  "updated_at": "..."
}
```

### `POST /session`

Open a new session.

**Request:**

```json
{
  "repo": "/path/to/repo",
  "base": "main"
}
```

**Response:** Session object

### `DELETE /session/:id`

Close a session without pushing.

***

## Commits

### `GET /session/:session_id/commits`

List commits in the review (branch vs base).

**Response:**

```json
{
  "commits": [
    {
      "hash": "abc1234",
      "hash_full": "abc1234...",
      "author": "Claude",
      "message": "Add login endpoint",
      "timestamp": "..."
    }
  ]
}
```

***

## Diff

### `GET /session/:session_id/diff`

Full diff for the session. Paginated.

**Query params:**

- `file` — filter to a single file path
- `commit` — diff for a single commit
- `skip_hunks` — Skip returning hunks

**Response:**

```json
{
  "files": [
    {
      "path": "src/auth/login.go",
      "additions": 12,
      "deletions": 3,
      "hunks": [
        {
          "header": "@@ -10,6 +10,12 @@",
          "lines": [
            { "type": "context", "number": 10, "content": "func Login() {" },
            { "type": "add",     "number": 11, "content": "  hash := bcrypt(pass)" },
            { "type": "remove",  "number": null, "content": "  hash := md5(pass)" }
          ]
        }
      ]
    }
  ]
}
```

***

## Comments

### `GET /session/:session_id/comments`

List comments.

**Query params:**

- `resolved` — `true` / `false` / omit for all
- `file` — filter by file path
- `author` — `human` or `agent`

**Response:**

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
      "created_at": "...",
      "resolved_at": null
    }
  ]
}
```

### `POST /session/:session_id/comments`

Add a comment. Supports the following options:

|          |              |          |                                                                                                                                            |
| -------- | ------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `file`   | `string`     | Required | The path to the file relative to the root of the repo                                                                                      |
| `lines`  | `[int, int]` | Optional | The starting line of the comment and the ending line of the comment. If not provided the comment is treated as feedback on the entire file |
| `body`   | `string`     | Required | Body of the comment in markdown                                                                                                            |
| `author` | `string`     | Required | Used to indicate who made the comment                                                                                                      |

**Request:**

```json
{
  "file": "src/auth/login.go",
  "line": 42,
  "body": "Use bcrypt here not md5",
  "author": "human"
}
```

**Response:** Comment object

### `PATCH /session/:session_id/comments/:comment_id`

Update a comment (resolve or edit body).

**Request:**

```json
{
  "resolved": true
}
```

**Response:** Updated comment object

### `DELETE /session/:session_id/comments/:comment_id`

Delete a comment.

***

## Description

### `GET /session/:session_id/description`

Get the current MR description.

**Response:**

```json
{
  "body": "## Summary\n\nAdds login endpoint...",
  "generated_at": "..."
}
```

Returns `404` if no description generated yet.

### `POST /session/:session_id/description`

Set the current MR description.

**Payload:**

```json
{
  "body": "## Summary\n\nAdds login endpoint...",
}
```

### `POST /session/:session_id/description/generate`

Trigger AI description generation. Streams response as newline-delimited JSON chunks. Supports an optional prompt to support generation:

```json
{"prompt":"pay special attention to ..."}
```

**Response (streamed):**

```text
{"chunk": "## Summary\n"}
{"chunk": "Adds login endpoint with"}
{"chunk": " password hashing...\n"}
{"done": true, "generated_at": "..."}
```

### `POST /session/:session_id/description/generate/edit`

Ask the AI model to update the description. Supports an optional prompt and an optional highlighted selection:

**Payload:**

```json
{"prompt":"pay special attention to ... “, “highlight”: “## Summary\nAdds login endpoint ..."}
```

**Response (streamed):**

```text
{"chunk": "## Summary\n"}
{"chunk": "Adds login endpoint with"}
{"chunk": " password hashing...\n"}
{"done": true, "generated_at": "..."}
```

***

## Actions

### `POST /approve`

Push branch to origin. Returns the MR creation URL.

**Response:**

```json
{
  "pushed": true,
  "branch": "my-feature",
  "mr_url": "https://github.com/org/repo/compare/my-feature?body=..."
}
```

### `POST /request-changes`

Mark session as changes requested.

**Request:**

```json
{
  "message": "See comments on auth module"
}
```

**Response:** Updated session object

***

## Events

### `GET /events`

Server-Sent Events stream. Emits session events in real time.\
\
Used by `review watch` and the TUI.

```text
data: {"event":"comment_added","comment_id":"uuid","timestamp":"..."}

data: {"event":"changes_requested","timestamp":"..."}

data: {"event":"approved","timestamp":"..."}
```

**Event types:**

- `session_opened`
- `comment_added`
- `comment_resolved`
- `comment_deleted`
- `description_updated`
- `changes_requested`
- `approved`

***

## Health

### `GET /health`

Liveness check.

**Response:**

```json
{ "ok": true, "version": "0.1.0” }
```

### `GET /status`

Readiness check.

**Response:**

```json
{ "ok": true, "checks”: [...] }
```
