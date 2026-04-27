package widgets

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const keyDown = "down"

type ContextMenu struct {
	props Props
}

func NewContextMenu(props Props) ContextMenu {
	return ContextMenu{props: props}
}

func (w ContextMenu) Init() tea.Cmd { return nil }

func (w ContextMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return w, nil
	}

	switch key.String() {
	case "j", keyDown:
		w.props.MenuIndex++
	case "k", "up":
		if w.props.MenuIndex > 0 {
			w.props.MenuIndex--
		}
	}

	return w, nil
}

func (w ContextMenu) View() string {
	actions := actionsFor(w.props.MenuTarget)
	if len(actions) == 0 {
		actions = actionsFor("diff-line")
	}

	idx := w.props.MenuIndex % len(actions)
	rows := make([]string, 0, 1+len(actions))
	rows = append(rows, titleStyle.Render("Context: "+w.props.MenuTarget))
	for i, action := range actions {
		row := "  " + action
		if i == idx {
			row = activeStyle.Render("> " + action)
		}

		rows = append(rows, row)
	}

	return borderStyle.Width(w.props.Width).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
