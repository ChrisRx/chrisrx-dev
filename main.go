package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"go.chrisrx.dev/x/context"
	"go.chrisrx.dev/x/env"
	"go.chrisrx.dev/x/errors"
	"go.chrisrx.dev/x/strings"

	"github.com/ChrisRx/chrisrx-dev/pages"
)

var opts = env.MustParseFor[struct {
	Addr   string   `env:"ADDR" default:":8080" validate:"split_addr().port > 1024"`
	Dir    http.Dir `env:"DIR" default:""`
	Output bool     `env:"OUTPUT"`
}](env.RootPrefix("LOCAL_DEV"))

var UserKey = context.Key[string]()

func main() {
	if opts.Output {
		if err := generate(); err != nil {
			log.Fatal(err)
		}
		return
	}

	ctx := context.Shutdown()
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

var posts = []pages.Post{
	{
		Title:     "First entry and trying out htmx",
		Published: time.Date(2025, 12, 31, 11, 30, 0, 0, time.Local),
		Content: strings.Dedent(`
			Switched from using alpine.js to using htmx and changed the Netscape
			browser buttons to navigate to different pages. Seems pretty neat so far!
		`),
	},
}

func generate() error {
	var b bytes.Buffer
	if err := pages.Index().Render(context.Background(), &b); err != nil {
		return err
	}
	if err := os.WriteFile("index.html", b.Bytes(), 0755); err != nil {
		return err
	}
	b.Reset()
	if err := pages.Blog(posts).Render(context.Background(), &b); err != nil {
		return err
	}
	if err := os.WriteFile("blog.html", b.Bytes(), 0755); err != nil {
		return err
	}
	b.Reset()
	if err := pages.Packages().Render(context.Background(), &b); err != nil {
		return err
	}
	if err := os.WriteFile("packages.html", b.Bytes(), 0755); err != nil {
		return err
	}
	return nil
}
