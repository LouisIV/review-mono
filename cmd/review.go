package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"review/internal/client"
	"review/internal/config"
	"review/internal/git"
	"review/internal/models"
)

func describeCmd(args []string, g globals) error {
	printOnly := false
	prompt := ""
	provider := ""

	for i := 0; i < len(args); i++ {
		if isHelpArg(args[i]) {
			describeUsage()

			return nil
		}

		if args[i] == "--print" {
			printOnly = true
		}

		if args[i] == "--prompt" && i+1 < len(args) {
			prompt = args[i+1]
			i++
		}

		if args[i] == "--provider" && i+1 < len(args) {
			provider = args[i+1]
			i++
		}
	}

	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	desc, err := c.GenerateDescription(repo.Path, session, prompt, provider)
	if err != nil {
		return err
	}

	if printOnly {
		fmt.Print(desc.Body)

		return nil
	}

	if g.json {
		printJSON(desc)
	} else {
		fmt.Printf(
			"Generated MR description (saved to .git/reviews/sessions/%s/description.md)\n\n%s",
			session.ID,
			desc.Body,
		)
	}

	return nil
}

func descriptionCmd(args []string, g globals) error {
	if len(args) == 0 {
		return errors.New("description subcommand required")
	}

	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	switch args[0] {
	case "show":
		desc, err := c.Description(repo.Path, session)
		if err != nil {
			return err
		}

		if g.json {
			printJSON(desc)
		} else {
			fmt.Print(desc.Body)
		}
	case "edit":
		return editDescription(repo.Path, session, c)
	default:
		return fmt.Errorf("unknown description subcommand %q", args[0])
	}

	return nil
}

func approveCmd(args []string, g globals, cfg config.Config) error {
	noBrowser, dryRun := false, false

	for _, arg := range args {
		if arg == "--no-browser" {
			noBrowser = true
		}

		if arg == "--dry-run" {
			dryRun = true
		}
	}

	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}

	if err := ensureDaemon(g.port); err != nil {
		return err
	}

	out, err := client.New(g.port).Approve(repo.Path, dryRun)
	if err != nil {
		return err
	}

	mrURL, _ := out["mr_url"].(string)
	if !dryRun && !noBrowser && cfg.OpenBrowserOnApprove && mrURL != "" {
		_ = openBrowser(mrURL)
	}

	if g.json {
		printJSON(out)
	} else {
		if dryRun {
			fmt.Println("Dry run: push skipped")
		} else {
			fmt.Printf("Pushed %s to origin\n", out["branch"])
		}

		if mrURL != "" {
			fmt.Println(mrURL)
		}
	}

	return nil
}

func requestChangesCmd(args []string, g globals) error {
	message := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--message" && i+1 < len(args) {
			message = args[i+1]
			i++
		}
	}

	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}

	if err := ensureDaemon(g.port); err != nil {
		return err
	}

	session, err := client.New(g.port).RequestChanges(repo.Path, message)
	if err != nil {
		return err
	}

	if g.json {
		printJSON(session)
	} else {
		fmt.Printf("Changes requested for %s\n", session.Branch)
	}

	return nil
}

func watchCmd(args []string, g globals) error {
	eventFilter := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--event" && i+1 < len(args) {
			eventFilter = args[i+1]
			i++
		}
	}

	err := ensureDaemon(g.port)
	if err != nil {
		return err
	}

	return client.New(g.port).Watch(context.Background(), "", func(event models.Event) bool {
		if eventFilter == "" || event.Event == eventFilter {
			b, _ := json.Marshal(event)
			fmt.Println(string(b))
		}

		return event.Event != "approved"
	})
}
