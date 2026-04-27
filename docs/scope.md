---
type: Note
belongs_to: "[[local-git-reviews]]"
_organized: true
---
# Scope For Local Git Reviews

## What It Is

A lightweight CLI tool and local daemon for reviewing AI-generated code before pushing to a remote. Run it against any local git repo, browse commits and diffs, leave inline comments, generate MR descriptions with AI, then push when satisfied.

## Problem It Solves

AI agents commit code locally. The natural review moment is before pushing — not after creating a remote MR. This tool inserts a human (and optionally another AI agent) review step at that moment, entirely on local infrastructure.

## MVP Scope

### In

- Open a review session against any local git repo
- Compare current branch against a configurable base (default: `main`)
- View commit list for the branch
- View full diff and per-file diffs
- Add, list, and resolve inline comments (file + line)
- Generate an MR description via Claude API from the diff
- Approve (push to origin) or reject (stay local, flag for rework)
- CLI with `--json` flag on every command for agent/script consumption
- `review watch` command that streams events as newline-delimited JSON
- Local daemon serving JSON over HTTP on localhost
- State persisted in `.git/review/`

### Out (MVP)

- No staging remote / git push interception
- No GUI (Tauri deferred to post-MVP)
- No multi-repo sessions
- No CI integration
- No authentication
- No team/shared review state (local only)
- No support for comparing arbitrary refs beyond branch vs base
- No notifications / system tray

***

## User Types

**Human reviewer** — uses TUI or CLI to browse diffs, leave comments, approve

**AI review agent** — calls CLI with `--json`, reads comments, watches for events,\
pushes fixes back as new commits

**AI coding agent** — commits code locally, optionally calls `review open` to register\
a session, polls `review status --json` to know when approved

***

## Core Workflow

```text
1. AI agent commits code to a local branch
2. Human runs: review open
3. Daemon starts, reads repo state
4. Human browses diff, leaves comments
5. Optionally: review describe (AI generates MR description)
6. Optionally: AI review agent reads comments, pushes fixes
7. Human runs: review approve
8. review approve pushes branch to origin and opens browser
   with pre-filled MR/PR description
```

***

## Non-Goals (Explicit)

- This is not a git client. No staging, no commit creation, no branch management.
- This is not a CI system.
- This is not a replacement for GitHub/GitLab — it is a pre-flight gate before using them.
- This does not manage agent orchestration. It provides primitives agents can use.

***

## Success Criteria for MVP

- `review open` works on any valid local git repo
- Comments survive daemon restart (persisted to disk)
- `--json` output is stable enough for agents to parse reliably
- `review approve` pushes and opens browser MR URL with description pre-filled
- TUI shows diff and comments without external dependencies
