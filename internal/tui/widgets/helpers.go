package widgets

import (
	"sort"
	"strings"
)

func actionsFor(target string) []string {
	switch target {
	case "visual-selection":
		return []string{"Comment on range", "Clear selection", "Copy range", "Open range in editor"}
	case "comment":
		return []string{"Resolve", "Edit", "Delete", "Copy comment ID"}
	case "file":
		return []string{"Go to file", "Add file comment", "Copy path", "Open in editor"}
	default:
		return []string{"Add comment", "Start visual selection", "Copy file:line", "Open in editor"}
	}
}

func filteredFiles(query string) []fileRow {
	query = strings.ToLower(strings.TrimSpace(query))
	out := []fileRow{}
	for _, file := range files {
		if query == "" || fuzzyContains(strings.ToLower(file.Path), query) {
			out = append(out, file)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Unresolved > out[j].Unresolved
	})

	return out
}

func fuzzyContains(s, query string) bool {
	pos := 0
	for _, r := range query {
		idx := strings.IndexRune(s[pos:], r)
		if idx < 0 {
			return false
		}

		pos += idx + 1
	}

	return true
}

func highlight(s, query string) string {
	if query == "" {
		return s
	}

	lower := strings.ToLower(s)
	q := strings.ToLower(query)
	idx := strings.Index(lower, q)
	if idx < 0 {
		return s
	}

	return s[:idx] + searchStyle.Render(s[idx:idx+len(query)]) + s[idx+len(query):]
}

func truncateMiddle(s string, width int) string {
	if width <= 0 {
		return ""
	}

	if len(s) <= width {
		return s
	}

	if width < 5 {
		return s[:width]
	}

	left := (width - 1) / 2
	right := width - left - 1

	return s[:left] + "~" + s[len(s)-right:]
}
