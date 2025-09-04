package components

import (
	"maps"
	"strings"

	"github.com/a-h/templ"
	"go.chrisrx.dev/x/ptr"
	"go.chrisrx.dev/x/set"
	"go.chrisrx.dev/x/slices"
)

type Options struct {
	ID      string
	Classes []string
	Style   map[string]any
}

func (o Options) Apply(attrs templ.Attributes) {
	if o.ID != "" {
		WithAttrs("id", o.ID).Apply(attrs)
	}
	if len(o.Classes) > 0 {
		attrs["class"] = mergeAttr(getAttr(attrs, "class"), o.Classes...)
	}
	if len(o.Style) > 0 {
		maps.Copy(attrs, o.Style)
	}
}

func getAttr(attrs templ.Attributes, name string) string {
	attr, ok := attrs[name]
	if !ok {
		return ""
	}
	s, ok := attr.(string)
	if !ok {
		return ""
	}
	return s
}

type Option interface {
	Apply(templ.Attributes)
}

type OptionFunc func(templ.Attributes)

func (o OptionFunc) Apply(attrs templ.Attributes) {
	o(attrs)
}

func WithAttrs(pairs ...string) Option {
	if len(pairs)%2 != 0 {
		panic("odd attrs")
	}
	return OptionFunc(func(attrs templ.Attributes) {
		for i := range len(pairs) {
			if len(pairs)-1 > i {
				attr, ok := attrs[pairs[i]]
				if !ok {
					attrs[pairs[i]] = pairs[i+1]
					return
				}
				attrs[pairs[i]] = mergeAttr(attr.(string), pairs[i+1])
			}
		}
	})
}

func Class(classes ...string) Option {
	return OptionFunc(func(attrs templ.Attributes) {
		WithAttrs("class", strings.Join(classes, " ")).Apply(attrs)
	})
}

func mergeOptions[T any](s []T, elems ...T) []T {
	for _, elem := range elems {
		s = slices.Insert(s, 0, elem)
	}
	return s
}

func NewAttrs(opts []Option, defaults ...Option) templ.Attributes {
	attrs := make(templ.Attributes)
	for _, d := range defaults {
		d.Apply(attrs)
	}
	for _, o := range opts {
		o.Apply(attrs)
	}
	return attrs
}

func mergeAttr(v string, newAttrs ...string) string {
	return strings.Join(
		set.New(
			slices.DeleteFunc(strings.Split(v, " "), ptr.IsZero)...,
		).Union(
			set.New(slices.DeleteFunc(
				slices.FlatMap(newAttrs, func(attr string) []string {
					return strings.Split(attr, " ")
				}), ptr.IsZero)...),
		).List(),
		" ",
	)
}
