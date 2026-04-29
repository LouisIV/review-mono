//nolint:testpackage
package widgets

import (
	"fmt"
	"testing"
)

var benchmarkRenderSink string

func BenchmarkRenderReviewWorkspaceLargeDiff(b *testing.B) {
	data := benchmarkWorkspaceData(5000, 250, 400)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkRenderSink = RenderReviewWorkspace(120, 36, data)
	}
}

func BenchmarkUnresolvedCommentBadges(b *testing.B) {
	data := benchmarkWorkspaceData(100, 10, 5000)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if len(unresolvedCommentBadgeIDs(data)) == 0 {
			b.Fatal("empty badge map")
		}
	}
}

func benchmarkWorkspaceData(rowCount, fileCount, commentCount int) WorkspaceData {
	return WorkspaceData{
		Branch:       "feature/perf",
		Base:         "main",
		Status:       "open",
		CommitCount:  2,
		Files:        benchmarkFileItems(fileCount),
		Rows:         benchmarkDiffItems(rowCount),
		Comments:     benchmarkCommentItems(commentCount),
		ActiveFile:   "internal/tui/widgets/review.go",
		SelectedLine: 121,
		Top:          105,
		Focus:        "diff",
		Query:        "comment",
		ShowResolved: true,
	}
}

func benchmarkDiffItems(count int) []DiffItem {
	rows := make([]DiffItem, 0, count+count/40)
	for i := range count {
		if i%40 == 0 {
			rows = append(rows, DiffItem{
				Kind:    "hunk",
				Hunk:    i / 40,
				Content: fmt.Sprintf("@@ -%d,40 +%d,40 @@ func render%d()", i+1, i+1, i),
			})
		}

		kind := "context"
		switch i % 9 {
		case 0, 1:
			kind = lineKindAdd
		case 2:
			kind = lineKindRemove
		}

		rows = append(rows, DiffItem{
			Kind:    kind,
			Hunk:    i / 40,
			Line:    i + 1,
			Content: fmt.Sprintf("\tif row%d.Kind == %q { return fmt.Sprintf(\"value %%d\", row%d.Line) }", i, kind, i),
		})
	}

	return rows
}

func benchmarkFileItems(count int) []FileItem {
	files := make([]FileItem, 0, count)
	files = append(files, FileItem{
		Path:       "internal/tui/widgets/review.go",
		Additions:  350,
		Deletions:  40,
		Unresolved: 12,
		Viewed:     true,
	})
	for i := 1; i < count; i++ {
		files = append(files, FileItem{
			Path:       fmt.Sprintf("internal/package_%03d/file_%03d.go", i%20, i),
			Additions:  i % 90,
			Deletions:  i % 11,
			Unresolved: i % 4,
			Viewed:     i%2 == 0,
		})
	}

	return files
}

func benchmarkCommentItems(count int) []CommentItem {
	comments := make([]CommentItem, 0, count)
	for i := range count {
		file := "internal/tui/widgets/review.go"
		if i%3 != 0 {
			file = fmt.Sprintf("internal/package_%03d/file_%03d.go", i%20, i%250)
		}
		comments = append(comments, CommentItem{
			ID:       fmt.Sprintf("C-%04d", i),
			File:     file,
			Line:     i*7 + 1,
			Body:     "Please keep this render path cheap.",
			Resolved: i%5 == 0,
		})
	}

	return comments
}
