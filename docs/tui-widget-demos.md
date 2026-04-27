# TUI Widget Demos

The widget demo command is a fast, headless playground for the review TUI. It is
inspired by the feedback-loopable widget CLI idea: render small UI pieces from
props, inspect their terminal output, or inspect their widget tree without
launching the full TUI.

These demos use the same libraries the real TUI is expected to use:

- Bubble Tea for `tea.Model` widgets
- Bubbles for reusable inputs and viewports
- Lip Gloss for terminal layout and styling

The command is a harness around real widget models, not a separate ASCII mockup
system.

By default, `review widget` prints a deterministic snapshot and exits. That path
is meant for feedback-loopable iteration and should stay stable. Use
`--interactive` or `-I` when you want to run the selected widget as a Bubble Tea
program and try key handling directly. Press `q` or `ctrl+c` to exit.

Interactive diff movement uses the Bubbles viewport pager bindings:

- `j` / `k`: line down/up
- arrow down/up: line down/up
- `f` / `space` / `pgdown`: page down
- `b` / `pgup`: page up
- `d` / `ctrl+d`: half page down
- `u` / `ctrl+u`: half page up
- `h` / `l` or arrow left/right: horizontal scroll
- `g` / `G`: top/bottom

The context menu and file picker also support arrow up/down for moving the
active item.

```bash
review widget list
review widget workspace --width 100 --height 28
review widget diff --props '{"query":"widget","visual_start":83,"visual_end":84}'
review widget context-menu --props '{"menu_target":"comment","menu_index":1}'
review widget goto-file --event goto-picker
review widget search --event search-usage
review widget workspace --tree
review widget diff --interactive
review widget context-menu -I --props '{"menu_target":"comment"}'
```

## Widgets

- `workspace`: main review layout with file list, diff viewport, and bottom pane
- `file-list`: changed-file index with comment badges
- `diff`: diff viewport with selected line, search highlights, and visual range
- `context-menu`: target-aware actions for line, range, comment, or file
- `goto-file`: fuzzy picker scoped to changed files
- `search`: pager/Vim-style search prompt plus highlighted diff
- `comments`: comment panel with open and optionally resolved comments

## Props

Props are JSON and can be combined with events.

```json
{
  "width": 96,
  "height": 28,
  "query": "widget",
  "active_file": "cmd/root.go",
  "selected_line": 83,
  "visual_start": 83,
  "visual_end": 84,
  "y_offset": 0,
  "x_offset": 0,
  "show_resolved": false,
  "menu_target": "visual-selection",
  "menu_index": 0
}
```

## Events

Events mutate props before rendering. They are intentionally simple so edge
cases are easy to reproduce in a shell command.

- `comment-line`
- `comment-range`
- `clear-selection`
- `menu-down`
- `menu-up`
- `search-root`
- `search-usage`
- `goto-picker`
- `goto-doc`
- `page-down`
- `page-up`
- `half-page-down`
- `half-page-up`
- `scroll-right`
- `scroll-left`

The diff and search widgets use Bubbles `viewport.Model` as the pager. The demo
events above drive the same pager operations the real TUI can use: page movement,
half-page movement, and horizontal scrolling.
