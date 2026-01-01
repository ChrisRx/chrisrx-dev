package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/goccy/go-yaml"
	"go.chrisrx.dev/x/context"
	"go.chrisrx.dev/x/env"
	"go.chrisrx.dev/x/errors"
	"go.chrisrx.dev/x/must"
	"go.chrisrx.dev/x/slices"

	"github.com/ChrisRx/chrisrx-dev/pages"
)

var opts = env.MustParseFor[struct {
	Addr   string   `env:"ADDR" default:":8080" validate:"split_addr().port > 1024"`
	Dir    http.Dir `env:"DIR" default:""`
	Output bool     `env:"OUTPUT"`
}](env.RootPrefix("LOCAL_DEV"))

var UserKey = context.Key[string]()

func main() {
	ctx := context.Shutdown()

	posts := must.Ok(ReadPosts("posts/"))
	if opts.Output {
		if err := generate(ctx, posts); err != nil {
			log.Fatal(err)
		}
		return
	}

	s := &http.Server{
		Addr: opts.Addr,
		Handler: func() http.Handler {
			mux := http.NewServeMux()
			mux.Handle("/", templ.Handler(pages.Index()))
			mux.Handle("/blog.html", templ.Handler(pages.Blog(posts)))
			mux.Handle("/packages.html", templ.Handler(pages.Packages()))
			mux.Handle("/assets/", http.FileServer(opts.Dir))
			return mux
		}(),
		BaseContext: func(net.Listener) context.Context {
			return UserKey.WithValue(ctx, "ChrisRx")
		},
	}
	ctx.AddHandler(func() {
		fmt.Println("\rCTRL+C pressed, attempting graceful shutdown ...")
		if err := s.Shutdown(ctx); err != nil {
			panic(err)
		}
	})

	if err := errors.Ignore(s.ListenAndServe(), http.ErrServerClosed); err != nil {
		log.Fatal(err)
	}
}

func generate(ctx context.Context, posts []pages.Post) error {
	var b bytes.Buffer
	if err := pages.Index().Render(ctx, &b); err != nil {
		return err
	}
	if err := os.WriteFile("index.html", b.Bytes(), 0755); err != nil {
		return err
	}
	b.Reset()
	if err := pages.Blog(posts).Render(ctx, &b); err != nil {
		return err
	}
	if err := os.WriteFile("blog.html", b.Bytes(), 0755); err != nil {
		return err
	}
	b.Reset()
	if err := pages.Packages().Render(ctx, &b); err != nil {
		return err
	}
	if err := os.WriteFile("packages.html", b.Bytes(), 0755); err != nil {
		return err
	}
	return nil
}

func ReadPosts(path string) (posts []pages.Post, _ error) {
	if err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data = bytes.TrimPrefix(data, []byte("---\n"))
		parts := bytes.SplitN(data, []byte("---"), 2)
		if len(parts) != 2 {
			return fmt.Errorf("missing header")
		}

		var post pages.Post
		if err := yaml.Unmarshal(parts[0], &post); err != nil {
			return err
		}
		post.Content = string(parts[1])
		posts = append(posts, post)
		return nil

	}); err != nil {
		return nil, err
	}
	slices.SortFunc(posts, func(x, y pages.Post) int {
		switch {
		case x.Date.Equal(y.Date):
			return 0
		case x.Date.Before(y.Date):
			return 1
		default:
			return -1
		}
	})
	return posts, nil
}
