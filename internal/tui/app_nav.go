package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"review/internal/lsp"
	"review/internal/models"
	"review/internal/tui/widgets"
)

func (m *model) resizeInputs() {
	w := m.width - 8
	w = max(w, 20)
	m.gotoInput.Width = w
	m.search.Width = w
	m.composer.SetWidth(w)
	m.composer.SetHeight(5)
	m.request.SetWidth(w)
	m.request.SetHeight(4)
}

func (m *model) replaceFile(file models.DiffFile) {
	for i := range m.files {
		if m.files[i].Path == file.Path {
			m.files[i].Hunks = file.Hunks

			return
		}
	}
	m.files = append(m.files, file)
}

func flatten(file models.DiffFile) []diffRow {
	rows := make([]diffRow, 0, len(file.Hunks))
	for hunkIndex, hunk := range file.Hunks {
		rows = append(rows, diffRow{kind: "hunk", hunk: hunkIndex, content: hunk.Header})
		for _, line := range hunk.Lines {
			n := 0
			if line.Number != nil {
				n = *line.Number
			}
			rows = append(rows, diffRow{kind: line.Type, hunk: hunkIndex, line: n, content: line.Content})
		}
	}

	return rows
}

func toWidgetRows(rows []diffRow) []widgets.DiffItem {
	out := make([]widgets.DiffItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, widgets.DiffItem{Kind: row.kind, Hunk: row.hunk, Line: row.line, Content: row.content})
	}

	return out
}

func firstSelectable(rows []diffRow) int {
	for i, row := range rows {
		if row.line > 0 {
			return i
		}
	}

	return 0
}

func (m *model) moveLine(delta int) {
	if len(m.rows) == 0 {
		return
	}
	next := m.lineIndex
	for {
		next += delta
		if next < 0 || next >= len(m.rows) {
			break
		}
		if m.rows[next].line > 0 {
			m.lineIndex = next
			if m.visualStart > 0 {
				m.visualEnd = m.rows[next].line
			}

			break
		}
	}
	m.ensureVisible()
}

func (m *model) moveHunk(delta int) {
	if len(m.rows) == 0 {
		return
	}
	current := m.rows[m.lineIndex].hunk
	for i, row := range m.rows {
		if delta > 0 && row.hunk > current && row.line > 0 {
			m.lineIndex = i
			m.ensureVisible()

			return
		}
	}
	if delta < 0 {
		for i := len(m.rows) - 1; i >= 0; i-- {
			if m.rows[i].hunk < current && m.rows[i].line > 0 {
				m.lineIndex = i
				m.ensureVisible()

				return
			}
		}
	}
}

func (m *model) ensureVisible() {
	h := m.diffHeight()
	if m.lineIndex < m.top {
		m.top = m.lineIndex
	}
	if m.lineIndex >= m.top+h {
		m.top = m.lineIndex - h + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

func (m *model) diffHeight() int {
	return max(m.height-12, 5)
}

func (m *model) selectFile(index int) (tea.Model, tea.Cmd) {
	if len(m.files) == 0 {
		return m, nil
	}
	if index < 0 {
		index = len(m.files) - 1
	}
	if index >= len(m.files) {
		index = 0
	}
	m.fileIndex = index
	m.viewed[m.files[index].Path] = true
	m.status = "loading " + m.files[index].Path

	return m, loadFile(m.opts, m.files[index].Path)
}

func (m *model) currentFile() string {
	if len(m.files) == 0 || m.fileIndex >= len(m.files) {
		return ""
	}

	return m.files[m.fileIndex].Path
}

func (m *model) currentLine() int {
	if len(m.rows) == 0 || m.lineIndex >= len(m.rows) {
		return 0
	}

	return m.rows[m.lineIndex].line
}

func (m *model) updateMatches() {
	m.matches = nil
	query := strings.ToLower(strings.TrimSpace(m.query))
	if query == "" {
		return
	}
	for i, row := range m.rows {
		if strings.Contains(strings.ToLower(row.content), query) {
			m.matches = append(m.matches, i)
		}
	}
}

func (m *model) nextMatch(delta int) {
	if len(m.matches) == 0 {
		m.status = "no search matches"

		return
	}
	m.match += delta
	if m.match < 0 {
		m.match = len(m.matches) - 1
	}
	if m.match >= len(m.matches) {
		m.match = 0
	}
	m.lineIndex = m.matches[m.match]
	if m.rows[m.lineIndex].line == 0 {
		m.moveLine(1)
	} else {
		m.ensureVisible()
	}
}

func (m *model) filteredFiles() []widgets.FileItem {
	return widgets.FilterFileItems(m.gotoInput.Value(), m.widgetFiles())
}

func (m *model) clampGoto() {
	matches := m.filteredFiles()
	if len(matches) == 0 {
		m.gotoIndex = 0

		return
	}
	if m.gotoIndex < 0 {
		m.gotoIndex = len(matches) - 1
	}
	if m.gotoIndex >= len(matches) {
		m.gotoIndex = 0
	}
}

func (m *model) currentComment() models.Comment {
	file := m.currentFile()
	line := m.currentLine()
	for _, comment := range m.comments {
		if comment.File == file && !comment.Resolved && comment.Line == line {
			return comment
		}
	}

	return models.Comment{}
}

func (m *model) requestHover() (tea.Model, tea.Cmd) {
	file := m.currentFile()
	line := m.currentLine()

	if file == "" || line == 0 {
		m.status = "select a diff line to inspect"

		return m, nil
	}

	if m.lspManager == nil {
		m.lspManager = lsp.NewManager(m.opts.RepoPath, m.opts.Config.LSP.Servers)
	}

	m.mode = modeHover
	m.hoverInfo = "fetching hover info…"

	return m, loadHover(m.opts, m.lspManager, file, line)
}

func (m *model) openContext() {
	var target string
	switch {
	case m.visualStart > 0:
		target = "visual-selection"
	case m.currentComment().ID != "":
		target = "comment"
	case m.focus == focusFiles:
		target = "file"
	default:
		target = "diff-line"
	}
	m.context = widgets.ActionsForTarget(target)
	m.contextIdx = 0
	m.mode = modeContext
}

func (m *model) widgetFiles() []widgets.FileItem {
	unresolved := map[string]int{}
	for _, comment := range m.comments {
		if !comment.Resolved {
			unresolved[comment.File]++
		}
	}
	out := make([]widgets.FileItem, 0, len(m.files))
	for _, file := range m.files {
		out = append(out, widgets.FileItem{
			Path:       file.Path,
			Additions:  file.Additions,
			Deletions:  file.Deletions,
			Unresolved: unresolved[file.Path],
			Viewed:     m.viewed[file.Path],
		})
	}

	return out
}

func (m *model) widgetComments() []widgets.CommentItem {
	out := make([]widgets.CommentItem, 0, len(m.comments))
	for _, comment := range m.comments {
		end := comment.Line
		if len(comment.Lines) == 2 {
			end = comment.Lines[1]
		}
		out = append(out, widgets.CommentItem{
			ID:       comment.ID,
			File:     comment.File,
			Line:     comment.Line,
			EndLine:  end,
			Body:     comment.Body,
			Resolved: comment.Resolved,
		})
	}

	return out
}

func (m *model) focusName() string {
	switch m.focus {
	case focusFiles:
		return "files"
	case focusDiff:
		return "diff"
	case focusBottom:
		return "bottom"
	default:
		return "diff"
	}
}

func (m *model) bottomTitle() string {
	switch m.mode {
	case modeGoto:
		return "Go to file"
	case modeSearch:
		return "Search"
	case modeComment:
		return "Comment"
	case modeDescription:
		return "Description"
	case modeRequestChanges:
		return "Request changes"
	case modeConfirmApprove, modeConfirmRequest:
		return "Confirm"
	case modeHelp:
		return "Help"
	case modeHover:
		return "Hover Info"
	default:
		return "Comments"
	}
}

func (m *model) bottomHeight() int {
	switch m.mode {
	case modeComment:
		return 11
	case modeRequestChanges:
		return 8
	case modeDescription, modeHelp, modeHover:
		return 9
	default:
		return 0
	}
}

func (m *model) bottomBody() string {
	switch m.mode {
	case modeGoto:
		return widgets.RenderFilePicker(m.width-4, m.gotoInput.Value(), m.widgetFiles(), m.gotoIndex)
	case modeSearch:
		return "/" + m.search.Value() + fmt.Sprintf("\n%d matches  Enter next  Esc close", len(m.matches))
	case modeComment:
		return "Comment target: " + m.commentTarget() + "\n" + m.composer.View() + "\n" + m.commentActionRow()
	case modeDescription:
		return m.description
	case modeRequestChanges:
		return m.request.View() + "\nctrl+s continue  esc cancel"
	case modeConfirmApprove:
		return "Approve pushes the current branch and marks this review approved.\n" +
			"Press y to approve or n/Esc to cancel."
	case modeConfirmRequest:
		return "Mark this review as changes requested?\nPress y to request changes or n/Esc to cancel."
	case modeHover:
		return m.hoverInfo
	case modeHelp:
		return "j/k move lines, J/K move hunks, [/]/next/previous file, f file picker, " +
			"/ search, n/N search results, v visual selection, c comment, r resolve, " +
			"d show description, g generate description, a approve, x request changes, " +
			"i hover info (LSP), Space context menu."
	default:
		target := m.commentTarget()
		if target != ":" && target != "" {
			body := "Selected: " + target + "\nPress c to add a comment"
			if m.status != "" {
				body += "\n" + m.status
			}

			return body
		}
	}

	return ""
}

func (m *model) commentTarget() string {
	file := m.currentFile()
	if m.focus == focusFiles {
		return file
	}
	if m.visualStart > 0 {
		start, end := ordered(m.visualStart, m.visualEnd)

		return fmt.Sprintf("%s:%d-%d", file, start, end)
	}

	return fmt.Sprintf("%s:%d", file, m.currentLine())
}

func ordered(a, b int) (int, int) {
	if a > b {
		return b, a
	}

	return a, b
}
