package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"review/internal/client"
	"review/internal/config"
	"review/internal/git"
	"review/internal/models"
)

func current(g globals) (models.Session, git.Repo, client.Client, error) {
	repo, err := git.Open(g.repo)
	if err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}

	if err := ensureDaemon(g.port); err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}

	c := client.New(g.port)

	sessions, err := c.Sessions(repo.Path)
	if err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}

	branch, err := repo.Branch()
	if err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}

	for i := len(sessions) - 1; i >= 0; i-- {
		if sessions[i].Repo == repo.Path && sessions[i].Branch == branch &&
			sessions[i].Status != models.StatusApproved {
			return sessions[i], repo, c, nil
		}
	}

	return models.Session{}, git.Repo{}, client.Client{}, errors.New("no active review session; run review open")
}

func parseGlobalFlags(args []string, g *globals) []string {
	out := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			g.json = true
		case "--repo":
			if i+1 < len(args) {
				g.repo = args[i+1]
				i++
			}
		case "--port":
			if i+1 < len(args) {
				g.port, _ = strconv.Atoi(args[i+1])
				i++
			}
		default:
			out = append(out, args[i])
		}
	}

	return out
}

func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func ensureDaemon(port int) error {
	c := client.New(port)
	if c.Health() == nil {
		return nil
	}

	if _, err := startDaemon(port); err != nil {
		return err
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c.Health() == nil {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return errors.New("daemon did not become healthy")
}

func startDaemon(port int) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}

	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}

	//nolint:gosec // Log path is scoped under the user config directory.
	logFile, err := os.OpenFile(filepath.Join(dir, "daemon.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}

	//nolint:gosec // Starts the current review executable as its own daemon process.
	cmd := exec.CommandContext(context.Background(), exe, "daemon", "run", "--port", strconv.Itoa(port))
	cmd.Stdout = logFile

	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	_ = os.WriteFile(filepath.Join(dir, "daemon.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o600)
	pid := cmd.Process.Pid

	return pid, cmd.Process.Release()
}

func stopDaemon(jsonOut bool) error {
	pid, err := readPID()
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	_ = os.Remove(filepath.Join(config.ConfigDir(), "daemon.pid"))

	if jsonOut {
		printJSON(map[string]any{"stopped": true, "pid": pid})
	} else {
		fmt.Printf("Stopped daemon pid %d\n", pid)
	}

	return nil
}

func daemonStatus(g globals) error {
	running := client.New(g.port).Health() == nil

	pid, _ := readPID()
	switch {
	case g.json:
		printJSON(map[string]any{"running": running, "port": g.port, "pid": pid})
	case running:
		fmt.Printf("Daemon running on port %d", g.port)

		if pid > 0 {
			fmt.Printf(" (pid %d)", pid)
		}

		fmt.Println()
	default:
		fmt.Println("Daemon not running")
	}

	return nil
}

func readPID() (int, error) {
	b, err := os.ReadFile(filepath.Join(config.ConfigDir(), "daemon.pid"))
	if err != nil {
		return 0, errors.New("daemon pid not found")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

func parseLocation(s string) (string, int, error) {
	idx := strings.LastIndex(s, ":")
	if idx <= 0 || idx == len(s)-1 {
		return "", 0, errors.New("location must be file:line")
	}

	line, err := strconv.Atoi(s[idx+1:])
	if err != nil {
		return "", 0, err
	}

	return s[:idx], line, nil
}

func editDescription(repoPath string, session models.Session, c client.Client) error {
	desc, _ := c.Description(repoPath, session)

	tmp, err := os.CreateTemp("", "review-description-*.md")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}()

	if _, err := tmp.WriteString(desc.Body); err != nil {
		return err
	}

	_ = tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	//nolint:gosec // EDITOR is intentionally user-controlled for local editing.
	cmd := exec.CommandContext(context.Background(), editor, tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		return err
	}

	_, err = c.SetDescription(repoPath, session, string(b))

	return err
}

func openBrowser(rawurl string) error {
	switch runtime.GOOS {
	case "darwin":
		//nolint:gosec // Opens a URL using the platform browser helper.
		return exec.CommandContext(context.Background(), "open", rawurl).Start()
	case "windows":
		//nolint:gosec // Opens a URL using the platform browser helper.
		return exec.CommandContext(context.Background(), "rundll32", "url.dll,FileProtocolHandler", rawurl).Start()
	default:
		//nolint:gosec // Opens a URL using the platform browser helper.
		return exec.CommandContext(context.Background(), "xdg-open", rawurl).Start()
	}
}

func printJSON(value any) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Println(`{"error":"failed to encode json"}`)

		return
	}

	fmt.Println(string(b))
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	if limit <= 1 {
		return s[:limit]
	}

	return s[:limit-1] + "..."
}
