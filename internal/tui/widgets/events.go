package widgets

import tea "github.com/charmbracelet/bubbletea"

const (
	menuTargetDiffLine        = "diff-line"
	menuTargetVisualSelection = "visual-selection"
)

func ApplyEvent(props Props, event string) Props {
	props = Normalize(props)
	switch event {
	case "page-down":
		w := NewDiff(props)
		w.Viewport.PageDown()
		props.YOffset = w.Viewport.YOffset
	case "page-up":
		w := NewDiff(props)
		w.Viewport.PageUp()
		props.YOffset = w.Viewport.YOffset
	case "half-page-down":
		w := NewDiff(props)
		w.Viewport.HalfPageDown()
		props.YOffset = w.Viewport.YOffset
	case "half-page-up":
		w := NewDiff(props)
		w.Viewport.HalfPageUp()
		props.YOffset = w.Viewport.YOffset
	case "scroll-right":
		props.XOffset += 12
	case "scroll-left":
		if props.XOffset > 12 {
			props.XOffset -= 12
		} else {
			props.XOffset = 0
		}
	default:
		model := NewModel("workspace", props)
		next, _ := model.Update(KeyMsgFor(event))
		if demo, ok := next.(Model); ok {
			return demo.Props()
		}
	}

	return Normalize(props)
}

func UpdateProps(props Props, msg tea.Msg) Props {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return props
	}

	switch key.String() {
	case "esc":
		props.VisualStart = 0
		props.VisualEnd = 0
		props.MenuTarget = menuTargetDiffLine
	case "c":
		props.VisualStart = 0
		props.VisualEnd = 0
		props.MenuTarget = menuTargetDiffLine
		props.SelectedLine = 83
	case "v":
		props.VisualStart = props.SelectedLine
		props.VisualEnd = props.SelectedLine
		props.MenuTarget = menuTargetVisualSelection
	case "j", "down":
		switch props.MenuTarget {
		case "comment", menuTargetVisualSelection, "file":
			props.MenuIndex++
		default:
			if props.VisualStart > 0 {
				props.VisualEnd++
			} else {
				props.SelectedLine++
			}
		}
	case "k", "up":
		switch {
		case props.MenuIndex > 0:
			props.MenuIndex--
		case props.VisualEnd > props.VisualStart:
			props.VisualEnd--
		case props.SelectedLine > 1:
			props.SelectedLine--
		}
	case "/":
		props.Query = "usage"
		props.SelectedLine = 249
	case "f":
		props.Query = "tui"
	case "g":
		props.ActiveFile = "docs/tui-review.md"
		props.Query = "doc"
	case "pgdown", " ":
		w := NewDiff(props)
		w.Viewport.PageDown()
		props.YOffset = w.Viewport.YOffset
	case "pgup", "b":
		w := NewDiff(props)
		w.Viewport.PageUp()
		props.YOffset = w.Viewport.YOffset
	case "d", "ctrl+d":
		w := NewDiff(props)
		w.Viewport.HalfPageDown()
		props.YOffset = w.Viewport.YOffset
	case "u", "ctrl+u":
		w := NewDiff(props)
		w.Viewport.HalfPageUp()
		props.YOffset = w.Viewport.YOffset
	case "h", "left":
		if props.XOffset > 12 {
			props.XOffset -= 12
		} else {
			props.XOffset = 0
		}
	case "l", "right":
		props.XOffset += 12
	}

	return Normalize(props)
}

func KeyMsgFor(event string) tea.KeyMsg {
	switch event {
	case "clear-selection":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "comment-line":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}
	case "comment-range":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")}
	case "menu-down":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	case "menu-up":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	case "search-root":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	case "search-usage":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	case "goto-picker":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")}
	case "goto-doc":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(event)}
	}
}
