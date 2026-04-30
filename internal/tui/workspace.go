package tui

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"review/internal/client"
	"review/internal/config"
	"review/internal/git"
	"review/internal/models"
	"review/internal/repoconfig"
	"review/internal/tui/widgets"
)

// WorkspaceOptions configures a multi-repo workspace TUI session.
type WorkspaceOptions struct {
	Repos  []git.Repo
	Client client.Client
	Config config.Config
	Base   string
}

type workspaceRepoState struct {
	repo    git.Repo
	name    string
	branch  string
	session models.Session
	files   []models.DiffFile
	commits int
	err     error
	loading bool
}

type wsMode int

const (
	wsModeList   wsMode = iota
	wsModeReview
)

type workspaceModel struct {
	opts     WorkspaceOptions
	width    int
	height   int
	repos    []workspaceRepoState
	selected int
	mode     wsMode
	status   string
	review   *model
	program  *tea.Program
}

type workspaceRepoLoadedMsg struct {
	index   int
	branch  string
	session models.Session
	files   []models.DiffFile
	commits int
	err     error
}

// RunWorkspace launches the multi-repo workspace TUI.
func RunWorkspace(opts WorkspaceOptions) error {
	m := newWorkspaceModel(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p
	_, err := p.Run()
	return err
}

func newWorkspaceModel(opts WorkspaceOptions) *workspaceModel {
	repos := make([]workspaceRepoState, len(opts.Repos))
	for i, repo := range opts.Repos {
		repos[i] = workspaceRepoState{
			repo:    repo,
			name:    filepath.Base(repo.Path),
			loading: true,
		}
	}
	return &workspaceModel{
		opts:   opts,
		repos:  repos,
		mode:   wsModeList,
		status: "loading...",
	}
}

func (m *workspaceModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.repos))
	for i := range m.repos {
		cmds[i] = m.loadRepoCmd(i)
	}
	return tea.Batch(cmds...)
}

func (m *workspaceModel) loadRepoCmd(index int) tea.Cmd {
	repo := m.repos[index].repo
	base := m.baseForRepo(repo)
	c := m.opts.Client
	return func() tea.Msg {
		branch, err := repo.Branch()
		if err != nil {
			return workspaceRepoLoadedMsg{index: index, err: err}
		}
		session, err := workspaceOpenSession(repo, c, branch, base)
		if err != nil {
			return workspaceRepoLoadedMsg{index: index, branch: branch, err: err}
		}
		files, _, err := c.Diff(repo.Path, session, "", "", true)
		if err != nil {
			return workspaceRepoLoadedMsg{index: index, branch: branch, session: session, err: err}
		}
		commits, _ := c.Commits(repo.Path, session)
		return workspaceRepoLoadedMsg{
			index:   index,
			branch:  branch,
			session: session,
			files:   files,
			commits: len(commits),
		}
	}
}

func (m *workspaceModel) baseForRepo(repo git.Repo) string {
	repoCfg, err := repoconfig.Load(repo)
	if err == nil && repoCfg.DefaultBranch != "" {
		return repoCfg.DefaultBranch
	}
	return m.opts.Base
}

func workspaceOpenSession(repo git.Repo, c client.Client, branch, base string) (models.Session, error) {
	sessions, err := c.Sessions(repo.Path)
	if err != nil {
		return models.Session{}, err
	}
	for i := len(sessions) - 1; i >= 0; i-- {
		s := sessions[i]
		if s.Repo == repo.Path && s.Branch == branch && s.Status != models.StatusApproved {
			return s, nil
		}
	}
	return c.Open(repo.Path, base)
}

func (m *workspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.mode == wsModeReview && m.review != nil {
			next, cmd := m.review.Update(msg)
			if nm, ok := next.(*model); ok {
				m.review = nm
			}
			return m, cmd
		}
		return m, nil

	case workspaceRepoLoadedMsg:
		if msg.index >= 0 && msg.index < len(m.repos) {
			r := &m.repos[msg.index]
			r.loading = false
			r.branch = msg.branch
			r.session = msg.session
			r.files = msg.files
			r.commits = msg.commits
			r.err = msg.err
		}
		loading := 0
		for _, r := range m.repos {
			if r.loading {
				loading++
			}
		}
		if loading > 0 {
			m.status = fmt.Sprintf("loading (%d remaining)...", loading)
		} else {
			m.status = fmt.Sprintf("%d repos", len(m.repos))
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == wsModeReview && m.review != nil {
			if msg.String() == "esc" {
				return m.exitReview()
			}
			next, cmd := m.review.Update(msg)
			if nm, ok := next.(*model); ok {
				m.review = nm
			}
			return m, cmd
		}
		return m.handleListKey(msg)

	default:
		// Route all other messages (loadedMsg, watcherEventMsg, etc.) to the active review model.
		if m.mode == wsModeReview && m.review != nil {
			next, cmd := m.review.Update(msg)
			if nm, ok := next.(*model); ok {
				m.review = nm
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m *workspaceModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.selected < len(m.repos)-1 {
			m.selected++
		}
	case "k", "up":
		if m.selected > 0 {
			m.selected--
		}
	case "enter", " ":
		return m.enterReview()
	case "R":
		m.status = "reloading..."
		for i := range m.repos {
			m.repos[i].loading = true
		}
		cmds := make([]tea.Cmd, len(m.repos))
		for i := range m.repos {
			cmds[i] = m.loadRepoCmd(i)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m *workspaceModel) enterReview() (tea.Model, tea.Cmd) {
	if m.selected >= len(m.repos) {
		return m, nil
	}
	r := m.repos[m.selected]
	if r.loading || r.err != nil {
		return m, nil
	}
	reviewOpts := Options{
		RepoPath: r.repo.Path,
		Session:  r.session,
		Client:   m.opts.Client,
		Config:   m.opts.Config,
	}
	review := newModel(reviewOpts)
	review.program = m.program
	review.width = m.width
	review.height = m.height
	m.review = review
	m.mode = wsModeReview
	return m, review.Init()
}

func (m *workspaceModel) exitReview() (tea.Model, tea.Cmd) {
	if m.review != nil {
		m.review.stopWatcher()
		m.review = nil
	}
	m.mode = wsModeList
	m.repos[m.selected].loading = true
	return m, m.loadRepoCmd(m.selected)
}

func (m *workspaceModel) View() string {
	if m.mode == wsModeReview && m.review != nil {
		return m.review.View()
	}
	return m.renderList()
}

func (m *workspaceModel) renderList() string {
	items := make([]widgets.WorkspaceRepoItem, len(m.repos))
	for i, r := range m.repos {
		items[i] = widgets.WorkspaceRepoItem{
			Name:        r.name,
			Branch:      r.branch,
			Status:      r.session.Status,
			FileCount:   len(r.files),
			CommitCount: r.commits,
			Loading:     r.loading,
			Err:         r.err,
		}
	}
	return widgets.RenderWorkspaceList(m.width, m.height, items, m.selected, m.status)
}
