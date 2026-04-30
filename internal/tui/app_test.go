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

func newCommentTestModel(value string) *model {
	m := newModel(Options{})
	m.mode = modeComment
	m.composer.Focus()
	m.composer.SetValue(value)

	return m
}
