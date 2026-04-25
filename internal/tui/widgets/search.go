package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Search struct {
	props Props
	input textinput.Model
	diff  Diff
}

func NewSearch(props Props) Search {
	input := textinput.New()
	input.Prompt = "/"
	input.SetValue(props.Query)
	input.Focus()

	diffProps := props
	diffProps.Width = props.Width - 4
	diffProps.Height = props.Height - 6

	return Search{props: props, input: input, diff: NewDiff(diffProps)}
}

func (w Search) Init() tea.Cmd { return nil }

func (w Search) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	w.input, cmd = w.input.Update(msg)
	w.props.Query = w.input.Value()

	diffProps := w.props
	diffProps.Width = w.props.Width - 4
	diffProps.Height = w.props.Height - 6
	w.diff = NewDiff(diffProps)

	return w, cmd
}

func (w Search) View() string {
	matches := 0
	for _, row := range diffRows {
		if strings.Contains(strings.ToLower(row.Content), strings.ToLower(w.props.Query)) {
			matches++
		}
	}

	status := mutedStyle.Render(fmt.Sprintf("%d matches in %s  Enter next  n/N cycle  Esc close", matches, w.props.ActiveFile))
	content := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render("Search"), w.input.View(), status, w.diff.View())

	return borderStyle.Width(w.props.Width).Render(content)
}
