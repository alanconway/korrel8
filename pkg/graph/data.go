// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/multi"
)

// Data contains the class nodes and rule lines for rule/class graphs.
// All graphs based on the same Data have stable, consistent node and line IDs.
// Note new rules can be added to a Data instance but never removed.
type Data struct {
	Nodes  []*Node          // Nodes slice index == Node.ID()
	Lines  []*Line          // Lines slice index == Line.ID()
	nodeID map[string]int64 // Map by full class name
}

func NewData(rules ...korrel8r.Rule) *Data {
	d := Data{nodeID: make(map[string]int64)}
	for _, r := range rules {
		d.AddRule(r)
	}
	return &d
}

func (d *Data) AddRule(r korrel8r.Rule) {
	for _, start := range r.Start() {
		for _, goal := range r.Goal() {
			id := int64(len(d.Lines))
			l := &Line{
				Line:    multi.Line{F: d.NodeFor(start), T: d.NodeFor(goal), UID: id},
				Rule:    r,
				Attrs:   Attrs{},
				Queries: Queries{},
			}
			d.Lines = append(d.Lines, l)
		}
	}
}

// NodeFor returns the Node for class c, creating it if necessary.
func (d *Data) NodeFor(c korrel8r.Class) *Node {
	cname := c.String()
	if id, ok := d.nodeID[cname]; ok {
		return d.Nodes[id]
	}
	id := int64(len(d.Nodes))
	n := &Node{
		Node:    multi.Node(id),
		Class:   c,
		Attrs:   Attrs{},
		Result:  korrel8r.NewResult(c),
		Queries: Queries{},
	}
	d.Nodes = append(d.Nodes, n)
	d.nodeID[cname] = id
	return n
}

// EmptyGraph returns a new emptpy graph based on this Data.
func (d *Data) EmptyGraph() *Graph { return New(d) }

// NewGraph returns a new graph of all the Data.
func (d *Data) NewGraph() *Graph {
	g := New(d)
	for _, l := range d.Lines {
		g.SetLine(l)
	}
	for _, n := range d.Nodes {
		if nn := g.Node(n.ID()); nn == nil {
			g.AddNode(n)
		} else if nn != n {
			panic(fmt.Errorf("invalid node %v, already have %v", n, nn))
		}
	}
	return g
}

func (d *Data) Rules() []korrel8r.Rule {
	var rules []korrel8r.Rule
	for _, l := range d.Lines {
		rules = append(rules, l.Rule)
	}
	return rules
}

func (d *Data) Classes() []korrel8r.Class {
	var classs []korrel8r.Class
	for _, n := range d.Nodes {
		classs = append(classs, n.Class)
	}
	return classs
}

// Node is a graph Node, corresponds to a Class.
type Node struct {
	multi.Node
	Attrs   // GraphViz Attributer
	Class   korrel8r.Class
	Result  korrel8r.Result // Accumulate incoming query results.
	Queries Queries         // All queries leading to this node.
}

func NodeFor(n graph.Node) *Node           { return n.(*Node) }
func ClassFor(n graph.Node) korrel8r.Class { return NodeFor(n).Class }

func (n *Node) String() string { return n.Class.String() }
func (n *Node) DOTID() string  { return n.Class.String() }
func (n *Node) Empty() bool    { return len(n.Result.List()) == 0 }

// QueryCount records count of objects resulting from a query.
// Count == -1 means the query has not been evaluated.
type QueryCount struct {
	Query korrel8r.Query
	Count int
}

// Queries is a map of QueryCount by Query name.
type Queries map[string]QueryCount

func (qs Queries) Has(q korrel8r.Query) bool   { _, ok := qs[q.String()]; return ok }
func (qs Queries) Set(q korrel8r.Query, n int) { qs[q.String()] = QueryCount{q, n} }
func (qs Queries) Get(q korrel8r.Query) int {
	if qc, ok := qs[q.String()]; ok {
		return qc.Count
	}
	return -1
}

// Total of the counts
func (qs Queries) Total() (total int) {
	for _, qc := range qs {
		total += qc.Count
	}
	return total
}

// Line is one line in a multi-graph edge, corresponds to a rule.
type Line struct {
	multi.Line
	Attrs   // GraphViz Attributer
	Rule    korrel8r.Rule
	Queries Queries // Queries generated by Rule
}

func (l *Line) String() string           { return fmt.Sprint(l.Rule) }
func (l *Line) DOTID() string            { return l.Rule.Name() }
func LineFor(l graph.Line) *Line         { return l.(*Line) }
func RuleFor(l graph.Line) korrel8r.Rule { return LineFor(l).Rule }

type Edge multi.Edge

func (e *Edge) Start() *Node { return e.F.(*Node) }
func (e *Edge) Goal() *Node  { return e.T.(*Node) }
func (e *Edge) EachLine(visit func(*Line)) {
	lines := e.Lines
	for lines.Next() {
		visit(lines.Line().(*Line))
	}
}
