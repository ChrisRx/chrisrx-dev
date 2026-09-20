package main

import (
	"bytes"
	"cmp"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/a-h/templ"
	"github.com/goccy/go-yaml"
	"go.chrisrx.dev/x/context"
	"go.chrisrx.dev/x/env"
	"go.chrisrx.dev/x/errors"
	"go.chrisrx.dev/x/must"
	"go.chrisrx.dev/x/slices"
	"go.chrisrx.dev/x/strings"

	"github.com/ChrisRx/chrisrx-dev/components"
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
	defer ctx.Close()

	posts := must.Ok(ReadPosts("posts/"))
	if err := generate(ctx, posts); err != nil {
		log.Fatal(err)
	}
	if opts.Output {
		return
	}

	s := &http.Server{
		Addr: opts.Addr,
		Handler: func() http.Handler {
			mux := http.NewServeMux()
			mux.Handle("/{$}", templ.Handler(pages.Index(slices.Truncate(posts, 5))))
			mux.Handle("/assets/", http.FileServer(opts.Dir))
			mux.Handle("/archive/{$}", templ.Handler(pages.BlogArchive(posts...)))
			mux.Handle("/blog/{$}", templ.Handler(pages.Blog(posts...)))
			mux.Handle("/blog/", http.FileServer(opts.Dir))
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

const staticDir = "docs"

func generate(ctx context.Context, posts []components.Post) error {
	var b bytes.Buffer
	if err := pages.Index(slices.Truncate(posts, 5)).Render(ctx, &b); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(staticDir, "index.html"), b.Bytes()); err != nil {
		return err
	}
	b.Reset()

	for _, post := range posts {
		dir := filepath.Join(staticDir, "blog", post.Date.Format("2006"), strings.Slug(post.Title))
		if err := os.MkdirAll(dir, 0755); err != nil && err != os.ErrExist {
			return fmt.Errorf("failed to create dir %q: %v", dir, err)
		}

		var b bytes.Buffer
		if err := pages.Blog(post).Render(ctx, &b); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, "index.html"), b.Bytes()); err != nil {
			return err
		}
	}

	if err := pages.Blog(posts...).Render(ctx, &b); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(staticDir, "blog/index.html"), b.Bytes()); err != nil {
		return err
	}
	b.Reset()
	if err := pages.BlogArchive(posts...).Render(ctx, &b); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(staticDir, "archive/index.html"), b.Bytes()); err != nil {
		return err
	}
	modules := []struct {
		Name, Repo string
	}{
		{Name: "ptr", Repo: "ptr-go"},
		{Name: "x", Repo: "exp"},
		{Name: "leaselock", Repo: "leaselock"},
		{Name: "log", Repo: "log-go"},
		{Name: "group", Repo: "group-go"},
		{Name: "quake-kube", Repo: "quake-kube"},
		{Name: "result", Repo: "result-go"},
		{Name: "run", Repo: "run-go"},
		{Name: "tools", Repo: "tools-go"},
		{Name: "webos", Repo: "webos"},
	}
	for _, m := range modules {
		b.Reset()
		if err := redirectTemplate.Execute(&b, map[string]string{
			"Name": fmt.Sprintf("go.chrisrx.dev/%s", m.Name),
			"Repo": fmt.Sprintf("https://github.com/ChrisRx/%s", m.Repo),
		}); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(staticDir, filepath.Base(m.Name)), b.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && err != os.ErrExist {
		return fmt.Errorf("failed to create dir %q: %v", filepath.Dir(path), err)
	}
	return os.WriteFile(path, data, 0644)
}

var redirectTemplate = template.Must(template.New("").Parse(`<html>
  <head>
    <meta name="go-import" content="{{ .Name }} git {{ .Repo }}">
    <meta name="go-source" content="{{ .Name }} {{ .Repo }} {{ .Repo }}/tree/main{/dir} {{ .Repo }}/tree/main{/dir}/{file}#L{line}">
    <meta name="robots" content="noindex">
    <meta http-equiv="refresh" content="0; url=https://pkg.go.dev/{{ .Name }}">
  </head>
  <body>
    Redirecting to <a href="https://pkg.go.dev/{{ .Name }}">pkg.go.dev/{{ .Name }}</a>.
  </body>
</html>
`))

func ReadPosts(path string) (posts []components.Post, _ error) {
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

		var post components.Post
		if err := yaml.Unmarshal(parts[0], &post); err != nil {
			return err
		}
		post.Content = string(parts[1])
		if !post.Draft {
			posts = append(posts, post)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	slices.SortFunc(posts, func(x, y components.Post) int {
		return -cmp.Compare(x.Date.Format(time.RFC3339), y.Date.Format(time.RFC3339))
	})
	return posts, nil
}
