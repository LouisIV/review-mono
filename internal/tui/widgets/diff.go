package widgets

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Diff struct {
	props    Props
	Viewport viewport.Model
}

func NewDiff(props Props) Diff {
	height := props.Height
	if height <= 0 {
		height = 18
	}

	vp := viewport.New(props.Width-2, height)
	vp.SetHorizontalStep(12)
	w := Diff{props: props, Viewport: vp}
	w.Viewport.SetContent(w.content())
	w.Viewport.SetYOffset(props.YOffset)
	w.Viewport.SetXOffset(props.XOffset)

	return w
}

func (w Diff) Init() tea.Cmd { return nil }

func (w Diff) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "g":
			w.Viewport.GotoTop()
			return w, nil
		case "G":
			w.Viewport.GotoBottom()
			return w, nil
		}
	}

	var cmd tea.Cmd
	w.Viewport, cmd = w.Viewport.Update(msg)

	return w, cmd
}

func (w Diff) View() string {
	title := titleStyle.Render(w.props.ActiveFile)
	return borderStyle.Width(w.props.Width).Render(lipgloss.JoinVertical(lipgloss.Left, title, w.Viewport.View()))
}

func (w Diff) content() string {
	lines := []string{mutedStyle.Render("@ visual range comments use selected line bounds")}
	for _, row := range diffRows {
		if row.Kind == "hunk" {
			lines = append(lines, hunkStyle.Render("  "+row.Content))
			continue
		}

		marker := " "
		if row.Line == w.props.SelectedLine {
			marker = ">"
		}

		if w.props.VisualStart > 0 && row.Line >= w.props.VisualStart && row.Line <= w.props.VisualEnd {
			marker = "|"
		}

		sign := " "
		style := lipgloss.NewStyle()
		switch row.Kind {
		case "add":
			sign = "+"
			style = addStyle
		case "remove":
			sign = "-"
			style = removeStyle
		}

		content := highlight(row.Content, w.props.Query)
		line := fmt.Sprintf("%s %4d %s %s", marker, row.Line, sign, content)
		if row.Line == w.props.SelectedLine {
			line = activeStyle.Render(line)
		} else {
			line = style.Render(line)
		}

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
