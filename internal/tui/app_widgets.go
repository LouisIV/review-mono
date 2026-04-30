package tui

import "review/internal/tui/widgets"

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
			Author:   comment.Author,
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
