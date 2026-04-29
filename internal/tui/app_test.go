//nolint:testpackage
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func newCommentTestModel(value string) *model {
	m := newModel(Options{})
	m.mode = modeComment
	m.composer.Focus()
	m.composer.SetValue(value)

	return m
}
