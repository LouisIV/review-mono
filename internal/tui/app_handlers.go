package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"review/internal/models"
)

const (
	keyEsc   = "esc"
	keyDown  = "down"
	keyUp    = "up"
	keyEnter = "enter"
)

var ( //nolint:gochecknoglobals
	commentButtonStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 2)
	commentButtonActiveStyle = commentButtonStyle.Copy().
					Foreground(lipgloss.Color("230")).
					Background(lipgloss.Color("57")).
					BorderForeground(lipgloss.Color("57"))
)

func (m *model) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeReview:
		return m.handleReviewKey(key)
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
		if key.String() == keyEsc || key.String() == "q" || key.String() == "?" {
			m.mode = modeReview
		}

		return m, nil
	}

	return m, nil
}

func (m *model) handleReviewKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "tab":
		m.focus = (m.focus + 1) % 3
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
	case "j", keyDown:
		if m.focus == focusFiles {
			return m.selectFile(m.fileIndex + 1)
		}
		m.moveLine(1)
	case "k", "up":
		if m.focus == focusFiles {
			return m.selectFile(m.fileIndex - 1)
		}
		m.moveLine(-1)
	case keyEnter:
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
	case keyEsc:
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

func (m *model) handleGotoKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case keyEsc:
		m.mode = modeReview
	case keyEnter:
		matches := m.filteredFiles()
		if len(matches) > 0 {
			for i, file := range m.files {
				if file.Path == matches[m.gotoIndex].Path {
					m.mode = modeReview

					return m.selectFile(i)
				}
			}
		}
	case "j", keyDown:
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

func (m *model) handleSearchKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case keyEsc:
		m.mode = modeReview
	case keyEnter:
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

func (m *model) handleCommentKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commentActions {
		switch key.String() {
		case keyEsc:
			m.mode = modeReview
			m.commentActions = false
		case keyUp, "k":
			m.commentActions = false
		case "left", "h", "right", "l", "tab":
			if m.commentAction == 0 {
				m.commentAction = 1
			} else {
				m.commentAction = 0
			}
		case keyEnter:
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
	case keyEsc:
		m.mode = modeReview
	case keyDown:
		if m.composerAtLastInputLine() {
			m.commentActions = true
			m.commentAction = 0

			return m, nil
		}

		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(key)

		return m, cmd
	case "ctrl+s":
		return m.submitComment()
	default:
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(key)

		return m, cmd
	}

	return m, nil
}

func (m *model) composerAtLastInputLine() bool {
	if m.composer.Line() < m.composer.LineCount()-1 {
		return false
	}

	line := m.composer.LineInfo()
	return line.RowOffset+1 >= line.Height
}

func (m *model) submitComment() (tea.Model, tea.Cmd) {
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

func (m *model) commentActionRow() string {
	submit := renderCommentButton("Submit", m.commentActions && m.commentAction == 0)
	cancel := renderCommentButton("Cancel", m.commentActions && m.commentAction == 1)

	return lipgloss.JoinHorizontal(lipgloss.Top, submit, "  ", cancel) +
		"\nEnter newline  Down actions  Up comment  Ctrl+S save  Esc close"
}

func renderCommentButton(label string, active bool) string {
	if active {
		return commentButtonActiveStyle.Render(label)
	}

	return commentButtonStyle.Render(label)
}

func (m *model) handleRequestKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case keyEsc:
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

func (m *model) handleConfirmKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case keyEsc, "n":
		m.mode = modeReview
	case "y":
		if m.mode == modeConfirmApprove {
			return m, approve(m.opts)
		}

		return m, requestChanges(m.opts, strings.TrimSpace(m.request.Value()))
	}

	return m, nil
}

func (m *model) handleContextKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case keyEsc:
		m.mode = modeReview
	case "j", keyDown:
		if m.contextIdx < len(m.context)-1 {
			m.contextIdx++
		}
	case "k", "up":
		if m.contextIdx > 0 {
			m.contextIdx--
		}
	case keyEnter:
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

func (m *model) resolveCurrent() (tea.Model, tea.Cmd) {
	comment := m.currentComment()
	if comment.ID == "" {
		m.status = "no unresolved comment on selected line"

		return m, nil
	}

	return m, func() tea.Msg {
		_, err := m.opts.Client.PatchComment(
			m.opts.RepoPath, m.opts.Session, comment.ID, map[string]bool{"resolved": true},
		)
		if err != nil {
			return errMsg{err}
		}

		return commentsLoadedMsg{comments: mustComments(m.opts), status: "comment resolved"}
	}
}

func (m *model) showDescription(generate bool) (tea.Model, tea.Cmd) {
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
