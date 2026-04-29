package widgets

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderRuntimeDiffUsesAvailableRows(t *testing.T) {
	t.Parallel()

	const height = 28
	bottomHeight := 5
	bodyHeight := height - bottomHeight - 8
	visibleRows := bodyHeight - 2
	rows := make([]DiffItem, 0, visibleRows+1)
	for i := 1; i <= visibleRows+1; i++ {
		rows = append(rows, DiffItem{Kind: "context", Line: i, Content: fmt.Sprintf("visible-row-%02d", i)})
	}

	out := RenderReviewWorkspace(100, height, WorkspaceData{
		Branch:     "feature",
		Base:       "main",
		Status:     "in_review",
		Files:      []FileItem{{Path: "file.go"}},
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
