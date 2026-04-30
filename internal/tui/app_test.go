//nolint:testpackage
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"review/internal/models"
)

func TestCommentDownMovesWithinComposerBeforeLastLine(t *testing.T) {
	t.Parallel()
	m := newCommentTestModel("first\nsecond")
	m.composer.CursorUp()

	next, _ := m.handleCommentKey(tea.KeyMsg{Type: tea.KeyDown})
	got := next.(*model) //nolint:forcetypeassert

	if got.commentActions {
		t.Fatal("down on a non-final composer line entered comment actions")
	}
	if got.composer.Line() != 1 {
		t.Fatalf("composer cursor line = %d, want 1", got.composer.Line())
	}
}

func TestCommentDownEntersActionsOnLastLine(t *testing.T) {
	t.Parallel()
	m := newCommentTestModel("first\nsecond")

	next, _ := m.handleCommentKey(tea.KeyMsg{Type: tea.KeyDown})
	got := next.(*model) //nolint:forcetypeassert

	if !got.commentActions {
		t.Fatal("down on final composer line did not enter comment actions")
	}
	if got.commentAction != 0 {
		t.Fatalf("comment action = %d, want 0", got.commentAction)
	}
}

func TestCommentUpReturnsFromActionsToComposer(t *testing.T) {
	t.Parallel()
	m := newCommentTestModel("comment")
	m.commentActions = true

	next, _ := m.handleCommentKey(tea.KeyMsg{Type: tea.KeyUp})
	got := next.(*model) //nolint:forcetypeassert

	if got.commentActions {
		t.Fatal("up from comment actions did not return to composer")
	}
}

func TestCommentActionRowRendersButtons(t *testing.T) {
	t.Parallel()
	m := newCommentTestModel("comment")
	m.commentActions = true

	row := m.commentActionRow()

	if !strings.Contains(row, "Submit") || !strings.Contains(row, "Cancel") {
		t.Fatalf("comment action row = %q, want submit and cancel labels", row)
	}
	if !strings.Contains(row, "Up comment") {
		t.Fatalf("comment action help = %q, want up navigation hint", row)
	}
}

func TestDiffHeightMatchesRenderedViewportRows(t *testing.T) {
	t.Parallel()
	m := newModel(Options{})
	m.height = 28

	if got, want := m.diffHeight(), 13; got != want {
		t.Fatalf("diffHeight() = %d, want %d", got, want)
	}
}

func TestBottomBodyAllowsCurrentFileCommentsToRender(t *testing.T) {
	t.Parallel()
	m := newModel(Options{})
	m.files = []models.DiffFile{{Path: "a.go"}}
	m.comments = []models.Comment{{ID: "C1", File: "a.go", Line: 3, Body: "looks off"}}

	if got := m.bottomBody(); got != "" {
		t.Fatalf("bottomBody() = %q, want comments fallback", got)
	}
}

func TestBottomBodyShowsCommentHintWithoutCurrentFileComments(t *testing.T) {
	t.Parallel()
	m := newModel(Options{})
	m.files = []models.DiffFile{{Path: "a.go"}}
	m.rows = []diffRow{{line: 3}}

	got := m.bottomBody()
	if !strings.Contains(got, "Press c to add a comment") {
		t.Fatalf("bottomBody() = %q, want comment hint", got)
	}
}

func TestContextEditPrefillsCommentComposer(t *testing.T) {
	t.Parallel()
	m := newModel(Options{})
	m.files = []models.DiffFile{{Path: "a.go"}}
	m.rows = []diffRow{{line: 7}}
	m.comments = []models.Comment{{ID: "C1", File: "a.go", Line: 7, Body: "old body"}}
	m.mode = modeContext
	m.context = []string{"Edit"}

	next, cmd := m.handleContextKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(*model) //nolint:forcetypeassert

	if cmd != nil {
		t.Fatal("edit context action returned a command")
	}
	if got.mode != modeComment {
		t.Fatalf("mode = %v, want modeComment", got.mode)
	}
	if got.editCommentID != "C1" {
		t.Fatalf("editCommentID = %q, want C1", got.editCommentID)
	}
	if got.composer.Value() != "old body" {
		t.Fatalf("composer value = %q, want old body", got.composer.Value())
	}
}

func TestSubmitCommentEditReturnsPatchCommand(t *testing.T) {
	t.Parallel()
	m := newCommentTestModel("new body")
	m.editCommentID = "C1"

	next, cmd := m.submitComment()
	got := next.(*model) //nolint:forcetypeassert

	if cmd == nil {
		t.Fatal("submit edit returned nil command")
	}
	if got.mode != modeReview {
		t.Fatalf("mode = %v, want modeReview", got.mode)
	}
	if got.editCommentID != "" {
		t.Fatalf("editCommentID = %q, want empty", got.editCommentID)
	}
}

func TestFlattenIncludesSelectableHunksAndExpansionSlots(t *testing.T) {
	t.Parallel()

	file := expansionTestFile()
	rows := flatten(file, nil)

	if len(rows) < 3 {
		t.Fatalf("rows = %#v, want expansion slots and hunk", rows)
	}
	if rows[0].kind != rowKindExpand || rows[0].expandStart != 1 || rows[0].expandEnd != 6 {
		t.Fatalf("first row = %#v, want start expansion slot for lines 1-6", rows[0])
	}
	if rows[1].kind != "hunk" {
		t.Fatalf("second row = %#v, want selectable hunk", rows[1])
	}
	if rows[len(rows)-1].kind != rowKindExpand ||
		rows[len(rows)-1].expandStart != 12 ||
		rows[len(rows)-1].expandEnd != 24 {
		t.Fatalf("last row = %#v, want end expansion slot for lines 12-24", rows[len(rows)-1])
	}
	if got := firstSelectable(rows); got != 0 {
		t.Fatalf("first selectable = %d, want start expansion slot", got)
	}
}

func TestEnterExpandsCurrentSlotByWindow(t *testing.T) {
	t.Parallel()

	m := newModel(Options{})
	m.files = []models.DiffFile{expansionTestFile()}
	m.rows = flatten(m.files[0], nil)
	m.lineIndex = len(m.rows) - 1

	next, cmd := m.handleReviewKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(*model) //nolint:forcetypeassert

	if cmd != nil {
		t.Fatal("expanding context returned a command")
	}
	if len(got.expanded["a.go"]) != 1 {
		t.Fatalf("expanded ranges = %#v, want one range", got.expanded["a.go"])
	}
	if r := got.expanded["a.go"][0]; r.start != 12 || r.end != 23 {
		t.Fatalf("expanded range = %#v, want lines 12-23", r)
	}
	last := got.rows[len(got.rows)-1]
	if last.kind != rowKindExpand || last.expandStart != 24 || last.expandEnd != 24 {
		t.Fatalf("last row = %#v, want remaining expansion slot for line 24", last)
	}
}

func expansionTestFile() models.DiffFile {
	content := make([]string, 24)
	for i := range content {
		content[i] = "line"
	}
	line7, line8, line9, line10, line11 := 7, 8, 9, 10, 11

	return models.DiffFile{
		Path:         "a.go",
		ContentLines: content,
		Hunks: []models.DiffHunk{{
			Header:   "@@ -7,5 +7,5 @@",
			NewStart: 7,
			Lines: []models.DiffLine{
				{Type: "context", Number: &line7, Content: "line"},
				{Type: "context", Number: &line8, Content: "line"},
				{Type: "remove", Content: "old"},
				{Type: "add", Number: &line9, Content: "new"},
				{Type: "context", Number: &line10, Content: "line"},
				{Type: "context", Number: &line11, Content: "line"},
			},
		}},
	}
}

func newCommentTestModel(value string) *model {
	m := newModel(Options{})
	m.mode = modeComment
	m.composer.Focus()
	m.composer.SetValue(value)

	return m
}
