package widgets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Comments struct {
	props Props
}

func NewComments(props Props) Comments {
	return Comments{props: props}
}

func (w Comments) Init() tea.Cmd { return nil }

func (w Comments) Update(tea.Msg) (tea.Model, tea.Cmd) { return w, nil }

func (w Comments) View() string {
	rows := []string{titleStyle.Render("Comments")}
	for _, comment := range comments {
		if comment.Resolved && !w.props.ShowResolved {
			continue
		}

		state := commentStyle.Render("open")
		if comment.Resolved {
			state = mutedStyle.Render("resolved")
		}

		location := fmt.Sprintf("%s:%d", comment.File, comment.Line)
		if comment.EndLine > comment.Line {
			location = fmt.Sprintf("%s:%d-%d", comment.File, comment.Line, comment.EndLine)
		}

		rows = append(rows, fmt.Sprintf("%s  %s  %s", comment.ID, state, location))
		rows = append(rows, mutedStyle.Render("  "+comment.Body))
	}

	return borderStyle.Width(w.props.Width).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
