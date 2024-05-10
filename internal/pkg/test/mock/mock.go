// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package mock is a mock implementation of a korrel8r domain for testing.
// All the mock types (Domain, Class, Object etc.) are simply integers that implement the korrel8r interfaces.
// This simplifies tests since they can be initialized from int constants.
package mock

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"golang.org/x/exp/slices"
)

var (
	// Validate implementation of interfaces.
	_ korrel8r.Domain = Domain("")
	_ korrel8r.Class  = Domain("").Class("")
	_ korrel8r.Query  = Query{}
	_ korrel8r.Rule   = &Rule{}
	_ korrel8r.Store  = &Store{}
)

type Domain string

func (d Domain) Name() string                        { return string(d) }
func (d Domain) String() string                      { return d.Name() }
func (d Domain) Description() string                 { return "Mock domain." }
func (d Domain) Class(name string) korrel8r.Class    { return Class{name: name, domain: d} }
func (d Domain) Classes() (classes []korrel8r.Class) { return nil }
func (d Domain) Store(s any) (korrel8r.Store, error) { return NewStore(d, s) }
func (d Domain) Query(s string) (korrel8r.Query, error) {
	var (
		q   Query
		err error
	)
	q.class, err = impl.UnmarshalQueryString(d, s, &q.Results)
	return q, err
}

func Domains(names ...string) []korrel8r.Domain {
	var domains []korrel8r.Domain
	for _, name := range names {
		domains = append(domains, Domain(name))
	}
	return domains
}

type DomainWithClasses struct {
	Domain
	MClasses []korrel8r.Class
}

func NewDomainWithClasses(name string, classes ...string) *DomainWithClasses {
	d := &DomainWithClasses{Domain: Domain(name)}
	for _, c := range classes {
		d.MClasses = append(d.MClasses, Class{name: c, domain: d})
	}
	return d
}

func (d DomainWithClasses) Class(name string) korrel8r.Class {
	i := slices.IndexFunc(d.MClasses, func(c korrel8r.Class) bool { return c.Name() == name })
	if i < 0 {
		return nil
	}
	return d.MClasses[i]
}

func (d DomainWithClasses) Classes() []korrel8r.Class { return d.MClasses }

type Class struct {
	name   string
	domain korrel8r.Domain
}

func (c Class) Domain() korrel8r.Domain  { return c.domain }
func (c Class) String() string           { return impl.ClassString(c) }
func (c Class) Name() string             { return c.name }
func (c Class) Description() string      { return fmt.Sprintf("mock class %v", c.String()) }
func (c Class) ID(o korrel8r.Object) any { return o }
func (c Class) New() korrel8r.Object     { return "" }

type Rule struct {
	name        string
	start, goal []korrel8r.Class
}

func NewRule(name string, start, goal korrel8r.Class) *Rule {
	return &Rule{name: name, start: []korrel8r.Class{start}, goal: []korrel8r.Class{goal}}
}

func NewRuleMulti(name string, start, goal []korrel8r.Class) *Rule {
	return &Rule{name: name, start: start, goal: goal}
}

func NewRules(rules ...korrel8r.Rule) (mocks []*Rule) {
	for _, r := range rules {
		mocks = append(mocks, NewRuleMulti(r.Name(), r.Start(), r.Goal()))
	}
	return mocks
}

func (r *Rule) Start() []korrel8r.Class { return r.start }
func (r *Rule) Goal() []korrel8r.Class  { return r.goal }
func (r *Rule) Name() string            { return r.name }
func (r *Rule) Apply(start korrel8r.Object) (korrel8r.Query, error) {
	panic("not implemented") // See ApplyRule
}

// RuleLess orders rules.
func RuleLess(a, b korrel8r.Rule) int {
	if a.Start()[0].Name() != b.Start()[0].Name() {
		return strings.Compare(a.Start()[0].Name(), b.Start()[0].Name())
	}
	return strings.Compare(a.Goal()[0].Name(), b.Goal()[0].Name())
}

// SorRules  sorts rules by (start, goal) order.
func SortRules(rules []korrel8r.Rule) []korrel8r.Rule { slices.SortFunc(rules, RuleLess); return rules }

type ApplyFunc func(korrel8r.Object) (korrel8r.Query, error)
type ApplyRule struct {
	*Rule
	apply ApplyFunc
}

func NewApplyRule(name string, start, goal korrel8r.Class, apply ApplyFunc) *ApplyRule {
	return &ApplyRule{Rule: NewRule(name, start, goal), apply: apply}
}

func NewApplyRuleMulti(name string, start, goal []korrel8r.Class, apply ApplyFunc) *ApplyRule {
	return &ApplyRule{Rule: NewRuleMulti(name, start, goal), apply: apply}
}

func NewQueryRule(name string, start korrel8r.Class, query korrel8r.Query) *ApplyRule {
	return NewApplyRule(name, start, query.Class(), func(korrel8r.Object) (korrel8r.Query, error) {
		return query, nil
	})
}

func (r ApplyRule) Apply(start korrel8r.Object) (korrel8r.Query, error) {
	return r.apply(start)
}

// Store is a mock store, use with [Query]
type Store struct {
	domain korrel8r.Domain
	// Optional StoreConfig
	StoreConfig config.Store
	// Optional constraint testing function: return true if object is accepted.
	ConstraintFunc func(*korrel8r.Constraint, korrel8r.Object) bool
}

func NewStore(d korrel8r.Domain, s any) (Store, error) {
	sc, err := impl.TypeAssert[config.Store](s)
	if s != nil && err != nil {
		return Store{}, err
	}
	return Store{domain: d, StoreConfig: sc}, nil
}

func (s Store) Domain() korrel8r.Domain { return s.domain }

func (s Store) Get(ctx context.Context, q korrel8r.Query, constraint *korrel8r.Constraint, r korrel8r.Appender) error {
	results := slices.Clone(q.(Query).Results)
	if constraint != nil && s.ConstraintFunc != nil {
		results = slices.DeleteFunc(results, func(o korrel8r.Object) bool { return !s.ConstraintFunc(constraint, o) })
	}
	r.Append(results...)
	return nil
}

func (s *Store) Resolve(korrel8r.Query) *url.URL { panic("not implemented") }

// Query is a mock query that contains the desired results.
type Query struct {
	Results []korrel8r.Object
	class   korrel8r.Class
}

func NewQuery(c korrel8r.Class, results ...korrel8r.Object) korrel8r.Query {
	return Query{class: c, Results: results}
}
func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Data() string          { return impl.JSONString(q.Results) }
func (q Query) String() string        { return impl.QueryString(q) }

// Timestamper interface for objects with a Timestamp() method.
type Timestamper interface{ Timestamp() time.Time }
