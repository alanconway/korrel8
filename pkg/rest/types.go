// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"cmp"
	"encoding/json"
	"slices"

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// @description Domain configuration information.
type Domain struct {
	Name   string                 `json:"name"`
	Stores []korrel8r.StoreConfig `json:"stores,omitempty"`
	Errors []string               `json:"errors,omitempty"`
}

// @description Classes maps class names to a short description.
type Classes map[string]string

// Note: use json.RawMessage for and objects because we don't know the type of these values
// until the engine resolves the class name as a Class value.

// @description	Starting point for correlation.
// The starting object set includes:
// - results from getting each of the [Start.Queries]
// - unmarshalled objects from [Start.Objects]
type Start struct {
	// Class of starting objects
	Class string `json:"class,omitempty"`
	// Queries for starting objects, must return the start class.
	Queries []string `json:"queries,omitempty"`
	// Objects serialized as JSON to, must be of start class.
	Objects []json.RawMessage `json:"objects,omitempty" swaggertype:"object"`
	// Constraint (optional) to limit the results.
	Constraint *korrel8r.Constraint `json:"constraint,omitempty"`
}

// @description	Starting point for a goals search.
type GoalsRequest struct {
	// Start of correlation search.
	Start Start `json:"start"`
	// Goal classes for correlation.
	Goals []string `json:"goals,omitempty" example:"domain:class"`
}

// @description	Starting point for a neighbours search.
type NeighboursRequest struct {
	// Start of correlation search.
	Start Start `json:"start"`
	// Max depth of neighbours graph.
	Depth int `json:"depth"`
}

// GraphOptions control the format of the graph
type GraphOptions struct {
	// WithRules if true include rules in the graph edges.
	WithRules bool `form:"withRules"`
}

// QueryOptions provides a query to execute.
type QueryOptions struct {
	// Query string to execute.
	Query string `form:"query"`
	// Constraint (optional) to limit the results.
	Constraint *korrel8r.Constraint `json:"constraint,omitempty"`
}

// @description Query run during a correlation with a count of results found.
type QueryCount struct {
	// Query for correlation data.
	Query string `json:"query"`
	// Count of results or -1 if the query was not executed.
	Count int `json:"count"`
}

// Rule is a correlation rule with a list of queries and results counts found during navigation.
// Rules form a directed multi-graph over classes in the result graph.
type Rule struct {
	// Name is an optional descriptive name.
	Name string `json:"name,omitempty"`
	// Queries generated while following this rule.
	Queries []QueryCount `json:"queries,omitempty"`
}

// Node in the result graph, contains results for a single class.
type Node struct {
	// Class is the full class name in "DOMAIN:CLASS" form.
	Class string `json:"class" example:"domain:class"`
	// Queries yielding results for this class.
	Queries []QueryCount `json:"queries,omitempty"`
	// Count of results found for this class, after de-duplication.
	Count int `json:"count"`
}

// Directed edge in the result graph, from Start to Goal classes.
type Edge struct {
	// Start is the class name of the start node.
	Start string `json:"start"`
	// Goal is the class name of the goal node.
	Goal string `json:"goal" example:"domain:class"`
	// Rules is the set of rules followed along this edge (optional).
	Rules []Rule `json:"rules,omitempty"`
}

// @description	Graph resulting from a correlation search.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges,omitempty"`
}

func queryCounts(gq graph.Queries) (qc []QueryCount) {
	for q, c := range gq {
		qc = append(qc, QueryCount{Query: q, Count: c})
	}
	slices.SortFunc(qc, func(a, b QueryCount) int {
		if n := cmp.Compare(a.Count, b.Count); n != 0 {
			return -n
		}
		return cmp.Compare(a.Query, b.Query)
	})
	return qc
}

func rule(l *graph.Line) (r Rule) {
	r.Name = l.Rule.Name()
	r.Queries = queryCounts(l.Queries)
	return r
}

func node(n *graph.Node) Node {
	return Node{
		Class:   korrel8r.ClassName(n.Class),
		Queries: queryCounts(n.Queries),
		Count:   len(n.Result.List()),
	}
}

func nodes(g *graph.Graph) []Node {
	nodes := []Node{} // Want [] not null for empty in JSON.
	g.EachNode(func(n *graph.Node) {
		if !n.Empty() { // Skip empty nodes
			nodes = append(nodes, node(n))
		}
	})
	return nodes
}

func edge(e *graph.Edge, withRules bool) Edge {
	edge := Edge{
		Start: korrel8r.ClassName(e.Start().Class),
		Goal:  korrel8r.ClassName(e.Goal().Class),
	}
	if withRules {
		e.EachLine(func(l *graph.Line) {
			if l.Queries.Total() != 0 {
				edge.Rules = append(edge.Rules, rule(l))
			}
		})
	}
	return edge
}

func edges(g *graph.Graph, opts *GraphOptions) (edges []Edge) {
	g.EachEdge(func(e *graph.Edge) {
		if !e.Goal().Empty() { // Skip edges that lead to an empty node.
			edges = append(edges, edge(e, opts.WithRules))
		}
	})
	return edges
}
