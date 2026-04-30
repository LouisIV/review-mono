package git_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"review/internal/git"
	"review/internal/models"
)

func TestDiffIncludesUncommittedFiles(t *testing.T) {
	t.Parallel()

	repoPath := initTestRepo(t)
	runGit(t, repoPath, "checkout", "-b", "feature")

	writeFile(t, repoPath, "committed.txt", "committed\n")
	runGit(t, repoPath, "add", "committed.txt")
	runGit(t, repoPath, "commit", "-m", "add committed")

	writeFile(t, repoPath, "base.txt", "base\nunstaged\n")
	writeFile(t, repoPath, "staged.txt", "staged\n")
	runGit(t, repoPath, "add", "staged.txt")
	writeFile(t, repoPath, "untracked.txt", "untracked\n")

	repo, err := git.Open(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	files, raw, err := repo.Diff("main", "", "", false)
	if err != nil {
		t.Fatal(err)
	}

	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}

	for _, want := range []string{"base.txt", "committed.txt", "staged.txt", "untracked.txt"} {
		if !slices.Contains(paths, want) {
			t.Fatalf("diff paths = %v, want %q", paths, want)
		}
	}

	if raw == "" {
		t.Fatal("raw diff is empty")
	}

	committed := findDiffFile(files, "committed.txt")
	if committed == nil {
		t.Fatal("committed.txt not found")
	}

	if len(committed.Hunks) != 1 || committed.Hunks[0].Uncommitted {
		t.Fatalf("committed hunks = %#v, want one committed hunk", committed.Hunks)
	}

	unstaged := findDiffFile(files, "base.txt")
	if unstaged == nil {
		t.Fatal("base.txt not found")
	}

	if len(unstaged.Hunks) != 1 || !unstaged.Hunks[0].Uncommitted {
		t.Fatalf("unstaged hunks = %#v, want one uncommitted hunk", unstaged.Hunks)
	}

	untracked := findDiffFile(files, "untracked.txt")
	if untracked == nil {
		t.Fatal("untracked.txt not found")
	}

	if untracked.Additions != 1 || untracked.Deletions != 0 {
		t.Fatalf("untracked stats = +%d -%d, want +1 -0", untracked.Additions, untracked.Deletions)
	}

	if len(untracked.Hunks) != 1 || len(untracked.Hunks[0].Lines) != 1 {
		t.Fatalf("untracked hunks = %#v, want one added line", untracked.Hunks)
	}

	if !untracked.Hunks[0].Uncommitted {
		t.Fatalf("untracked hunks = %#v, want uncommitted hunk", untracked.Hunks)
	}

	if got := untracked.Hunks[0].Lines[0].Content; got != "untracked" {
		t.Fatalf("untracked line content = %q, want %q", got, "untracked")
	}
}

func TestDiffFiltersUntrackedFile(t *testing.T) {
	t.Parallel()

	repoPath := initTestRepo(t)
	runGit(t, repoPath, "checkout", "-b", "feature")
	writeFile(t, repoPath, "untracked.txt", "untracked\n")

	repo, err := git.Open(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	files, raw, err := repo.Diff("main", "untracked.txt", "", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 || files[0].Path != "untracked.txt" {
		t.Fatalf("files = %#v, want only untracked.txt", files)
	}

	if raw == "" {
		t.Fatal("raw diff is empty")
	}
}

func TestDiffExposesContentLinesAndHunkStart(t *testing.T) {
	t.Parallel()

	repoPath := initTestRepo(t)
	runGit(t, repoPath, "checkout", "-b", "feature")

	lines := make([]string, 0, 30)
	for i := 1; i <= 30; i++ {
		lines = append(lines, fmt.Sprintf("line %02d", i))
	}
	writeFile(t, repoPath, "long.txt", strings.Join(lines, "\n")+"\n")
	runGit(t, repoPath, "add", "long.txt")
	runGit(t, repoPath, "commit", "-m", "add long file")

	lines[14] = "changed line 15"
	writeFile(t, repoPath, "long.txt", strings.Join(lines, "\n")+"\n")
	runGit(t, repoPath, "add", "long.txt")
	runGit(t, repoPath, "commit", "-m", "change long file")

	repo, err := git.Open(repoPath)
	if err != nil {
		t.Fatal(err)
	}

	files, _, err := repo.Diff("main", "long.txt", "", false)
	if err != nil {
		t.Fatal(err)
	}

	file := findDiffFile(files, "long.txt")
	if file == nil {
		t.Fatal("long.txt not found")
	}
	if got, want := len(file.ContentLines), 30; got != want {
		t.Fatalf("content lines = %d, want %d", got, want)
	}
	if got := file.ContentLines[14]; got != "changed line 15" {
		t.Fatalf("content line 15 = %q, want changed line", got)
	}
	if len(file.Hunks) != 1 || file.Hunks[0].NewStart == 0 {
		t.Fatalf("hunks = %#v, want parsed new start", file.Hunks)
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	writeFile(t, dir, "base.txt", "base\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "base")

	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	//nolint:gosec // Test helpers pass fixed git subcommands and temp paths.
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func findDiffFile(files []models.DiffFile, path string) *models.DiffFile {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}

	return nil
}
