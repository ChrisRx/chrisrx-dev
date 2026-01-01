---
title: Blog supports markdown
date: 2026-01-01T09:30:00Z
---

Blog posts are now being parsed out of files and the content supports markdown. I'm using the [goldmark](https://github.com/yuin/goldmark) package to render the content as a `templ.Component`:

```go
type Post struct {
	Title     string
	Date      time.Time
	Content   string
}

func (p Post) Render() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
        return goldmark.Convert([]byte(p.Content), w)
	})
}
```
