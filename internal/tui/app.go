package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"review/internal/client"
	"review/internal/models"
	"review/internal/tui/widgets"
)

type Options struct {
	RepoPath string
	Session  models.Session
	Client   client.Client
}

func Run(opts Options) error {
	m := newModel(opts)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()

	return err
}

type mode int

const (
	modeReview mode = iota
	modeHelp
	modeGoto
	modeSearch
	modeComment
	modeDescription
	modeRequestChanges
	modeConfirmApprove
	modeConfirmRequest
	modeContext
)

type focus int

const (
	focusFiles focus = iota
	focusDiff
	focusBottom
)

type diffRow struct {
	kind    string
	hunk    int
	line    int
	content string
}

type model struct {
	opts     Options
	width    int
	height   int
	mode     mode
	focus    focus
	loading  bool
	status   string
	err      error
	session  models.Session
	commits  []models.Commit
	files    []models.DiffFile
	comments []models.Comment

	fileIndex int
	rows      []diffRow
	lineIndex int
	top       int
	xOffset   int
	viewed    map[string]bool

	unresolvedOnly bool
	visualStart    int
	visualEnd      int

	gotoInput      textinput.Model
	gotoIndex      int
	search         textinput.Model
	query          string
	matches        []int
	match          int
	composer       textarea.Model
	commentActions bool
	commentAction  int
	request        textarea.Model

	description string
	context     []string
	contextIdx  int
}

func newModel(opts Options) model {
	gotoInput := textinput.New()
	gotoInput.Placeholder = "changed file"
	search := textinput.New()
	search.Placeholder = "search diff"
	composer := textarea.New()
	composer.Placeholder = "Write a markdown comment"
	composer.ShowLineNumbers = false
	request := textarea.New()
	request.Placeholder = "Request changes message"
	request.ShowLineNumbers = false

	return model{
		opts:      opts,
		session:   opts.Session,
		loading:   true,
		status:    "loading review",
		focus:     focusDiff,
		viewed:    map[string]bool{},
		gotoInput: gotoInput,
		search:    search,
		composer:  composer,
		request:   request,
	}
}

func (m model) Init() tea.Cmd {
	return loadReview(m.opts)
}

type loadedMsg struct {
	session  models.Session
	commits  []models.Commit
	files    []models.DiffFile
	comments []models.Comment
}

type errMsg struct{ err error }

func loadReview(opts Options) tea.Cmd {
	return func() tea.Msg {
		session, err := opts.Client.Session(opts.RepoPath, opts.Session)
		if err != nil {
			return errMsg{err}
		}

		commits, err := opts.Client.Commits(opts.RepoPath, session)
		if err != nil {
			return errMsg{err}
		}

		files, _, err := opts.Client.Diff(opts.RepoPath, session, "", "", true)
		if err != nil {
			return errMsg{err}
		}

		comments, err := opts.Client.Comments(opts.RepoPath, session, nil)
		if err != nil {
			return errMsg{err}
		}

		return loadedMsg{session: session, commits: commits, files: files, comments: comments}
	}
}

type fileLoadedMsg struct {
	file models.DiffFile
}

func loadFile(opts Options, path string) tea.Cmd {
	return func() tea.Msg {
		files, _, err := opts.Client.Diff(opts.RepoPath, opts.Session, path, "", false)
		if err != nil {
			return errMsg{err}
		}
		if len(files) == 0 {
			return errMsg{fmt.Errorf("no diff for %s", path)}
		}

		return fileLoadedMsg{file: files[0]}
	}
}

type commentsLoadedMsg struct {
	comments []models.Comment
	status   string
}

func refreshComments(opts Options, status string) tea.Cmd {
	return func() tea.Msg {
		comments, err := opts.Client.Comments(opts.RepoPath, opts.Session, nil)
		if err != nil {
			return errMsg{err}
		}

		return commentsLoadedMsg{comments: comments, status: status}
	}
}

type sessionMsg struct {
	session models.Session
	status  string
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInputs()
		return m, nil
	case loadedMsg:
		m.loading = false
		m.err = nil
		m.session = msg.session
		m.opts.Session = msg.session
		m.commits = msg.commits
		m.files = msg.files
		m.comments = msg.comments
		if len(m.files) > 0 {
			m.viewed[m.files[0].Path] = true
			return m, loadFile(m.opts, m.files[0].Path)
		}
		m.status = "no changed files"
		return m, nil
	case fileLoadedMsg:
		m.replaceFile(msg.file)
		m.rows = flatten(msg.file)
		m.lineIndex = firstSelectable(m.rows)
		m.top = 0
		m.xOffset = 0
		m.status = "loaded " + msg.file.Path
		m.updateMatches()
		return m, nil
	case commentsLoadedMsg:
		m.comments = msg.comments
		m.status = msg.status
		return m, nil
	case sessionMsg:
		m.session = msg.session
		m.opts.Session = msg.session
		m.status = msg.status
		m.mode = modeReview
		return m, nil
	case descriptionMsg:
		m.description = msg.body
		if strings.TrimSpace(m.description) == "" {
			m.description = "No description saved."
		}
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
		m.status = msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	switch m.mode {
	case modeGoto:
		m.gotoInput, cmd = m.gotoInput.Update(msg)
		m.clampGoto()
	case modeSearch:
		m.search, cmd = m.search.Update(msg)
		m.query = m.search.Value()
		m.updateMatches()
	case modeComment:
		m.composer, cmd = m.composer.Update(msg)
	case modeRequestChanges:
		m.request, cmd = m.request.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	if m.loading {
		return "Loading review...\n"
	}
	if m.err != nil && len(m.files) == 0 {
		return "error: " + m.err.Error() + "\n"
	}

	data := widgets.WorkspaceData{
		Branch:       m.session.Branch,
		Base:         m.session.Base,
		Status:       m.session.Status,
		CommitCount:  len(m.commits),
		Files:        m.widgetFiles(),
		Rows:         m.widgetRows(),
		Comments:     m.widgetComments(),
		ActiveFile:   m.currentFile(),
		SelectedLine: m.currentLine(),
		VisualStart:  m.visualStart,
		VisualEnd:    m.visualEnd,
		Query:        m.query,
		Top:          m.top,
		XOffset:      m.xOffset,
		Focus:        m.focusName(),
		BottomTitle:  m.bottomTitle(),
		BottomBody:   m.bottomBody(),
		BottomHeight: m.bottomHeight(),
		ShowResolved: true,
	}
	if m.mode == modeContext {
		data.Context = m.context
		data.ContextIndex = m.contextIdx
	}

	return widgets.RenderReviewWorkspace(m.width, m.height, data) + "\n"
}

func (m *model) resizeInputs() {
	w := m.width - 8
	if w < 20 {
		w = 20
	}
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
	rows := []diffRow{}
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

func (m model) diffHeight() int {
	h := m.height - 12
	if h < 5 {
		h = 5
	}

	return h
}

func (m model) selectFile(index int) (tea.Model, tea.Cmd) {
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

func (m model) currentFile() string {
	if len(m.files) == 0 || m.fileIndex >= len(m.files) {
		return ""
	}

	return m.files[m.fileIndex].Path
}

func (m model) currentLine() int {
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

func (m model) filteredFiles() []widgets.FileItem {
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

func (m model) currentComment() models.Comment {
	file := m.currentFile()
	line := m.currentLine()
	for _, comment := range m.comments {
		if comment.File == file && !comment.Resolved && comment.Line == line {
			return comment
		}
	}

	return models.Comment{}
}

func (m *model) openContext() {
	target := "diff-line"
	if m.visualStart > 0 {
		target = "visual-selection"
	} else if m.currentComment().ID != "" {
		target = "comment"
	} else if m.focus == focusFiles {
		target = "file"
	}
	m.context = widgets.ActionsForTarget(target)
	m.contextIdx = 0
	m.mode = modeContext
}

func (m model) widgetFiles() []widgets.FileItem {
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

func (m model) widgetRows() []widgets.DiffItem {
	out := make([]widgets.DiffItem, 0, len(m.rows))
	for _, row := range m.rows {
		out = append(out, widgets.DiffItem{Kind: row.kind, Hunk: row.hunk, Line: row.line, Content: row.content})
	}

	return out
}

func (m model) widgetComments() []widgets.CommentItem {
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

func (m model) focusName() string {
	switch m.focus {
	case focusFiles:
		return "files"
	case focusBottom:
		return "bottom"
	default:
		return "diff"
	}
}

func (m model) bottomTitle() string {
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
	default:
		return "Comments"
	}
}

func (m model) bottomHeight() int {
	switch m.mode {
	case modeComment:
		return 11
	case modeRequestChanges:
		return 8
	case modeDescription, modeHelp:
		return 9
	default:
		return 0
	}
}

func (m model) bottomBody() string {
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
		return "Approve pushes the current branch and marks this review approved.\nPress y to approve or n/Esc to cancel."
	case modeConfirmRequest:
		return "Mark this review as changes requested?\nPress y to request changes or n/Esc to cancel."
	case modeHelp:
		return "j/k move lines, J/K move hunks, [/]/next/previous file, f file picker, / search, n/N search results, v visual selection, c comment, r resolve, d show description, g generate description, a approve, x request changes, Space context menu."
	default:
		target := m.commentTarget()
		if target != ":" && target != "" {
			body := "Selected: " + target + "\nPress c to add a comment"
			if m.status != "" {
				body += "\n" + mutedStatus(m.status)
			}
			return body
		}
	}

	return ""
}

func (m model) commentTarget() string {
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

func mutedStatus(s string) string {
	return s
}

func (m model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeGoto:
		return m.handleGotoKey(key)
	case modeSearch:
		return m.handleSearchKey(key)
	case modeComment:
		return m.handleCommentKey(key)
	case modeRequestChanges:
		return m.handleRequestKey(key)
	case modeConfirmApprove, modeConfirmRequest:
		return m.handleConfirmKey(key)
	case modeContext:
		return m.handleContextKey(key)
	case modeHelp, modeDescription:
		if key.String() == "esc" || key.String() == "q" || key.String() == "?" {
			m.mode = modeReview
		}
		return m, nil
	}

	switch key.String() {
	case "q":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "tab":
		m.focus = (m.focus + 1) % 3
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
	case "j", "down":
		if m.focus == focusFiles {
			return m.selectFile(m.fileIndex + 1)
		}
		m.moveLine(1)
	case "k", "up":
		if m.focus == focusFiles {
			return m.selectFile(m.fileIndex - 1)
		}
		m.moveLine(-1)
	case "enter":
		if m.focus == focusFiles {
			return m.selectFile(m.fileIndex)
		}
	case "J":
		m.moveHunk(1)
	case "K":
		m.moveHunk(-1)
	case "]", "right":
		return m.selectFile(m.fileIndex + 1)
	case "[", "left":
		return m.selectFile(m.fileIndex - 1)
	case "pgdown":
		m.moveLine(m.diffHeight())
	case "pgup":
		m.moveLine(-m.diffHeight())
	case "h":
		if m.xOffset > 0 {
			m.xOffset -= 8
			if m.xOffset < 0 {
				m.xOffset = 0
			}
		}
	case "l":
		m.xOffset += 8
	case "f":
		m.mode = modeGoto
		m.gotoInput.SetValue("")
		m.gotoInput.Focus()
		m.gotoIndex = 0
	case "/":
		m.mode = modeSearch
		m.search.SetValue(m.query)
		m.search.Focus()
	case "n":
		m.nextMatch(1)
	case "N":
		m.nextMatch(-1)
	case "u":
		m.unresolvedOnly = !m.unresolvedOnly
	case "v", "V":
		line := m.currentLine()
		if line > 0 {
			m.visualStart = line
			m.visualEnd = line
			m.status = "visual selection started"
		}
	case "esc":
		m.visualStart = 0
		m.visualEnd = 0
	case "c":
		if m.currentFile() != "" {
			m.mode = modeComment
			m.composer.SetValue("")
			m.composer.Focus()
			m.commentActions = false
			m.commentAction = 0
			m.status = "comment composer"
		}
	case "r":
		return m.resolveCurrent()
	case "d":
		return m.showDescription(false)
	case "g":
		return m.showDescription(true)
	case "a":
		m.mode = modeConfirmApprove
	case "x":
		m.mode = modeRequestChanges
		m.request.SetValue("")
		m.request.Focus()
	case " ":
		m.openContext()
	}

	return m, nil
}

func (m model) handleGotoKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeReview
	case "enter":
		matches := m.filteredFiles()
		if len(matches) > 0 {
			for i, file := range m.files {
				if file.Path == matches[m.gotoIndex].Path {
					m.mode = modeReview
					return m.selectFile(i)
				}
			}
		}
	case "j", "down":
		m.gotoIndex++
		m.clampGoto()
	case "k", "up":
		m.gotoIndex--
		m.clampGoto()
	default:
		var cmd tea.Cmd
		m.gotoInput, cmd = m.gotoInput.Update(key)
		m.gotoIndex = 0
		return m, cmd
	}

	return m, nil
}

func (m model) handleSearchKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeReview
	case "enter":
		m.mode = modeReview
		m.nextMatch(1)
	default:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(key)
		m.query = m.search.Value()
		m.updateMatches()
		return m, cmd
	}

	return m, nil
}

func (m model) handleCommentKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commentActions {
		switch key.String() {
		case "esc":
			m.mode = modeReview
			m.commentActions = false
		case "left", "h", "right", "l", "tab":
			if m.commentAction == 0 {
				m.commentAction = 1
			} else {
				m.commentAction = 0
			}
		case "enter":
			if m.commentAction == 1 {
				m.mode = modeReview
				m.commentActions = false
				return m, nil
			}

			return m.submitComment()
		}

		return m, nil
	}

	switch key.String() {
	case "esc":
		m.mode = modeReview
	case "down":
		m.commentActions = true
		m.commentAction = 0
	case "ctrl+s":
		return m.submitComment()
	default:
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(key)
		return m, cmd
	}

	return m, nil
}

func (m model) submitComment() (tea.Model, tea.Cmd) {
	body := strings.TrimSpace(m.composer.Value())
	if body == "" {
		m.status = "comment body is empty"
		m.commentActions = false
		return m, nil
	}
	comment := models.Comment{File: m.currentFile(), Line: m.currentLine(), Body: body, Author: models.AuthorHuman}
	if m.focus == focusFiles {
		comment.Line = 0
	}
	if m.visualStart > 0 {
		start, end := ordered(m.visualStart, m.visualEnd)
		comment.Line = start
		comment.Lines = []int{start, end}
	}
	m.mode = modeReview
	m.commentActions = false
	m.visualStart = 0
	m.visualEnd = 0

	return m, addComment(m.opts, comment)
}

func (m model) commentActionRow() string {
	submit := "[ Submit ]"
	cancel := "[ Cancel ]"
	if m.commentActions && m.commentAction == 0 {
		submit = "> Submit <"
	}
	if m.commentActions && m.commentAction == 1 {
		cancel = "> Cancel <"
	}

	return submit + "  " + cancel + "\nEnter newline  Down actions  Ctrl+S save  Esc close"
}

func addComment(opts Options, comment models.Comment) tea.Cmd {
	return func() tea.Msg {
		saved, err := opts.Client.AddComment(opts.RepoPath, opts.Session, comment)
		if err != nil {
			return errMsg{err}
		}
		_ = saved

		return commentsLoadedMsg{comments: mustComments(opts), status: "comment added"}
	}
}

func (m model) handleRequestKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeReview
	case "ctrl+s":
		m.mode = modeConfirmRequest
	default:
		var cmd tea.Cmd
		m.request, cmd = m.request.Update(key)
		return m, cmd
	}

	return m, nil
}

func (m model) handleConfirmKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "n":
		m.mode = modeReview
	case "y":
		if m.mode == modeConfirmApprove {
			return m, approve(m.opts)
		}
		return m, requestChanges(m.opts, strings.TrimSpace(m.request.Value()))
	}

	return m, nil
}

func (m model) handleContextKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.mode = modeReview
	case "j", "down":
		if m.contextIdx < len(m.context)-1 {
			m.contextIdx++
		}
	case "k", "up":
		if m.contextIdx > 0 {
			m.contextIdx--
		}
	case "enter":
		if len(m.context) == 0 {
			m.mode = modeReview
			return m, nil
		}
		action := m.context[m.contextIdx]
		m.mode = modeReview
		switch action {
		case "Add comment", "Comment on selection":
			m.mode = modeComment
			m.composer.SetValue("")
			m.composer.Focus()
			m.commentActions = false
			m.commentAction = 0
		case "Resolve comment":
			return m.resolveCurrent()
		case "Go to file":
			m.mode = modeGoto
			m.gotoInput.SetValue("")
			m.gotoInput.Focus()
		case "Clear selection":
			m.visualStart = 0
			m.visualEnd = 0
		}
	}

	return m, nil
}

func approve(opts Options) tea.Cmd {
	return func() tea.Msg {
		_, err := opts.Client.Approve(opts.RepoPath, false)
		if err != nil {
			return errMsg{err}
		}
		session, err := opts.Client.Session(opts.RepoPath, opts.Session)
		if err != nil {
			return errMsg{err}
		}

		return sessionMsg{session: session, status: "review approved"}
	}
}

func requestChanges(opts Options, message string) tea.Cmd {
	return func() tea.Msg {
		session, err := opts.Client.RequestChanges(opts.RepoPath, message)
		if err != nil {
			return errMsg{err}
		}

		return sessionMsg{session: session, status: "changes requested"}
	}
}

func (m model) resolveCurrent() (tea.Model, tea.Cmd) {
	comment := m.currentComment()
	if comment.ID == "" {
		m.status = "no unresolved comment on selected line"
		return m, nil
	}

	return m, func() tea.Msg {
		_, err := m.opts.Client.PatchComment(m.opts.RepoPath, m.opts.Session, comment.ID, map[string]bool{"resolved": true})
		if err != nil {
			return errMsg{err}
		}

		return commentsLoadedMsg{comments: mustComments(m.opts), status: "comment resolved"}
	}
}

func (m model) showDescription(generate bool) (tea.Model, tea.Cmd) {
	m.mode = modeDescription
	m.description = "loading description..."

	return m, func() tea.Msg {
		var (
			desc models.Description
			err  error
		)
		if generate {
			desc, err = m.opts.Client.GenerateDescription(m.opts.RepoPath, m.opts.Session, "", "")
		} else {
			desc, err = m.opts.Client.Description(m.opts.RepoPath, m.opts.Session)
		}
		if err != nil {
			return errMsg{err}
		}

		return descriptionMsg{body: desc.Body}
	}
}

type descriptionMsg struct {
	body string
}

func mustComments(opts Options) []models.Comment {
	comments, _ := opts.Client.Comments(opts.RepoPath, opts.Session, nil)

	return comments
}
