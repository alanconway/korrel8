// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Class(t *testing.T) {
	domain := mock.Domain("mock")
	rule := mock.NewRule(domain.Name(), domain.Class("foo"), domain.Class("bar"))
	for _, x := range []struct {
		name string
		want korrel8r.Class
		err  string
	}{
		{"mock:foo", domain.Class("foo"), ""},
		{"x.", nil, "invalid class name: x."},
		{".x", nil, "invalid class name: .x"},
		{"x", nil, "invalid class name: x"},
		{"", nil, "invalid class name: "},
		{"bad:foo", nil, `domain not found: "bad"`},
	} {
		t.Run(x.name, func(t *testing.T) {
			e, err := Build().Rules(rule).Engine()
			require.NoError(t, err)
			assert.Equal(t, []korrel8r.Domain{domain}, e.Domains())
			c, err := e.Class(x.name)
			if x.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, x.err)
			}
			assert.Equal(t, x.want, c)
		})
	}
}

func TestFollower_Traverse(t *testing.T) {
	d := mock.Domain("mock")
	s, err := mock.NewStore(d, nil)
	require.NoError(t, err)
	a, b, c, z := d.Class("a"), d.Class("b"), d.Class("c"), d.Class("z")
	e, err := Build().Rules(
		// Return 2 results, must follow both
		mock.NewApplyRule("ab", a, b, func(korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(b, 1, 2), nil
		}),
		// 2 rules, must follow both. Incorporate data from start object.
		mock.NewApplyRule("bc1", b, c, func(start korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(c, start), nil
		}),
		mock.NewApplyRule("bc2", b, c, func(start korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(c, start.(int)+10), nil
		}),
		mock.NewApplyRule("cz", c, z, func(start korrel8r.Object) (korrel8r.Query, error) {
			return mock.NewQuery(z, start), nil
		})).Stores(s).Engine()
	require.NoError(t, err)
	g := e.Graph()
	g.NodeFor(a).Result.Append(0)
	f := e.Follower(context.Background(), nil)
	_, err = g.Traverse(a, []korrel8r.Class{z}, f.Traverse)
	assert.NoError(t, err)
	// Check node results
	assert.ElementsMatch(t, []korrel8r.Object{0}, g.NodeFor(a).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2}, g.NodeFor(b).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(c).Result.List())
	assert.ElementsMatch(t, []korrel8r.Object{1, 2, 11, 12}, g.NodeFor(z).Result.List())
	// Check line results
	g.EachLine(func(l *graph.Line) {
		switch l.Rule.Name() {
		case "ab":
			q, err := l.Rule.Apply(0)
			require.NoError(t, err)
			assert.Equal(t, 2, l.Queries.Get(q))
			assert.Len(t, l.Queries, 1)
		case "bc1", "bc2":
			q1, err := l.Rule.Apply(1)
			require.NoError(t, err)
			q2, err := l.Rule.Apply(2)
			require.NoError(t, err)
			assert.Len(t, l.Queries, 2)
			assert.Equal(t, 1, l.Queries.Get(q1))
			assert.Equal(t, 1, l.Queries.Get(q2))
		case "cz":
			q1, err := l.Rule.Apply(1)
			require.NoError(t, err)
			q2, err := l.Rule.Apply(2)
			require.NoError(t, err)
			q3, err := l.Rule.Apply(11)
			require.NoError(t, err)
			q4, err := l.Rule.Apply(12)
			require.NoError(t, err)
			assert.Len(t, l.Queries, 4)
			assert.Equal(t, 1, l.Queries.Get(q1))
			assert.Equal(t, 1, l.Queries.Get(q2))
			assert.Equal(t, 1, l.Queries.Get(q3))
			assert.Equal(t, 1, l.Queries.Get(q4))
		default:
			t.Fatalf("unexpected rule: %v", l.Rule)
		}
	})
}

// Mock object has a name and a timestamp.
type obj struct {
	Name string
	Time time.Time
}

func (o obj) String() string { return o.Name }

func TestEngine_PropagateConstraints(t *testing.T) {
	d := mock.Domain("mock")
	a, b, c := d.Class("a"), d.Class("b"), d.Class("c")
	// Time range [start,end] and some time points.
	start := time.Now()
	end := start.Add(time.Minute)
	early, ontime, late := start.Add(-1), start.Add(1), end.Add(1)

	s, err := mock.NewStore(d, nil)
	require.NoError(t, err)
	s.ConstraintFunc = func(c *korrel8r.Constraint, o korrel8r.Object) bool { return c.CompareTime(o.(obj).Time) == 0 }

	// Rules to test constraints.
	e, err := Build().Rules(
		// Generate objects with timepoints.
		mock.NewQueryRule("ab", a, mock.NewQuery(b, obj{"x", early}, obj{"y", ontime}, obj{"z", late})),
		// Generate objects with timeponts and record name of previous object.
		mock.NewApplyRule("bc", b, c, func(o korrel8r.Object) (korrel8r.Query, error) {
			name := o.(obj).Name
			return mock.NewQuery(c, obj{"u" + name, early}, obj{"v" + name, ontime}, obj{"w" + name, late}), nil
		})).Stores(s).Engine()
	require.NoError(t, err)

	// Test traversals, verify constraints are applied.
	for _, x := range []struct {
		constraint *korrel8r.Constraint
		want       []obj
	}{
		{nil, []obj{
			{"ux", early}, {"vx", ontime}, {"wx", late},
			{"uy", early}, {"vy", ontime}, {"wy", late},
			{"uz", early}, {"vz", ontime}, {"wz", late},
		}},
		{&korrel8r.Constraint{}, []obj{
			{"ux", early}, {"vx", ontime}, {"wx", late},
			{"uy", early}, {"vy", ontime}, {"wy", late},
			{"uz", early}, {"vz", ontime}, {"wz", late},
		}},
		{&korrel8r.Constraint{Start: &start, End: &end}, []obj{
			{"vy", ontime},
		}},
		{&korrel8r.Constraint{Start: &start}, []obj{
			{"vy", ontime}, {"wy", late},
			{"vz", ontime}, {"wz", late},
		}},
		{&korrel8r.Constraint{End: &end}, []obj{
			{"ux", early}, {"vx", ontime},
			{"uy", early}, {"vy", ontime},
		}},
	} {
		t.Run(fmt.Sprintf("%+v", x.constraint), func(t *testing.T) {
			goals := []korrel8r.Class{c}
			g := e.Graph().AllPaths(a, goals...)
			g, err := e.GoalSearch(context.Background(), g, a, []korrel8r.Object{obj{"a", ontime}}, nil, x.constraint, goals)
			assert.NoError(t, err)
			got := g.NodeFor(c).Result.List()
			assert.Equal(t, asStrings(x.want), asStrings(got), "want %v got %v", x.want, got)
		})
	}
}

func asStrings[T any](v []T) []string {
	s := make([]string, len(v))
	for i := range v {
		s[i] = fmt.Sprint(v[i])
	}
	slices.Sort(s)
	return s
}
