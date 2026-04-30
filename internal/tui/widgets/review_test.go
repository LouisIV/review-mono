package widgets_test

import (
	"fmt"
	"strings"
	"testing"

	"review/internal/tui/widgets"
)

func TestRenderRuntimeDiffUsesAvailableRows(t *testing.T) {
	t.Parallel()

	const height = 28
	bottomHeight := 5
	bodyHeight := height - bottomHeight - 8
	visibleRows := bodyHeight - 2
	rows := make([]widgets.DiffItem, 0, visibleRows+1)
	for i := 1; i <= visibleRows+1; i++ {
		rows = append(rows, widgets.DiffItem{Kind: "context", Line: i, Content: fmt.Sprintf("visible-row-%02d", i)})
	}

	out := widgets.RenderReviewWorkspace(100, height, widgets.WorkspaceData{
		Branch:     "feature",
		Base:       "main",
		Status:     "in_review",
		Files:      []widgets.FileItem{{Path: "file.go"}},
		Rows:       rows,
		ActiveFile: "file.go",
		Focus:      "diff",
	})

	lastVisible := fmt.Sprintf("visible-row-%02d", visibleRows)
	firstClipped := fmt.Sprintf("visible-row-%02d", visibleRows+1)
	if !strings.Contains(out, lastVisible) {
		t.Fatalf("rendered diff did not include the last available row:\n%s", out)
	}
	if strings.Contains(out, firstClipped) {
		t.Fatalf("rendered diff overflowed available rows:\n%s", out)
	}
}

func TestRenderRuntimeDiffUsesUncommittedIndicator(t *testing.T) {
	t.Parallel()

	out := widgets.RenderReviewWorkspace(100, 28, widgets.WorkspaceData{
		Branch:     "feature",
		Base:       "main",
		Status:     "in_review",
		Files:      []widgets.FileItem{{Path: "file.go"}},
		ActiveFile: "file.go",
		Focus:      "diff",
		Rows: []widgets.DiffItem{
			{Kind: "add", Line: 1, Content: "committed"},
			{Kind: "add", Line: 2, Content: "worktree", Uncommitted: true},
		},
	})

	if !strings.Contains(out, "▌") {
		t.Fatalf("rendered diff did not include committed indicator:\n%s", out)
	}

	if !strings.Contains(out, "▒") {
		t.Fatalf("rendered diff did not include uncommitted indicator:\n%s", out)
	}
}

func TestRenderRuntimeFilesScrollsToActiveFile(t *testing.T) {
	t.Parallel()

	files := make([]widgets.FileItem, 0, 24)
	for i := 1; i <= 24; i++ {
		files = append(files, widgets.FileItem{Path: fmt.Sprintf("file-%02d.go", i)})
	}

	out := widgets.RenderReviewWorkspace(100, 28, widgets.WorkspaceData{
		Branch:     "feature",
		Base:       "main",
		Status:     "in_review",
		Files:      files,
		ActiveFile: "file-23.go",
		Focus:      "files",
	})

	if !strings.Contains(out, "file-23.go") {
		t.Fatalf("rendered files did not include active file:\n%s", out)
	}
	if strings.Contains(out, "file-01.go") {
		t.Fatalf("rendered files did not scroll past the first page:\n%s", out)
	}
}
