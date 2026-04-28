package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"review/internal/models"
)

func commentCmd(args []string, g globals) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		commentUsage()

		return nil
	}

	switch args[0] {
	case "add":
		return commentAdd(args[1:], g)
	case subcommandList:
		return commentList(args[1:], g)
	case "resolve":
		return commentResolve(args[1:], g)
	case "delete":
		return commentDelete(args[1:], g)
	default:
		return fmt.Errorf("unknown comment subcommand %q", args[0])
	}
}

func commentAdd(args []string, g globals) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		commentAddUsage()

		return nil
	}

	author := models.AuthorHuman
	pos := []string{}

	for i := 0; i < len(args); i++ {
		if args[i] == "--author" && i+1 < len(args) {
			author = args[i+1]
			i++
		} else {
			pos = append(pos, args[i])
		}
	}

	if len(pos) < 2 {
		return errors.New("usage: review comment add file:line body")
	}

	file, line, err := parseLocation(pos[0])
	if err != nil {
		return err
	}

	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	comment, err := c.AddComment(
		repo.Path,
		session,
		models.Comment{File: file, Line: line, Body: strings.Join(pos[1:], " "), Author: author},
	)
	if err != nil {
		return err
	}

	if g.json {
		printJSON(comment)
	} else {
		fmt.Printf("Added comment %s on %s:%d\n", comment.ID, comment.File, comment.Line)
	}

	return nil
}

func commentList(args []string, g globals) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		commentListUsage()

		return nil
	}

	filters := url.Values{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--unresolved":
			filters.Set("resolved", "false")
		case "--file":
			if i+1 < len(args) {
				filters.Set("file", args[i+1])
				i++
			}
		case "--author":
			if i+1 < len(args) {
				filters.Set("author", args[i+1])
				i++
			}
		}
	}

	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	comments, err := c.Comments(repo.Path, session, filters)
	if err != nil {
		return err
	}

	if g.json {
		printJSON(map[string]any{"comments": comments})
	} else {
		for _, comment := range comments {
			state := "open"
			if comment.Resolved {
				state = "resolved"
			}

			fmt.Printf("%s %s %s:%d %s\n", comment.ID, state, comment.File, comment.Line, comment.Body)
		}
	}

	return nil
}

func commentResolve(args []string, g globals) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		commentResolveUsage()

		return nil
	}

	all := len(args) == 1 && args[0] == "--all"
	if len(args) == 0 {
		return errors.New("comment id or --all required")
	}

	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	ids := args

	if all {
		comments, err := c.Comments(repo.Path, session, url.Values{"resolved": []string{"false"}})
		if err != nil {
			return err
		}

		ids = nil
		for _, comment := range comments {
			ids = append(ids, comment.ID)
		}
	}

	for _, id := range ids {
		if _, err := c.PatchComment(repo.Path, session, id, map[string]bool{"resolved": true}); err != nil {
			return err
		}
	}

	if g.json {
		printJSON(map[string]any{"resolved": ids})
	} else {
		fmt.Printf("Resolved %d comment(s)\n", len(ids))
	}

	return nil
}

func commentDelete(args []string, g globals) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		commentDeleteUsage()

		return nil
	}

	if len(args) != 1 {
		return errors.New("comment id required")
	}

	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	if err := c.DeleteComment(repo.Path, session, args[0]); err != nil {
		return err
	}

	if g.json {
		printJSON(map[string]any{"deleted": true, "id": args[0]})
	} else {
		fmt.Printf("Deleted comment %s\n", args[0])
	}

	return nil
}

func commentUsage() {
	fmt.Println("review comment <subcommand> [flags]")
	fmt.Println("subcommands: add, list, resolve, delete")
	fmt.Println("run 'review comment <subcommand> --help' for details")
}

func commentAddUsage() {
	fmt.Println("review comment add [--author <author>] <file:line> <body>")
	fmt.Println("adds an inline comment to the current review session")
}

func commentListUsage() {
	fmt.Println("review comment list [--unresolved] [--file <path>] [--author <author>]")
	fmt.Println("lists comments in the current review session")
}

func commentResolveUsage() {
	fmt.Println("review comment resolve <comment-id>")
	fmt.Println("review comment resolve --all")
	fmt.Println("marks one or all unresolved comments as resolved")
}

func commentDeleteUsage() {
	fmt.Println("review comment delete <comment-id>")
	fmt.Println("deletes a comment from the current review session")
}
