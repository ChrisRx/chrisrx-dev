package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"go.chrisrx.dev/x/context"
	"go.chrisrx.dev/x/env"
	"go.chrisrx.dev/x/errors"

	"github.com/ChrisRx/chrisrx-dev/pages"
)

var opts = env.MustParseFor[struct {
	Addr   string   `env:"ADDR" default:":8080" validate:"split_addr().port > 1024"`
	Dir    http.Dir `env:"DIR" default:""`
	Output string   `env:"OUTPUT"`
}](env.RootPrefix("LOCAL_DEV"))

var UserKey = context.Key[string]()

func main() {
	if opts.Output != "" {
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

func generate() error {
	var b bytes.Buffer
	if err := pages.Index().Render(context.Background(), &b); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opts.Output), 0755); err != nil {
		return err
	}
	return os.WriteFile(opts.Output, b.Bytes(), 0755)
}
