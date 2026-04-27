# Review TUI

## Goal

Build a keyboard-first terminal UI for reviewing a local branch before it is
pushed. The TUI is focused entirely on human review: browse changed files,
inspect diffs, leave inline comments, resolve feedback, generate or edit the MR
description, then approve or request changes.

The TUI should not become a general git client. It sits on top of the local
daemon API and uses the same review session, diff, comment, description, and
action primitives as the CLI.

## Primary User Flow

```text
review tui
  -> ensure daemon is running
  -> find or open the current repo's review session
  -> load session summary, commits, diff index, and comments
  -> reviewer navigates files and hunks
  -> reviewer leaves comments or resolves existing comments
  -> reviewer approves or requests changes
```

The first screen should be the review workspace, not a launcher. If there is no
active session for the current repo, the TUI can offer to open one using the
configured default base branch.

## Layout

Use a stable three-region layout:

```text
+----------------------+-----------------------------------------------+
| Review / Files       | Diff                                          |
|                      |                                               |
| status summary       | file header                                   |
| file list            | hunks with line numbers                       |
| comments badges      | inline comment markers                        |
|                      |                                               |
+----------------------+-----------------------------------------------+
| Context / Comments / Command Bar                                      |
+-----------------------------------------------------------------------+
```

### Left Pane

The left pane is an index for review progress.

- Session summary: branch, base, status, commit count, changed file count
- File list with additions, deletions, and unresolved comment count
- Optional filters: all files, commented files, unresolved only
- Visual state per file: unseen, viewed, has unresolved comments, resolved

### Diff Pane

The diff pane is the primary workspace.

- Show one file at a time
- Preserve git diff structure: file header, hunks, context/add/remove lines
- Keep line numbers visible for added and context lines
- Anchor comments to file and line
- Support enough horizontal scrolling to inspect long code lines
- Highlight the currently selected diff line

### Bottom Pane

The bottom pane changes mode based on reviewer intent.

- Default: comments for the selected file or selected line
- Comment composer: multiline markdown input
- Description preview/editor
- Command prompt for rare actions
- Status messages for saves, errors, and daemon reconnects

## Navigation Model

The TUI should be usable without a mouse.

| Key       | Action                                              |
| --------- | --------------------------------------------------- |
| `j` / `k` | Move line selection                                 |
| `J` / `K` | Next / previous hunk                                |
| `]` / `[` | Next / previous file                                |
| `Tab`     | Switch focus between panes                          |
| `Space`   | Open context menu for current target                |
| `f`       | Go to file                                          |
| `v`       | Start visual line selection                         |
| `V`       | Start visual block selection                        |
| `Esc`     | Clear visual selection or cancel current mode       |
| `c`       | Add comment on selected line or selected range      |
| `r`       | Resolve selected comment                            |
| `u`       | Toggle unresolved-only view                         |
| `d`       | Show/edit MR description                            |
| `g`       | Generate MR description                             |
| `a`       | Approve review                                      |
| `x`       | Request changes                                     |
| `/`       | Search current diff                                 |
| `n` / `N` | Next / previous search match while search is active |
| `?`       | Show key help                                       |
| `q`       | Quit                                                |

Destructive or terminal actions should require confirmation:

- Approve, because it pushes the branch
- Request changes, because it changes session state
- Delete comment, if supported in the TUI

## Context Menu

The context menu should be keyboard-first and target-aware. It is a command menu
for the thing under the cursor, not a mouse-only right-click surface.

- `Space` opens the context menu
- `j` / `k` move through actions
- `Enter` runs the selected action
- `Esc` closes the menu
- Direct hotkeys still work for common actions like comment, resolve, search,
  approve, and request changes

The menu should appear in the bottom pane so the diff viewport does not jump or
get covered by a floating box.

Target-specific actions:

```text
Diff line
  Add comment
  Start visual selection
  Copy file:line
  Open in editor

Visual selection
  Comment on range
  Clear selection
  Copy range
  Open range in editor

Comment
  Resolve
  Edit
  Delete
  Copy comment ID

File
  Go to file
  Add file comment
  Copy path
  Open in editor
```

The active target should be resolved in this order:

1. Visual selection, if active
2. Comment under the cursor, if present
3. Selected diff line
4. Selected file

For MVP, the menu can omit edit/delete actions if the underlying comment API is
not ready. The important part is that the model is extensible and discoverable.

## Go To File

`f` opens a fuzzy file picker over the changed files in the current review. This
is the fast path for jumping around larger diffs without using the left pane.

- The picker opens in the bottom pane
- Typing filters by path using fuzzy matching
- `Enter` jumps to the selected file and closes the picker
- `Esc` closes the picker without changing files
- `j` / `k` or arrow keys move through filtered results
- Results should show path, additions/deletions, and unresolved comment count
- The current file should be marked in the result list

The file picker is scoped to review files, not the whole repository. A broader
repo-wide file opener would turn the TUI toward a general editor, which is out of
scope.

## Visual Selection

Line selection should feel closer to Vim than to a form-based review tool.

- `v` starts visual line selection from the current diff line
- `V` starts visual block selection for contiguous changed or context lines
- Moving with `j`, `k`, hunk jumps, or search extends the active selection
- `Esc` clears the selection
- `c` comments on the selected range
- The selected range should be visibly highlighted in the diff pane
- The bottom pane should show the selected file and line range before composing

For MVP, visual block selection can be represented as a contiguous line range in
one file. True rectangular column selection is not needed for comments.

Comment ranges should prefer the documented `lines: [start, end]` shape. If the
daemon only supports a single line when the TUI is implemented, comment on the
start line and include the selected range in the rendered body until the API is
updated.

## Search

Search should behave like a pager or Vim inside the current diff.

- `/` opens an inline search prompt in the bottom pane
- Typing updates the query without leaving the diff context
- `Enter` jumps to the next match
- `n` and `N` move to next and previous matches after a search exists
- Matches should be highlighted in the diff pane
- The current match should be visually distinct from other matches
- Search should include code content, hunk headers, and file paths
- Search starts within the current file; crossing files can be added after the
  per-file flow is solid

Search should not require an external pager. The TUI owns the viewport,
highlighting, and selected-line state so comments can be added immediately from a
match.

## Review States

The TUI should make progress obvious without requiring a separate checklist.

- `in_review`: normal editable state
- `changes_requested`: comments can still be browsed and resolved, but the main
  call to action is waiting for updates or reopening review
- `approved`: read-only summary, with the MR/PR URL if available

Per-file state can be derived locally:

- Viewed: reviewer has selected the file during this TUI session
- Has unresolved comments: any comment for the file where `resolved=false`
- Complete: viewed and no unresolved comments

Viewed state can start in-memory. Persisting it can wait until there is a clear
need.

## Comment UX

Inline comments are the core interaction.

- `c` opens a composer bound to the selected file and line or visual range
- Saving posts `POST /session/:id/comments`
- Comment bodies are markdown text, but the editor can be plain multiline text
- Existing comments appear in the bottom pane when the selected line has comments
- File-level comments can be added from the file list or when no diff line is
  selected
- Resolving a comment patches `resolved=true`

For the MVP, comment editing and deletion can stay in the CLI unless the API and
implementation are already cheap.

## Description UX

The MR description is a secondary but important review artifact.

- `d` opens the saved description, if present
- `g` generates a description from the current diff
- Streaming generation should update the bottom pane incrementally
- The reviewer can edit the generated result before approval
- Saving posts `POST /session/:id/description`

Description editing does not need to compete with a full editor. A textarea-like
terminal input is enough for MVP; `$EDITOR` integration can remain available via
the CLI.

## Data Loading

Initial load:

1. `GET /sessions` filtered by repo path and current branch, or open a session
2. `GET /session/:id`
3. `GET /session/:id/commits`
4. `GET /session/:id/comments`
5. `GET /session/:id/diff?skip_hunks=true`
6. Lazy load `GET /session/:id/diff?file=...` when a file is selected

Live updates:

- Subscribe to `GET /events`
- On comment events, refresh comments
- On description events, refresh description metadata
- On session status events, refresh the session

The TUI should be resilient to daemon restarts. If the event stream disconnects,
show a reconnecting state and retry with backoff while keeping the already-loaded
review visible.

## Implementation Shape

Recommended command:

```text
review tui [repo] [--base main] [--port 7080]
```

Recommended package boundary:

```text
cmd/tui.go
internal/tui/
  app.go          # top-level Bubble Tea program
  state.go        # loaded session, files, comments, selected positions
  keys.go         # key map and help
  views.go        # layout rendering
  diff_view.go    # diff rendering and navigation
  comments.go     # comment list/composer model
  description.go  # description viewer/editor/generator model
```

Recommended dependencies:

- `github.com/charmbracelet/bubbletea` for the app runtime
- `github.com/charmbracelet/lipgloss` for layout and styling
- `github.com/charmbracelet/bubbles` for viewport, textarea, spinner, and help

These fit a Go CLI and avoid introducing a separate frontend runtime.

## API Gaps To Check

The existing API is close, but the TUI will pressure a few details:

- Diff pagination is mentioned but not fully specified. Large diffs need either
  reliable file-level lazy loading or hunk/page parameters.
- Comments support `lines` in the documented request table, but examples and data
  models use a single `line`. Pick one shape before building range comments.
- Events are currently global. The TUI should be able to filter by session, or
  event payloads should always include `session_id`.
- `POST /approve` and `POST /request-changes` are documented without session IDs.
  If multiple sessions exist, these should be session-scoped.
- Description generation streams newline-delimited JSON, while events use SSE.
  The TUI can handle both, but the client package should expose typed helpers.

## MVP

The first usable TUI should include:

- Open/current session resolution
- File list with changed counts and unresolved comment badges
- Lazy-loaded per-file diff viewer
- Keyboard navigation by file, hunk, and line
- Context menu for target-aware actions
- Fuzzy goto-file picker scoped to changed files
- Visual line/range selection for comments
- Pager/Vim-style search within the current file
- Add inline or range comment
- List and resolve comments
- Generate and show description
- Approve with confirmation
- Request changes with confirmation and message input

Explicitly defer:

- Mouse support
- Comment edit/delete
- Persistent viewed-file tracking
- True rectangular column selection
- Side-by-side diffs
- Syntax highlighting
- Multi-session dashboard
- CI/checks integration

## Open Questions

- Should `review tui` automatically run `review open` when no session exists, or
  should it ask first?
- Is approval allowed while unresolved comments exist, or should that require an
  explicit override?
- Should request-changes require at least one unresolved comment or message?
- Should viewed-file state eventually be persisted in `.git/reviews/`, or remain
  local to each TUI run?
- Is the TUI the long-term primary human UI, or a bridge until a GUI exists?
