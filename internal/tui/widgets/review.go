package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type FileItem struct {
	Path       string
	Additions  int
	Deletions  int
	Unresolved int
	Viewed     bool
}

type DiffItem struct {
	Kind    string
	Hunk    int
	Line    int
	Content string
}

type CommentItem struct {
	ID       string
	File     string
	Line     int
	EndLine  int
	Body     string
	Resolved bool
}

type WorkspaceData struct {
	Branch       string
	Base         string
	Status       string
	CommitCount  int
	Files        []FileItem
	Rows         []DiffItem
	Comments     []CommentItem
	ActiveFile   string
	SelectedLine int
	VisualStart  int
	VisualEnd    int
	Query        string
	Top          int
	XOffset      int
	Focus        string
	BottomTitle  string
	BottomBody   string
	BottomHeight int
	Context      []string
	ContextIndex int
	ShowResolved bool
}

func RenderReviewWorkspace(width, height int, data WorkspaceData) string {
	if width <= 0 {
		width = 96
	}
	if height <= 0 {
		height = 28
	}

	leftWidth := 34
	if width < 90 {
		leftWidth = 28
	}
	rightWidth := width - leftWidth - 1
	if rightWidth < 30 {
		rightWidth = 30
	}

	bottomHeight := 5
	if height > 32 {
		bottomHeight = 7
	}
	if data.BottomHeight > bottomHeight {
		bottomHeight = data.BottomHeight
	}

	// Three bordered regions plus the one-line help footer consume eight rows:
	// header border/content, body border, bottom border, and help.
	bodyHeight := height - bottomHeight - 8
	if bodyHeight < 8 {
		bodyHeight = 8
	}

	open, resolved := commentCounts(data.Comments)
	header := titleStyle.Render(data.Branch+" -> "+data.Base) +
		mutedStyle.Render(fmt.Sprintf("  %s  focus:%s  %d commits  %d files  %d open  %d resolved", data.Status, data.Focus, data.CommitCount, len(data.Files), open, resolved))
	headerBox := fixedBox(width, 1, []string{header})
	left := renderRuntimeFiles(leftWidth, bodyHeight, data)
	right := renderRuntimeDiff(rightWidth, bodyHeight, data)
	bottom := renderRuntimeBottom(width, bottomHeight, data)
	help := mutedStyle.Render("Tab focus  Files:j/k choose  Diff:j/k line J/K hunk  [/]/file  f goto  / search  v visual  c comment  r resolve  d desc  g gen  a approve  x request  ? help  q quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerBox,
		lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right),
		bottom,
		help,
	)
}

func renderRuntimeFiles(width, height int, data WorkspaceData) string {
	rows := []string{
		titleStyle.Render("Files"),
		focusLabel(data.Focus == "files", data.Status),
	}
	for _, file := range data.Files {
		prefix := "  "
		if file.Path == data.ActiveFile {
			prefix = "> "
		}

		state := " "
		if file.Unresolved > 0 {
			state = commentStyle.Render("!")
		} else if file.Viewed {
			state = mutedStyle.Render("v")
		}

		pathWidth := width - 18
		if pathWidth < 10 {
			pathWidth = 10
		}
		meta := mutedStyle.Render(fmt.Sprintf("+%d -%d", file.Additions, file.Deletions))
		if file.Unresolved > 0 {
			meta += " " + commentStyle.Render(fmt.Sprintf("%d", file.Unresolved))
		}

		row := fmt.Sprintf("%s%s %-*s %s", prefix, state, pathWidth, truncateMiddle(file.Path, pathWidth), meta)
		if file.Path == data.ActiveFile {
			row = activeStyle.Render(row)
		}
		rows = append(rows, row)
	}

	return fixedBox(width, height, rows)
}

func renderRuntimeDiff(width, height int, data WorkspaceData) string {
	title := titleStyle.Render(data.ActiveFile)
	if data.ActiveFile == "" {
		title = titleStyle.Render("Diff")
	}
	if data.Focus == "diff" {
		title = activeStyle.Render(" " + lipgloss.NewStyle().Bold(true).Render(stripANSI(data.ActiveFile, "Diff")) + " ")
	}

	rows := []string{title, mutedStyle.Render(strings.Repeat("-", max(0, width-4)))}
	visible := data.Rows
	if data.Top > 0 && data.Top < len(visible) {
		visible = visible[data.Top:]
	}
	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}
	if len(visible) > maxRows {
		visible = visible[:maxRows]
	}

	for _, row := range visible {
		rows = append(rows, renderRuntimeDiffLine(width-4, row, data))
	}
	if len(data.Rows) == 0 {
		rows = append(rows, mutedStyle.Render("  no diff loaded"))
	}

	return fixedBox(width, height, rows)
}

func focusLabel(active bool, text string) string {
	if active {
		return activeStyle.Render(" active ") + " " + mutedStyle.Render(text)
	}

	return mutedStyle.Render(text)
}

func stripANSI(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func renderRuntimeDiffLine(width int, row DiffItem, data WorkspaceData) string {
	if row.Kind == "hunk" {
		return hunkStyle.Render("      " + truncateMiddle(row.Content, width-6))
	}

	marker := " "
	if row.Line == data.SelectedLine {
		marker = ">"
	}
	if data.VisualStart > 0 {
		start, end := ordered(data.VisualStart, data.VisualEnd)
		if row.Line >= start && row.Line <= end {
			marker = "|"
		}
	}

	line := "    "
	if row.Line > 0 {
		line = fmt.Sprintf("%4d", row.Line)
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

	content := row.Content
	if data.XOffset > 0 && data.XOffset < len(content) {
		content = content[data.XOffset:]
	} else if data.XOffset >= len(content) {
		content = ""
	}
	content = highlight(strings.ReplaceAll(content, "\t", "    "), data.Query)
	badge := ""
	for _, comment := range data.Comments {
		if comment.File == data.ActiveFile && !comment.Resolved && comment.Line == row.Line {
			badge = " " + commentStyle.Render("@"+comment.ID)
			break
		}
	}

	rendered := fmt.Sprintf("%s %s  %s %s%s", marker, mutedStyle.Render(line), style.Render(sign), style.Render(truncateMiddle(content, width-12)), badge)
	if row.Line == data.SelectedLine {
		return selectedLineStyle().Render(rendered)
	}

	return rendered
}

func renderRuntimeBottom(width, height int, data WorkspaceData) string {
	if len(data.Context) > 0 {
		rows := []string{titleStyle.Render("Context")}
		for i, action := range data.Context {
			row := "  " + action
			if i == data.ContextIndex {
				row = activeStyle.Render("> " + action)
			}
			rows = append(rows, row)
		}

		return fixedBox(width, height, rows)
	}

	title := data.BottomTitle
	if title == "" {
		title = "Comments"
	}
	rows := []string{titleStyle.Render(title)}
	if data.BottomBody != "" {
		rows = append(rows, splitLines(data.BottomBody)...)
	} else {
		for _, comment := range data.Comments {
			if comment.File != data.ActiveFile {
				continue
			}
			if comment.Resolved && !data.ShowResolved {
				continue
			}
			state := commentStyle.Render("open")
			if comment.Resolved {
				state = mutedStyle.Render("resolved")
			}
			loc := fmt.Sprintf("%s:%d", comment.File, comment.Line)
			if comment.EndLine > comment.Line {
				loc = fmt.Sprintf("%s:%d-%d", comment.File, comment.Line, comment.EndLine)
			}
			rows = append(rows, fmt.Sprintf("%s  %s  %s", comment.ID, state, loc))
			rows = append(rows, mutedStyle.Render("  "+comment.Body))
		}
		if len(rows) == 1 {
			rows = append(rows, mutedStyle.Render("No comments for selected file"))
		}
	}

	return fixedBox(width, height, rows)
}

func RenderFilePicker(width int, query string, files []FileItem, cursor int) string {
	rows := []string{titleStyle.Render("Go to file"), "/" + query}
	matches := FilterFileItems(query, files)
	for i, file := range matches {
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}
		meta := mutedStyle.Render(fmt.Sprintf("+%d -%d %d open", file.Additions, file.Deletions, file.Unresolved))
		row := fmt.Sprintf("%s%-42s %s", prefix, truncateMiddle(file.Path, 42), meta)
		if i == cursor {
			row = activeStyle.Render(row)
		}
		rows = append(rows, row)
	}
	if len(matches) == 0 {
		rows = append(rows, mutedStyle.Render("  no changed files match"))
	}

	return fixedBox(width, 8, rows)
}

func FilterFileItems(query string, files []FileItem) []FileItem {
	query = strings.ToLower(strings.TrimSpace(query))
	out := []FileItem{}
	for _, file := range files {
		if query == "" || fuzzyContains(strings.ToLower(file.Path), query) {
			out = append(out, file)
		}
	}

	return out
}

func ActionsForTarget(target string) []string {
	return actionsFor(target)
}

func fixedBox(width, height int, rows []string) string {
	if height <= 0 {
		height = len(rows)
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	for len(rows) < height {
		rows = append(rows, "")
	}

	return borderStyle.Width(width).Height(height).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}

	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func commentCounts(comments []CommentItem) (int, int) {
	open, resolved := 0, 0
	for _, comment := range comments {
		if comment.Resolved {
			resolved++
		} else {
			open++
		}
	}

	return open, resolved
}

func ordered(a, b int) (int, int) {
	if a > b {
		return b, a
	}

	return a, b
}
