package widgets

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Workspace struct {
	props       Props
	fileList    FileList
	diff        Diff
	contextMenu ContextMenu
}

func NewWorkspace(props Props) Workspace {
	leftWidth := 32
	rightWidth := props.Width - leftWidth - 3
	if rightWidth < 40 {
		rightWidth = 40
	}

	fileProps := props
	fileProps.Width = leftWidth

	diffProps := props
	diffProps.Width = rightWidth
	diffProps.Height = props.Height - 10

	menuProps := props
	menuProps.Width = props.Width

	return Workspace{
		props:       props,
		fileList:    NewFileList(fileProps),
		diff:        NewDiff(diffProps),
		contextMenu: NewContextMenu(menuProps),
	}
}

func (w Workspace) Init() tea.Cmd { return nil }

func (w Workspace) Update(tea.Msg) (tea.Model, tea.Cmd) { return w, nil }

func (w Workspace) View() string {
	left := w.fileList.View()
	right := w.diff.View()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	header := titleStyle.Render("feature/tui-widgets -> main") + mutedStyle.Render("  in_review  3 commits  5 files  4 open comments")
	help := mutedStyle.Render("Space context menu   f goto file   / search   v visual   c comment   a approve")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, help, w.contextMenu.View()) + "\n"
}
