# ADR 0001: Widget Visual Diff on Pull Requests

## Status

Accepted

## Context

The project has a `review widget <name>` command that renders TUI widgets headlessly to stdout as ANSI-styled terminal text. There are seven widgets: `workspace`, `file-list`, `diff`, `context-menu`, `goto-file`, `search`, `comments`.

Widget rendering is driven by `lipgloss` styles and `bubbletea` layout logic. Changes to these — whether intentional redesigns or accidental regressions — are invisible in normal code review because the diff shows Go source changes, not visual output. A reviewer cannot tell from a style constant change whether the result looks right without running the widget locally.

## Decision

We run a GitHub Actions workflow on every pull request that:

1. **Detects visual changes** by rendering all widgets with `NO_COLOR=1` on both the PR branch and the base branch, then diffing the plain-text outputs. A widget is considered changed if its text output differs, or if it is new (no corresponding base snapshot exists).

2. **Generates color screenshots** for changed widgets only, using [`charmbracelet/freeze`](https://github.com/charmbracelet/freeze). Freeze runs the widget command in a PTY via `--execute`, which causes `lipgloss`/`termenv` to detect terminal color support and render with full ANSI styling. This produces a PNG that matches what a developer would see in their terminal.

3. **Hosts images on [catbox.moe](https://catbox.moe)** by uploading each PNG with a single `curl -F` call using an authenticated userhash (stored as `CATBOX_USERHASH` in GitHub Actions secrets). The service returns a plain URL which is embedded directly in the PR comment. No storage infrastructure is required in the repository.

4. **Posts a PR comment** with a `| Before | After |` screenshot table for each changed widget. The comment is identified by an HTML marker (`<!-- widget-snapshots -->`) so subsequent pushes to the PR update the same comment rather than creating duplicates. If a later push produces no changes, the comment is deleted.

### Avoiding redundant base builds

The base branch binary and its rendered snapshots are cached in GitHub Actions using `actions/cache`, keyed on the base branch HEAD SHA:

```
key: widget-base-<sha>
```

On a cache hit the base checkout and build are skipped entirely. The base is only rebuilt when the base branch itself advances, which is typically infrequent relative to PR push volume.

### Change detection vs. screenshot rendering

Two separate renders are performed per widget:

- `NO_COLOR=1` — used for change detection. Output is stable and deterministic regardless of terminal environment.
- Full color (via `freeze --execute`) — used for screenshots. Only runs for widgets where the text diff confirmed a change, keeping CI time low.

### Why catbox.moe over alternatives

| Option | Rejected because |
|--------|-----------------|
| Commit PNGs to a `widget-snapshots` branch | Accumulates binary git objects indefinitely; no automatic cleanup |
| GitHub Release assets | Requires creating and managing a release; more setup for a non-release artifact |
| GitHub Actions artifacts | Download URLs require authentication; cannot be embedded as `<img>` in PR comments |
| Inline base64 / data URIs | GitHub strips these from markdown comment rendering |
| 0x0.st | Uploads disabled at time of adoption |
| transfer.sh | Service unreachable at time of adoption |

Catbox.moe accepts a file over HTTP and returns a permanent plain URL. Authenticated uploads (via userhash) are permitted for CI use. The userhash is stored as a GitHub Actions secret (`CATBOX_USERHASH`) and never appears in workflow logs.

The images contain terminal UI screenshots and carry no sensitive information, so public hosting is appropriate.

## Consequences

- PRs that change widget appearance get an automatic before/after screenshot comment with no manual steps.
- PRs that do not touch widget rendering produce no comment (the workflow runs but exits silently).
- The workflow requires `pull-requests: write` permission to post comments. No `contents: write` permission is needed.
- PRs from forks will not receive comments because `pull_request` events from forks run with read-only tokens. This is a GitHub Actions security constraint and not specific to this implementation.
- Old PR comments will have broken images after 0x0.st's retention window expires. This is tolerable since the screenshots are only referenced during the review lifecycle.
- `charmbracelet/freeze` is installed at workflow runtime via `go install`. It is not a declared module dependency.
