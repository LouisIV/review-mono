package widgets

import (
	"fmt"
	"strings"

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

	vp := viewport.New(props.Width, height)
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
	meta := mutedStyle.Render("  2 hunks  +18 -3")
	rule := mutedStyle.Render(strings.Repeat("-", max(0, w.props.Width-2)))

	return lipgloss.JoinVertical(lipgloss.Left, title+meta, rule, w.Viewport.View())
}

func (w Diff) content() string {
	lines := []string{}
	for _, row := range diffRows {
		if row.Kind == "hunk" {
			lines = append(lines, w.renderHunk(row.Content))
			continue
		}

		lines = append(lines, w.renderLine(row))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (w Diff) renderHunk(header string) string {
	return hunkStyle.Render("          " + compactHunk(header))
}

func (w Diff) renderLine(row diffRow) string {
	bg := lineBackground(row.Kind, row.Line == w.props.SelectedLine)

	marker := " "
	if row.Line == w.props.SelectedLine {
		marker = ">"
	}

	if w.props.VisualStart > 0 && row.Line >= w.props.VisualStart && row.Line <= w.props.VisualEnd {
		marker = "|"
	}

	line := lineNumBlank
	if row.Line > 0 {
		line = fmt.Sprintf("%4d", row.Line)
	}

	sign := " "
	signStyle := lipgloss.NewStyle()

	switch row.Kind {
	case "add":
		sign = "▌"
		signStyle = addStyle
	case "remove":
		sign = "▌"
		signStyle = removeStyle
	}

	raw := strings.ReplaceAll(row.Content, "\t", lineNumBlank)
	content := renderSyntaxLine(w.props.ActiveFile, raw, w.props.Query, bg)
	comment := w.commentBadge(row.Line, bg)
	gutter := styleWithLineBg(mutedStyle, bg).Render(fmt.Sprintf("%s %s", marker, line))
	rendered := gutter +
		lineBgStyle(bg).Render("  ") +
		styleWithLineBg(signStyle, bg).Render(sign) +
		lineBgStyle(bg).Render(" ") +
		content +
		comment

	return padLineBackground(rendered, w.props.Width, bg)
}

func (w Diff) commentBadge(line int, bg lipgloss.Color) string {
	for _, comment := range comments {
		if comment.File == w.props.ActiveFile && comment.Line == line && !comment.Resolved {
			return lineBgStyle(bg).Render(" ") + styleWithLineBg(commentStyle, bg).Render("@ "+comment.ID)
		}
	}

	return ""
}

func compactHunk(header string) string {
	header = strings.TrimPrefix(header, "@@ ")
	header = strings.TrimSuffix(header, " @@")
	return "@@ " + header
}

func lineBackground(kind string, selected bool) lipgloss.Color {
	if selected {
		return lipgloss.Color("236")
	}

	switch kind {
	case "add":
		return addBg
	case "remove":
		return removeBg
	default:
		return ""
	}
}
