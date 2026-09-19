package extensions

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/renderer"
	"github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/util"
)

type ImageTemplateFunc func(id, src, alt string) templ.Component

// EnlargeableImageRenderer wraps standard markdown images in an <a> tag for a
// lightbox effect.
type EnlargeableImageRenderer struct {
	html.Config
	n atomic.Int64

	fn func(id, src, alt string) templ.Component
}

func NewEnlargeableImageRenderer(fn ImageTemplateFunc, opts ...html.Option) html.Extension {
	r := &EnlargeableImageRenderer{
		fn: fn,
	}
	for _, opt := range opts {
		opt.SetFormatOption(&r.Config)
	}
	return r
}

func (r *EnlargeableImageRenderer) RendererOptions(_ *html.Config) []html.Option {
	return []html.Option{
		html.WithNodeRendererDecorator(ast.KindImage, func(next html.NodeRenderer) html.NodeRenderer {
			return html.NodeRendererFunc(func(w io.Writer, source []byte, node ast.Node, entering bool, rc renderer.Context) (ast.WalkStatus, error) {
				switch n := node.(type) {
				case *ast.Image:
					return r.render(w, source, n, entering, next, rc)
				default:
					return next.Render(w, source, n, entering, rc)
				}
			})
		}),
	}
}

func (r *EnlargeableImageRenderer) render(w io.Writer, source []byte, node ast.Node, entering bool, next html.NodeRenderer, rc renderer.Context) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	bw := w.(util.BufWriter)

	n := node.(*ast.Image)
	m := r.fn(fmt.Sprintf("md-image-%d", r.n.Add(1)), n.Destination.Str(source), altText(n, source))
	if err := m.Render(context.TODO(), bw); err != nil {
		return ast.WalkStop, err
	}
	// The alt text lives in the image's children, which the modal has already
	// consumed. Continuing would render them again as body text.
	return ast.WalkSkipChildren, nil
}

// altText flattens the text nodes beneath an image, which is where goldmark
// keeps the markdown alt text.
func altText(node ast.Node, source []byte) string {
	var sb strings.Builder
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			sb.WriteString(t.Value.Str(source))
		} else {
			sb.WriteString(altText(c, source))
		}
	}
	return sb.String()
}
