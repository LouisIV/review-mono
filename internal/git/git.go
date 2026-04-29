package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"review/internal/models"
)

type Repo struct {
	Path string
}

func Open(path string) (Repo, error) {
	if path == "" {
		path = "."
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return Repo{}, err
	}

	root, err := run(abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repo{}, fmt.Errorf("not a git repository: %w", err)
	}

	return Repo{Path: strings.TrimSpace(root)}, nil
}

func (r Repo) GitDir() (string, error) {
	out, err := run(r.Path, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}

	dir := strings.TrimSpace(out)
	if filepath.IsAbs(dir) {
		return dir, nil
	}

	return filepath.Join(r.Path, dir), nil
}

func (r Repo) Branch() (string, error) {
	out, err := run(r.Path, "branch", "--show-current")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(out)
	if branch == "" {
		return "", errors.New("detached HEAD is not supported")
	}

	return branch, nil
}

func (r Repo) Commits(base string) ([]models.Commit, error) {
	out, err := run(r.Path, "log", "--date=iso-strict", "--pretty=format:%H%x1f%h%x1f%an%x1f%aI%x1f%s", base+"..HEAD")
	if err != nil {
		return nil, err
	}

	var commits []models.Commit

	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, "\x1f", 5)
		if len(parts) != 5 {
			continue
		}

		ts, _ := time.Parse(time.RFC3339, parts[3])
		commits = append(commits, models.Commit{
			HashFull:  parts[0],
			Hash:      parts[1],
			Author:    parts[2],
			Timestamp: ts,
			Message:   parts[4],
		})
	}

	return commits, nil
}

func (r Repo) Diff(base, file, commit string, skipHunks bool) ([]models.DiffFile, string, error) {
	numstatArgs, diffArgs := makeDiffArgs(file, diffSpec(base, commit)...)
	numstat, err := run(r.Path, numstatArgs...)
	if err != nil {
		return nil, "", err
	}

	files := parseNumstat(numstat)

	raw, err := run(r.Path, diffArgs...)
	if err != nil {
		return nil, "", err
	}
	committedRaw := raw
	worktreeRaw := ""
	untrackedRaw := ""

	if commit == "" {
		worktreeNumstatArgs, worktreeDiffArgs := makeDiffArgs(file, "HEAD")
		worktreeNumstat, err := run(r.Path, worktreeNumstatArgs...)
		if err != nil {
			return nil, "", err
		}

		worktreeFiles := parseNumstat(worktreeNumstat)

		worktreeRaw, err = run(r.Path, worktreeDiffArgs...)
		if err != nil {
			return nil, "", err
		}

		untrackedFiles, nextUntrackedRaw, err := r.untrackedDiff(file)
		if err != nil {
			return nil, "", err
		}
		untrackedRaw = nextUntrackedRaw

		files = mergeDiffFiles(files, worktreeFiles)
		files = mergeDiffFiles(files, untrackedFiles)
		raw += worktreeRaw + untrackedRaw
	}

	if !skipHunks {
		addParsedHunks(files, parseUnifiedDiff(committedRaw, false))
		if commit == "" {
			addParsedHunks(files, parseUnifiedDiff(worktreeRaw+untrackedRaw, true))
		}
	}

	return files, raw, nil
}

func (r Repo) RawDiff(base string) (string, error) {
	_, diffArgs := makeDiffArgs("", diffSpec(base, "")...)
	raw, err := run(r.Path, diffArgs...)
	if err != nil {
		return "", err
	}

	_, worktreeDiffArgs := makeDiffArgs("", "HEAD")
	worktreeRaw, err := run(r.Path, worktreeDiffArgs...)
	if err != nil {
		return "", err
	}

	_, untrackedRaw, err := r.untrackedDiff("")
	if err != nil {
		return "", err
	}

	return raw + worktreeRaw + untrackedRaw, nil
}

func (r Repo) Push() error {
	_, err := run(r.Path, "push", "-u", "origin", "HEAD")

	return err
}

func (r Repo) RemoteURL() (string, error) {
	out, err := run(r.Path, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

func (r Repo) MRURL(branch string, body string) string {
	remote, err := r.RemoteURL()
	if err != nil {
		return ""
	}

	host, owner, repo := parseRemote(remote)
	if host == "" || owner == "" || repo == "" {
		return ""
	}

	q := url.Values{}
	if body != "" {
		q.Set("body", body)
	}

	if strings.Contains(host, "gitlab") {
		u := fmt.Sprintf(
			"https://%s/%s/%s/-/merge_requests/new?merge_request[source_branch]=%s",
			host,
			owner,
			repo,
			url.QueryEscape(branch),
		)
		if body != "" {
			u += "&merge_request[description]=" + url.QueryEscape(body)
		}

		return u
	}

	u := fmt.Sprintf("https://%s/%s/%s/compare/%s?expand=1", host, owner, repo, url.PathEscape(branch))
	if qs := q.Encode(); qs != "" {
		u += "&" + qs
	}

	return u
}

func run(dir string, args ...string) (string, error) {
	//nolint:gosec // All call sites pass fixed git subcommands with repo-derived arguments.
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}

		return "", errors.New(msg)
	}

	return string(out), nil
}

func runAllowExit(dir string, allowed []int, args ...string) (string, error) {
	//nolint:gosec // All call sites pass fixed git subcommands with repo-derived arguments.
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err == nil {
		return string(out), nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if slices.Contains(allowed, exitErr.ExitCode()) {
			return string(out), nil
		}
	}

	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}

	return "", errors.New(msg)
}

func diffSpec(base, commit string) []string {
	if commit != "" {
		return []string{commit + "^", commit}
	}

	return []string{base + "...HEAD"}
}

func makeDiffArgs(file string, spec ...string) ([]string, []string) {
	numstatArgs := append([]string{"diff", "--numstat"}, spec...)
	diffArgs := append([]string{"diff", "--no-color", "--find-renames"}, spec...)

	if file != "" {
		numstatArgs = append(numstatArgs, "--", file)
		diffArgs = append(diffArgs, "--", file)
	}

	return numstatArgs, diffArgs
}

func (r Repo) untrackedDiff(file string) ([]models.DiffFile, string, error) {
	args := []string{"ls-files", "--others", "--exclude-standard"}
	if file != "" {
		args = append(args, "--", file)
	}

	out, err := run(r.Path, args...)
	if err != nil {
		return nil, "", err
	}

	var files []models.DiffFile
	var raw strings.Builder

	for path := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		info, err := os.Stat(filepath.Join(r.Path, path))
		if err != nil {
			return nil, "", err
		}

		if info.IsDir() {
			continue
		}

		diff, err := runAllowExit(r.Path, []int{1}, "diff", "--no-color", "--no-index", "--", "/dev/null", path)
		if err != nil {
			return nil, "", err
		}

		numstat, err := runAllowExit(r.Path, []int{1}, "diff", "--no-index", "--numstat", "--", "/dev/null", path)
		if err != nil {
			return nil, "", err
		}

		files = append(files, parseNumstat(numstat)...)
		raw.WriteString(diff)
	}

	return files, raw.String(), nil
}

func parseNumstat(out string) []models.DiffFile {
	var files []models.DiffFile

	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		fields := strings.Split(sc.Text(), "\t")
		if len(fields) < 3 {
			continue
		}

		add, _ := strconv.Atoi(fields[0])
		del, _ := strconv.Atoi(fields[1])
		files = append(files, models.DiffFile{Path: normalizeNumstatPath(fields[2]), Additions: add, Deletions: del})
	}

	return files
}

func normalizeNumstatPath(path string) string {
	if after, ok := strings.CutPrefix(path, "/dev/null => "); ok {
		return after
	}

	return path
}

func mergeDiffFiles(files, additions []models.DiffFile) []models.DiffFile {
	for _, addition := range additions {
		found := false
		for i := range files {
			if files[i].Path == addition.Path {
				files[i].Additions += addition.Additions
				files[i].Deletions += addition.Deletions
				files[i].Hunks = append(files[i].Hunks, addition.Hunks...)
				found = true

				break
			}
		}

		if !found {
			files = append(files, addition)
		}
	}

	return files
}

func addParsedHunks(files []models.DiffFile, parsed map[string]models.DiffFile) {
	for i := range files {
		if parsedFile, ok := parsed[files[i].Path]; ok {
			files[i].Hunks = append(files[i].Hunks, parsedFile.Hunks...)
		}
	}
}

var hunkRe = regexp.MustCompile(`@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func parseUnifiedDiff(raw string, uncommitted bool) map[string]models.DiffFile {
	result := map[string]models.DiffFile{}

	var (
		current *models.DiffFile
		hunk    *models.DiffHunk
	)

	newLine := 0

	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := sc.Text()
		if after, ok := strings.CutPrefix(line, "+++ b/"); ok {
			path := after
			df := result[path]
			df.Path = path
			result[path] = df
			current = &df
			hunk = nil

			continue
		}

		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "@@ ") {
			m := hunkRe.FindStringSubmatch(line)
			if len(m) == 3 {
				newLine, _ = strconv.Atoi(m[2])
			}

			current.Hunks = append(current.Hunks, models.DiffHunk{Header: line, Uncommitted: uncommitted})
			hunk = &current.Hunks[len(current.Hunks)-1]
			result[current.Path] = *current

			continue
		}

		if hunk == nil || line == `\ No newline at end of file` {
			continue
		}

		switch {
		case strings.HasPrefix(line, "+"):
			n := newLine
			hunk.Lines = append(
				hunk.Lines,
				models.DiffLine{Type: "add", Number: &n, Content: strings.TrimPrefix(line, "+")},
			)
			newLine++
		case strings.HasPrefix(line, "-"):
			hunk.Lines = append(
				hunk.Lines,
				models.DiffLine{Type: "remove", Number: nil, Content: strings.TrimPrefix(line, "-")},
			)
		default:
			n := newLine
			hunk.Lines = append(
				hunk.Lines,
				models.DiffLine{Type: "context", Number: &n, Content: strings.TrimPrefix(line, " ")},
			)
			newLine++
		}

		result[current.Path] = *current
	}

	return result
}

func parseRemote(remote string) (string, string, string) {
	remote = strings.TrimSuffix(remote, ".git")
	if after, ok := strings.CutPrefix(remote, "git@"); ok {
		parts := strings.SplitN(after, ":", 2)
		if len(parts) == 2 {
			host := parts[0]

			path := strings.Split(parts[1], "/")
			if len(path) >= 2 {
				return host, path[len(path)-2], path[len(path)-1]
			}
		}

		return "", "", ""
	}

	u, err := url.Parse(remote)
	if err != nil {
		return "", "", ""
	}

	path := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(path) >= 2 {
		return u.Host, path[len(path)-2], path[len(path)-1]
	}

	return "", "", ""
}
