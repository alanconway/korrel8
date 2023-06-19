// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package templaterule

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// rule makes a mock rule
var mockRule = func(s, g korrel8r.Class) mock.Rule { return mock.NewRule("", s, g) }

// mockRules copies public parts of korrel8r.Rule to a mock.Rule for easy comparison.
func mockRules(k ...korrel8r.Rule) []mock.Rule {
	m := make([]mock.Rule, len(k))
	for i := range k {
		m[i] = mockRule(k[i].Start(), k[i].Goal())
	}
	return m
}

func TestRule_Rules(t *testing.T) {
	foo := mock.NewDomainWithClasses("foo", "a", "b", "c")
	bar := mock.NewDomainWithClasses("bar", "x", "y", "z")
	a, b, c := foo.Class("a"), foo.Class("b"), foo.Class("c")
	_, _, z := bar.Class("x"), bar.Class("y"), bar.Class("z")
	for _, x := range []struct {
		rule string
		want []mock.Rule
	}{
		{
			rule: `
name:   "simple"
start:  {domain: "foo", classes: [a]}
goal:   {domain: "bar", classes: [z]}
result: {query: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule(a, z)},
		},
		{
			rule: `
name: "multistart"
start: {domain: foo, classes: [a, b, c]}
goal:  {domain: bar, classes: [z]}
result: {query: dummy, class: dummy}
`,
			want: []mock.Rule{mockRule(a, z), mockRule(b, z), mockRule(c, z)},
		},
		{
			rule: `
name: "all-all"
start: {domain: foo}
goal: {domain: bar}
result: {query: dummy, class: dummy}
`,
			want: func() []mock.Rule {
				var rules []mock.Rule
				for _, foo := range foo.Classes() {
					for _, bar := range bar.Classes() {
						rules = append(rules, mockRule(foo, bar))
					}
				}
				return rules
			}(),
		},
	} {
		t.Run(x.rule, func(t *testing.T) {
			f := NewFactory(map[string]korrel8r.Domain{"foo": foo, "bar": bar}, nil)
			var r api.Rule
			require.NoError(t, yaml.Unmarshal([]byte(x.rule), &r))
			got, err := f.Rules(r)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.want, mockRules(got...))
			}
		})
	}
}
