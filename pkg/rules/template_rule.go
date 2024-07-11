// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package rules uses templates to generate goal queries from start objects.
package rules

import (
	"text/template"

	"bytes"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

// NewTemplateRule returns a korrel8r.Rule that uses Go templates to transform objects to queries.
func NewTemplateRule(start, goal []korrel8r.Class, query *template.Template) korrel8r.Rule {
	return &templateRule{start: start, goal: goal, query: query}
}

var _ = impl.AssertRule(&templateRule{})

type templateRule struct {
	query       *template.Template
	start, goal []korrel8r.Class
}

func (r *templateRule) Name() string            { return r.query.Name() }
func (r *templateRule) String() string          { return r.Name() }
func (r *templateRule) Start() []korrel8r.Class { return r.start }
func (r *templateRule) Goal() []korrel8r.Class  { return r.goal }

// Apply the rule by applying the template.
func (r *templateRule) Apply(start korrel8r.Object) (korrel8r.Query, error) {
	b := &bytes.Buffer{}
	err := r.query.Execute(b, start)
	if err != nil {
		return nil, err
	}
	return r.Goal()[0].Domain().Query(b.String())
}
