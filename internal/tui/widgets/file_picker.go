package widgets

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FilePicker struct {
	props  Props
	input  textinput.Model
	cursor int
}

func NewFilePicker(props Props) FilePicker {
	input := textinput.New()
	input.Placeholder = "changed file"
	input.SetValue(props.Query)
	input.Focus()

	return FilePicker{props: props, input: input}
}

func (w FilePicker) Init() tea.Cmd { return nil }

func (w FilePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		matches := filteredFiles(w.props.Query)
		switch key.String() {
		case "down":
			if w.cursor < len(matches)-1 {
				w.cursor++
			}

			return w, nil
		case "up":
			if w.cursor > 0 {
				w.cursor--
			}

			return w, nil
		}
	}

	var cmd tea.Cmd
	w.input, cmd = w.input.Update(msg)
	w.props.Query = w.input.Value()
	if w.cursor >= len(filteredFiles(w.props.Query)) {
		w.cursor = 0
	}

	return w, cmd
}

func (w FilePicker) View() string {
	rows := []string{titleStyle.Render("Go to file"), w.input.View()}
	matches := filteredFiles(w.props.Query)

	for i, file := range matches {
		prefix := "  "
		if i == w.cursor {
			prefix = "> "
		}

		meta := mutedStyle.Render(fmt.Sprintf("+%d -%d %d open", file.Additions, file.Deletions, file.Unresolved))
		row := fmt.Sprintf("%s%-42s %s", prefix, truncateMiddle(file.Path, 42), meta)
		if i == w.cursor {
			row = activeStyle.Render(row)
		}

		rows = append(rows, row)
	}

	if len(matches) == 0 {
		rows = append(rows, mutedStyle.Render("  no changed files match"))
	}

	return borderStyle.Width(w.props.Width).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
