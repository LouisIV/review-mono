package tui

import (
	"fmt"

	"review/internal/tui/widgets"
)

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
