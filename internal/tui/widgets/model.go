package widgets

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	name        string
	props       Props
	widget      tea.Model
	interactive bool
}

func NewModel(name string, props Props) Model {
	return NewModelWithMode(name, props, false)
}

func NewInteractiveModel(name string, props Props) Model {
	return NewModelWithMode(name, props, true)
}

func NewModelWithMode(name string, props Props, interactive bool) Model {
	props = Normalize(props)
	var widget tea.Model

	switch name {
	case "workspace":
		widget = NewWorkspace(props)
	case "file-list":
		widget = NewFileList(props)
	case "diff":
		widget = NewDiff(props)
	case "context-menu":
		widget = NewContextMenu(props)
	case "goto-file":
		widget = NewFilePicker(props)
	case "search":
		widget = NewSearch(props)
	case "comments":
		widget = NewComments(props)
	default:
		widget = unknownWidget{name: name}
	}

	return Model{name: name, props: props, widget: widget, interactive: interactive}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	next, cmd := m.widget.Update(msg)
	m.widget = next

	m.props = UpdateProps(m.props, msg)

	// Snapshot mode rebuilds after prop updates so the headless harness stays
	// deterministic. Interactive mode keeps child widget state, such as viewport
	// scroll offsets and focused inputs.
	if _, ok := msg.(tea.KeyMsg); ok && !m.interactive {
		m = NewModel(m.name, m.props)
	}

	return m, cmd
}

func (m Model) View() string {
	return m.widget.View()
}

func (m Model) Props() Props {
	return m.props
}

func (m Model) Unknown() bool {
	_, ok := m.widget.(unknownWidget)
	return ok
}

type unknownWidget struct {
	name string
}

func (u unknownWidget) Init() tea.Cmd { return nil }

func (u unknownWidget) Update(tea.Msg) (tea.Model, tea.Cmd) { return u, nil }

func (u unknownWidget) View() string { return "unknown widget: " + u.name + "\n" }

func RenderTree(name string) string {
	trees := map[string][]string{
		"workspace": {
			"WorkspaceWidget [tea.Model]",
			"  Header [lipgloss.Style]",
			"  FileListWidget [tea.Model]",
			"  DiffWidget [tea.Model + bubbles/viewport]",
			"  ContextMenuWidget [tea.Model]",
		},
		"file-list":    {"FileListWidget [tea.Model]", "  FileRow * n [lipgloss.Style]"},
		"diff":         {"DiffWidget [tea.Model]", "  viewport.Model", "  DiffLine * n [lipgloss.Style]"},
		"context-menu": {"ContextMenuWidget [tea.Model]", "  TargetResolver", "  MenuAction * n [lipgloss.Style]"},
		"goto-file":    {"FilePickerWidget [tea.Model]", "  textinput.Model", "  FuzzyResult * n [lipgloss.Style]"},
		"search":       {"SearchWidget [tea.Model]", "  textinput.Model", "  DiffWidget"},
		"comments":     {"CommentsWidget [tea.Model]", "  CommentThread * n [lipgloss.Style]"},
	}

	lines, ok := trees[name]
	if !ok {
		return "unknown widget\n"
	}

	return strings.Join(lines, "\n") + "\n"
}

func ValidateName(name string) error {
	for _, candidate := range Names() {
		if name == candidate {
			return nil
		}
	}

	return fmt.Errorf("unknown widget %q", name)
}
