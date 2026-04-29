//nolint:testpackage
package tui

import (
	"fmt"
	"testing"

	"review/internal/models"
)

var benchmarkViewSink string

func BenchmarkModelViewLargeReview(b *testing.B) {
	m := newBenchmarkModel()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkViewSink = m.View()
	}
}

func newBenchmarkModel() *model {
	rows := benchmarkDiffRows(5000)
	comments := benchmarkComments(400)

	return &model{
		width:     120,
		height:    36,
		mode:      modeReview,
		focus:     focusDiff,
		session:   models.Session{Branch: "feature/perf", Base: "main", Status: "open"},
		commits:   []models.Commit{{Hash: "abc123"}, {Hash: "def456"}},
		files:     benchmarkFiles(250),
		comments:  comments,
		fileIndex: 0,
		rows:      rows,
		diffItems: toWidgetRows(rows),
		lineIndex: 120,
		top:       105,
		viewed:    map[string]bool{"internal/tui/widgets/review.go": true},
	}
}

func benchmarkDiffRows(count int) []diffRow {
	rows := make([]diffRow, 0, count+count/40)
	for i := range count {
		if i%40 == 0 {
			rows = append(rows, diffRow{
				kind:    "hunk",
				hunk:    i / 40,
				content: fmt.Sprintf("@@ -%d,40 +%d,40 @@ func render%d()", i+1, i+1, i),
			})
		}

		kind := "context"
		switch i % 9 {
		case 0, 1:
			kind = "add"
		case 2:
			kind = "remove"
		}

		rows = append(rows, diffRow{
			kind:    kind,
			hunk:    i / 40,
			line:    i + 1,
			content: fmt.Sprintf("\tif row%d.Kind == %q { return fmt.Sprintf(\"value %%d\", row%d.Line) }", i, kind, i),
		})
	}

	return rows
}

func benchmarkFiles(count int) []models.DiffFile {
	files := make([]models.DiffFile, 0, count)
	files = append(files, models.DiffFile{
		Path:      "internal/tui/widgets/review.go",
		Additions: 350,
		Deletions: 40,
	})
	for i := 1; i < count; i++ {
		files = append(files, models.DiffFile{
			Path:      fmt.Sprintf("internal/package_%03d/file_%03d.go", i%20, i),
			Additions: i % 90,
			Deletions: i % 11,
		})
	}

	return files
}

func benchmarkComments(count int) []models.Comment {
	comments := make([]models.Comment, 0, count)
	for i := range count {
		file := "internal/tui/widgets/review.go"
		if i%3 != 0 {
			file = fmt.Sprintf("internal/package_%03d/file_%03d.go", i%20, i%250)
		}
		comments = append(comments, models.Comment{
			ID:       fmt.Sprintf("C-%04d", i),
			File:     file,
			Line:     i*7 + 1,
			Body:     "Please keep this render path cheap.",
			Resolved: i%5 == 0,
		})
	}

	return comments
}

func BenchmarkToWidgetRowsLargeDiff(b *testing.B) {
	rows := benchmarkDiffRows(10000)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if len(toWidgetRows(rows)) == 0 {
			b.Fatal("empty widget rows")
		}
	}
}
