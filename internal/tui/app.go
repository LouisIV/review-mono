package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"review/internal/client"
	"review/internal/config"
	"review/internal/lsp"
	"review/internal/models"
	"review/internal/tui/widgets"
)

type Options struct {
	RepoPath string
	Session  models.Session
	Client   client.Client
	Config   config.Config
}

func Run(opts Options) error {
	m := newModel(opts)
	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if fm, ok := final.(*model); ok && fm.lspManager != nil {
		fm.lspManager.Close()
	}

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
	modeHover
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

	lspManager *lsp.Manager
	hoverInfo  string
}

func newModel(opts Options) *model {
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

	return &model{
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

func (m *model) Init() tea.Cmd {
	return loadReview(m.opts)
}

type loadedMsg struct {
	session  models.Session
	commits  []models.Commit
	files    []models.DiffFile
	comments []models.Comment
}

type errMsg struct{ err error }

type fileLoadedMsg struct {
	file models.DiffFile
}

type commentsLoadedMsg struct {
	comments []models.Comment
	status   string
}

type sessionMsg struct {
	session models.Session
	status  string
}

type descriptionMsg struct {
	body string
}

type hoverMsg struct {
	text string
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case hoverMsg:
		m.hoverInfo = msg.text

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
	default:
	}

	return m, cmd
}

func (m *model) View() string {
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

func mustComments(opts Options) []models.Comment {
	comments, _ := opts.Client.Comments(opts.RepoPath, opts.Session, nil)

	return comments
}

func loadHover(opts Options, mgr *lsp.Manager, file string, line int) tea.Cmd {
	return func() tea.Msg {
		text, err := mgr.Hover(opts.RepoPath, file, line)
		if err != nil {
			return errMsg{err}
		}
		if strings.TrimSpace(text) == "" {
			text = "No hover info available for this location."
		}

		return hoverMsg{text: text}
	}
}
