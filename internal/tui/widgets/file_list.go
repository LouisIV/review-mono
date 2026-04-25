package widgets

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FileList struct {
	props Props
}

func NewFileList(props Props) FileList {
	return FileList{props: props}
}

func (w FileList) Init() tea.Cmd { return nil }

func (w FileList) Update(tea.Msg) (tea.Model, tea.Cmd) { return w, nil }

func (w FileList) View() string {
	rows := []string{
		titleStyle.Render("Files"),
		mutedStyle.Render("in_review  5 changed  4 open"),
	}

	for _, file := range files {
		prefix := "  "
		if file.Path == w.props.ActiveFile {
			prefix = "> "
		}

		state := " "
		if file.Unresolved > 0 {
			state = commentStyle.Render("!")
		} else if file.Viewed {
			state = mutedStyle.Render("v")
		}

		meta := mutedStyle.Render(fmt.Sprintf("+%d -%d", file.Additions, file.Deletions))
		if file.Unresolved > 0 {
			meta += " " + commentStyle.Render(fmt.Sprintf("%d open", file.Unresolved))
		}

		row := fmt.Sprintf("%s%s %-24s %s", prefix, state, truncateMiddle(file.Path, 24), meta)
		if file.Path == w.props.ActiveFile {
			row = activeStyle.Render(row)
		}

		rows = append(rows, row)
	}

	return borderStyle.Width(w.props.Width).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
