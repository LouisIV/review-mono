package widgets

import tea "github.com/charmbracelet/bubbletea"

// WorkspaceList is the demo widget for the workspace repo selection screen.
type WorkspaceList struct {
	props Props
}

func NewWorkspaceList(props Props) WorkspaceList {
	return WorkspaceList{props: props}
}

func (w WorkspaceList) Init() tea.Cmd { return nil }

func (w WorkspaceList) Update(tea.Msg) (tea.Model, tea.Cmd) { return w, nil }

func (w WorkspaceList) View() string {
	return RenderWorkspaceList(w.props.Width, w.props.Height, sampleWorkspaceRepos, 0, "3 repos")
}

var sampleWorkspaceRepos = []WorkspaceRepoItem{
	{
		Name:        "api-gateway",
		Branch:      "feature/auth-refresh",
		Status:      "in_review",
		FileCount:   7,
		CommitCount: 3,
	},
	{
		Name:        "web-client",
		Branch:      "feature/auth-refresh",
		Status:      "in_review",
		FileCount:   12,
		CommitCount: 5,
	},
	{
		Name:        "infra",
		Branch:      "main",
		Status:      "",
		FileCount:   0,
		CommitCount: 0,
	},
}
