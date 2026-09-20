// Command newpost creates a placeholder post in posts/, prefilling the
// frontmatter with the current time and marking it as a draft.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/goccy/go-yaml"
	"go.chrisrx.dev/x/strings"

	"github.com/ChrisRx/chrisrx-dev/components"
)

const postsDir = "posts"

func main() {
	now := time.Now().Truncate(time.Second)

	var (
		title  string
		create = true
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Article name").
				Placeholder("Neovim Nix module").
				Value(&title).
				Validate(func(s string) error {
					return validate(s, now)
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {
					return fmt.Sprintf("Create %s?", postPath(title, now))
				}, &title).
				Value(&create),
		),
	)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !create {
		return
	}

	path, err := writePost(title, now)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}

// postPath builds the date-prefixed path a post with the given title is
// written to, matching the names the generator expects in posts/.
func postPath(title string, date time.Time) string {
	return filepath.Join(postsDir, fmt.Sprintf("%s-%s.md", date.Format("2006-01-02"), strings.Slug(title)))
}

func validate(title string, date time.Time) error {
	if strings.Slug(title) == "" {
		return errors.New("article name must contain at least one letter or number")
	}
	switch _, err := os.Stat(postPath(title, date)); {
	case err == nil:
		return fmt.Errorf("%s already exists", postPath(title, date))
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	return nil
}

func writePost(title string, date time.Time) (string, error) {
	header, err := yaml.Marshal(components.Post{
		Title: title,
		Date:  date.UTC(),
		Draft: true,
	})
	if err != nil {
		return "", err
	}
	path := postPath(title, date)
	if err := os.WriteFile(path, []byte(fmt.Sprintf(strings.Dedent(`
		---
		%s
		---

		TODO
	`), header)), 0644); err != nil {
		return "", err
	}
	return path, nil
}
