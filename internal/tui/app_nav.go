package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"review/internal/lsp"
	"review/internal/models"
	"review/internal/tui/widgets"
)

type refreshPosition struct {
	preserve  bool
	file      string
	fileIndex int
	line      int
	row       int
	top       int
	xOffset   int
}

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
			m.files[i].ContentLines = file.ContentLines

			return
		}
	}
	m.files = append(m.files, file)
}

func flatten(file models.DiffFile, expanded []expandedRange) []diffRow {
	rows := make([]diffRow, 0, len(file.Hunks))
	cursor := 1
	for hunkIndex, hunk := range file.Hunks {
		start, end := hunkBounds(hunk)
		if start > 0 {
			rows = appendExpandedGap(rows, file.ContentLines, cursor, start-1, expanded)
		}
		rows = append(rows, diffRow{
			kind:        "hunk",
			hunk:        hunkIndex,
			content:     hunk.Header,
			uncommitted: hunk.Uncommitted,
		})
		for _, line := range hunk.Lines {
			n := 0
			if line.Number != nil {
				n = *line.Number
			}
			rows = append(rows, diffRow{
				kind:        line.Type,
				hunk:        hunkIndex,
				line:        n,
				content:     line.Content,
				uncommitted: hunk.Uncommitted,
			})
		}
		if end >= cursor {
			cursor = end + 1
		}
	}
	rows = appendExpandedGap(rows, file.ContentLines, cursor, len(file.ContentLines), expanded)

	return rows
}

func toWidgetRows(rows []diffRow, selected int) []widgets.DiffItem {
	out := make([]widgets.DiffItem, 0, len(rows))
	for i, row := range rows {
		out = append(out, widgets.DiffItem{
			Kind:        row.kind,
			Hunk:        row.hunk,
			Line:        row.line,
			Content:     row.content,
			Selected:    i == selected,
			Uncommitted: row.uncommitted,
		})
	}

	return out
}

func firstSelectable(rows []diffRow) int {
	for i, row := range rows {
		if rowSelectable(row) {
			return i
		}
	}

	return 0
}

func (m *model) refreshPosition() refreshPosition {
	return refreshPosition{
		preserve:  true,
		file:      m.currentFile(),
		fileIndex: m.fileIndex,
		line:      m.currentLine(),
		row:       m.lineIndex,
		top:       m.top,
		xOffset:   m.xOffset,
	}
}

func (m *model) fileIndexForPath(path string, fallback int) int {
	return fileIndexForPath(m.files, path, fallback)
}

func fileIndexForPath(files []models.DiffFile, path string, fallback int) int {
	for i, file := range files {
		if file.Path == path {
			return i
		}
	}
	if fallback < 0 {
		if len(files) == 0 {
			return -1
		}

		return 0
	}
	if fallback >= len(files) {
		return len(files) - 1
	}

	return fallback
}

func (m *model) applyReviewData(
	session models.Session,
	commits []models.Commit,
	files []models.DiffFile,
	comments []models.Comment,
	refresh refreshPosition,
) {
	m.session = session
	m.opts.Session = session
	m.commits = commits
	m.files = files
	m.comments = comments
	if len(m.files) > 0 {
		index := 0
		if refresh.preserve {
			index = m.fileIndexForPath(refresh.file, refresh.fileIndex)
		}
		m.fileIndex = index
		m.viewed[m.files[index].Path] = true

		return
	}

	m.rows = nil
	m.diffItems = nil
	m.fileIndex = 0
	m.lineIndex = 0
	m.top = 0
	m.xOffset = 0
	m.status = "no changed files"
}

func (m *model) restoreRefreshPosition(refresh refreshPosition) {
	if len(m.rows) == 0 {
		m.lineIndex = 0
		m.top = 0
		m.xOffset = 0

		return
	}

	if !refresh.preserve {
		m.lineIndex = firstSelectable(m.rows)
		m.top = 0
		m.xOffset = 0

		return
	}

	m.lineIndex = closestRowForLine(m.rows, refresh.line, refresh.row)
	if refresh.top < 0 {
		refresh.top = 0
	}
	if refresh.top >= len(m.rows) {
		refresh.top = len(m.rows) - 1
	}
	m.top = refresh.top
	m.xOffset = max(refresh.xOffset, 0)
	m.ensureVisible()
}

func (m *model) shouldDelayRefresh() bool {
	return m.mode != modeReview
}

func (m *model) applyPendingRefresh(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if m.mode != modeReview || !m.pendingRefresh {
		return m, cmd
	}

	m.pendingRefresh = false
	m.status = "refreshing review..."
	refreshCmd := liveRefresh(m.opts, m.refreshPosition())
	if cmd != nil {
		return m, tea.Batch(cmd, refreshCmd)
	}

	return m, refreshCmd
}

func closestRowForLine(rows []diffRow, line int, fallback int) int {
	if line > 0 {
		bestIndex := -1
		bestDistance := int(^uint(0) >> 1)
		for i, row := range rows {
			if row.line <= 0 {
				continue
			}
			distance := abs(row.line - line)
			if distance < bestDistance {
				bestIndex = i
				bestDistance = distance
			}
			if distance == 0 {
				break
			}
		}
		if bestIndex >= 0 {
			return bestIndex
		}
	}

	if fallback >= 0 && fallback < len(rows) {
		return fallback
	}

	return firstSelectable(rows)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}

	return n
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
		if rowSelectable(m.rows[next]) {
			m.lineIndex = next
			if m.visualStart > 0 && m.rows[next].line > 0 {
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
		if delta > 0 && row.hunk > current && rowSelectable(row) {
			m.lineIndex = i
			m.ensureVisible()

			return
		}
	}
	if delta < 0 {
		for i := len(m.rows) - 1; i >= 0; i-- {
			if m.rows[i].hunk < current && rowSelectable(m.rows[i]) {
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
	bottomHeight := 5
	if m.height > 32 {
		bottomHeight = 7
	}
	if h := m.bottomHeight(); h > bottomHeight {
		bottomHeight = h
	}

	bodyHeight := max(m.height-bottomHeight-8, 8)

	return max(bodyHeight-2, 1)
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

func ordered(a, b int) (int, int) {
	if a > b {
		return b, a
	}

	return a, b
}
