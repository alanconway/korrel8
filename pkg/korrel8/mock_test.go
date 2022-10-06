package korrel8

import (
	"context"
	"fmt"
	"strings"
)

// Mock implementations

type mockDomain struct{}

func (d mockDomain) String() string          { return "mock" }
func (d mockDomain) Class(name string) Class { return mockClass(name) }
func (d mockDomain) KnownClasses() []Class   { return nil }

var _ Domain = mockDomain{} // Implements interface

type mockClass string

func (c mockClass) Domain() Domain { return mockDomain{} }
func (c mockClass) New() Object    { return mockObject{} }
func (c mockClass) String() string { return string(c) }

var _ Class = mockClass("") // Implements interface

type mockObject struct {
	name  string
	class mockClass
}

func o(name, class string) Object           { return mockObject{name: name, class: mockClass(class)} }
func (o mockObject) Native() any            { return o }
func (o mockObject) Identifier() Identifier { return o.name }

var _ Object = mockObject{} // Implements interface

type mockRule struct {
	start, goal Class
	apply       func(Object, *Constraint) []string
}

func (r mockRule) Start() Class   { return r.start }
func (r mockRule) Goal() Class    { return r.goal }
func (r mockRule) String() string { return fmt.Sprintf("(%v)->%v", r.start, r.goal) }
func (r mockRule) Apply(start Object, c *Constraint) ([]string, error) {
	return r.apply(start, c), nil
}

var _ Rule = mockRule{} // Implements interface

func rr(start, goal string, apply func(Object, *Constraint) []string) mockRule {
	return mockRule{
		start: mockClass(start),
		goal:  mockClass(goal),
		apply: apply,
	}
}

func r(start, goal string) mockRule { return rr(start, goal, nil) }

type mockStore struct{}

// Query a mock "query" is a comma-separated list of "name.class" to be turned into mock objects.
func (s mockStore) Get(_ context.Context, q string, r Result) error {
	for _, s := range strings.Split(q, ",") {
		nc := strings.Split(s, ".")
		r.Append(mockObject{name: nc[0], class: mockClass(nc[1])})
	}
	return nil
}

var _ Store = mockStore{} // Implements interface