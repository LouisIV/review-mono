package tui

import (
	"slices"
	"strconv"

	"review/internal/models"
)

const rowKindExpand = "expand"

func hunkBounds(hunk models.DiffHunk) (int, int) {
	start, end := 0, 0
	for _, line := range hunk.Lines {
		if line.Number == nil {
			continue
		}
		n := *line.Number
		if start == 0 || n < start {
			start = n
		}
		if n > end {
			end = n
		}
	}
	if start == 0 {
		start = hunk.NewStart
		end = hunk.NewStart - 1
	}

	return start, end
}

func appendExpandedGap(
	rows []diffRow,
	content []string,
	start int,
	end int,
	expanded []expandedRange,
) []diffRow {
	if start < 1 {
		start = 1
	}
	if end > len(content) {
		end = len(content)
	}
	if start > end {
		return rows
	}

	cursor := start
	for _, r := range normalizeExpanded(expanded, start, end) {
		rows = appendExpansionSlot(rows, cursor, r.start-1, len(content))
		for line := r.start; line <= r.end; line++ {
			rows = append(rows, diffRow{
				kind:    "context",
				line:    line,
				content: content[line-1],
			})
		}
		cursor = r.end + 1
	}

	return appendExpansionSlot(rows, cursor, end, len(content))
}

func appendExpansionSlot(rows []diffRow, start int, end int, lineCount int) []diffRow {
	if start > end {
		return rows
	}
	edge := "middle"
	switch {
	case start == 1:
		edge = "start"
	case end == lineCount:
		edge = "end"
	}

	return append(rows, diffRow{
		kind:        rowKindExpand,
		content:     expansionLabel(start, end),
		expandStart: start,
		expandEnd:   end,
		expandEdge:  edge,
	})
}

func expansionLabel(start, end int) string {
	if start == end {
		return "show line " + strconv.Itoa(start)
	}

	return "show lines " + strconv.Itoa(start) + "-" + strconv.Itoa(end)
}

func normalizeExpanded(expanded []expandedRange, start int, end int) []expandedRange {
	out := make([]expandedRange, 0, len(expanded))
	for _, r := range expanded {
		if r.end < start || r.start > end {
			continue
		}
		r.start = max(r.start, start)
		r.end = min(r.end, end)
		if r.start > r.end {
			continue
		}
		if len(out) > 0 && r.start <= out[len(out)-1].end+1 {
			out[len(out)-1].end = max(out[len(out)-1].end, r.end)
			continue
		}
		out = append(out, r)
	}

	return out
}

func rowSelectable(row diffRow) bool {
	return row.line > 0 || row.kind == "hunk" || row.kind == rowKindExpand
}

func (m *model) expandCurrentSlot() {
	if len(m.rows) == 0 || m.lineIndex >= len(m.rows) {
		return
	}
	row := m.rows[m.lineIndex]
	if row.kind != rowKindExpand {
		return
	}
	file := m.currentFile()
	if file == "" {
		return
	}
	if m.expanded == nil {
		m.expanded = map[string][]expandedRange{}
	}

	const expandLines = 12
	start, end := expansionWindow(row, expandLines)
	m.expanded[file] = mergeExpanded(append(m.expanded[file], expandedRange{start: start, end: end}))
	current := m.files[m.fileIndex]
	m.rows = flatten(current, m.expanded[file])
	m.lineIndex = closestRowForLine(m.rows, start, m.lineIndex)
	m.ensureVisible()
	m.status = "expanded context"
}

func expansionWindow(row diffRow, count int) (int, int) {
	start, end := row.expandStart, row.expandEnd
	if end-start+1 <= count {
		return start, end
	}
	switch row.expandEdge {
	case "start":
		return end - count + 1, end
	case "end":
		return start, start + count - 1
	default:
		mid := start + (end-start)/2
		nextStart := mid - count/2 + 1
		nextEnd := nextStart + count - 1
		if nextStart < start {
			nextStart = start
			nextEnd = start + count - 1
		}
		if nextEnd > end {
			nextEnd = end
			nextStart = end - count + 1
		}

		return nextStart, nextEnd
	}
}

func mergeExpanded(expanded []expandedRange) []expandedRange {
	if len(expanded) < 2 {
		return expanded
	}
	slices.SortFunc(expanded, func(a, b expandedRange) int {
		return a.start - b.start
	})

	merged := expanded[:0]
	for _, r := range expanded {
		if r.start > r.end {
			continue
		}
		if len(merged) > 0 && r.start <= merged[len(merged)-1].end+1 {
			merged[len(merged)-1].end = max(merged[len(merged)-1].end, r.end)
			continue
		}
		merged = append(merged, r)
	}

	return merged
}
