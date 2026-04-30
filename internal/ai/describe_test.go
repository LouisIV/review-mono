package ai_test

import (
	"strings"
	"testing"

	"review/internal/ai"
)

func TestFallbackDescriptionIncludesChangedFileContext(t *testing.T) {
	t.Parallel()

	diff := strings.Join([]string{
		"diff --git a/internal/tui/app.go b/internal/tui/app.go",
		"index 1111111..2222222 100644",
		"--- a/internal/tui/app.go",
		"+++ b/internal/tui/app.go",
		"@@ -10,6 +10,7 @@ func renderApp() {",
		"+\treturn",
		"diff --git a/internal/ai/describe.go b/internal/ai/describe.go",
		"index 3333333..4444444 100644",
		"--- a/internal/ai/describe.go",
		"+++ b/internal/ai/describe.go",
		"@@ -20,6 +20,7 @@ func fallbackDescription(diff string) string {",
		"-\treturn old",
		"+\treturn new",
	}, "\n")

	out, err := ai.GenerateDescription("", "fallback", diff, "")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Updates 2 file(s), with 2 additions and 1 deletions.",
		"- internal/tui/app.go: func renderApp() {",
		"- internal/ai/describe.go: func fallbackDescription(diff string) string {",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("fallback description missing %q:\n%s", want, out)
		}
	}
}

func TestFallbackDescriptionLimitsChangedFileContext(t *testing.T) {
	t.Parallel()

	lines := []string{}
	for i := 1; i <= 24; i++ {
		lines = append(
			lines,
			"diff --git a/file.go b/file.go",
			"--- a/file.go",
			"+++ b/file.go",
			"@@ -1,1 +1,1 @@ func changed() {",
			"+changed",
		)
	}

	out, err := ai.GenerateDescription("", "fallback", strings.Join(lines, "\n"), "")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "...and 4 more file(s)") {
		t.Fatalf("fallback description did not report omitted files:\n%s", out)
	}
	if got := strings.Count(out, "- file.go"); got != 20 {
		t.Fatalf("rendered %d changed file entries, want 20:\n%s", got, out)
	}
}

func TestGeneratedDescriptionIsCapped(t *testing.T) {
	t.Parallel()

	longPath := strings.Repeat("very-long-directory-name/", 120) + "file.go"
	lines := []string{}
	for i := 1; i <= 20; i++ {
		lines = append(
			lines,
			"diff --git a/"+longPath+" b/"+longPath,
			"--- a/"+longPath,
			"+++ b/"+longPath,
			"@@ -1,1 +1,1 @@ "+strings.Repeat("long hunk context ", 40),
			"+changed",
		)
	}

	out, err := ai.GenerateDescription("", "fallback", strings.Join(lines, "\n"), "")
	if err != nil {
		t.Fatal(err)
	}

	if len(out) > 4000 {
		t.Fatalf("description length = %d, want <= 4000", len(out))
	}
	if !strings.Contains(out, "[description truncated]") {
		t.Fatalf("description missing truncation marker:\n%s", out)
	}
}
