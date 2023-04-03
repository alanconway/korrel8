package graph

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTraverse(t *testing.T) {
	for _, x := range []struct {
		name  string
		graph []rule
		want  [][]rule
	}{
		{
			name:  "multipath",
			graph: []rule{{1, 11}, {1, 12}, {11, 99}, {12, 99}},
			want:  [][]rule{{{1, 11}, {1, 12}}, {{11, 99}, {12, 99}}},
		},
		{
			name:  "simple",
			graph: []rule{{1, 2}, {2, 3}, {3, 4}, {4, 5}},
			want:  [][]rule{{{1, 2}}, {{2, 3}}, {{3, 4}}, {{4, 5}}},
		},
		{
			name:  "cycle", // cycle of 2,3,4
			graph: []rule{{1, 2}, {2, 3}, {3, 4}, {4, 2}, {4, 5}},
			want:  [][]rule{{{1, 2}}, {{2, 3}, {3, 4}, {4, 2}}, {{4, 5}}},
		},
	} {
		t.Run(x.name, func(t *testing.T) {
			g := testGraph(x.graph)
			var got []rule
			err := g.Traverse(func(l *Line) { got = append(got, RuleFor(l).(rule)) })
			assert.NoError(t, err)
			assertComponentOrder(t, x.want, got)
		})
	}
}

func TestNeighbours(t *testing.T) {
	g := testGraph([]rule{{1, 11}, {1, 12}, {1, 13}, {11, 22}, {12, 22}, {12, 13}, {22, 99}})
	for _, x := range []struct {
		depth int
		want  [][]rule
	}{
		{
			depth: 1,
			want:  [][]rule{{{1, 11}, {1, 12}, {1, 13}}},
		},
		{
			depth: 2,
			want:  [][]rule{{{1, 11}, {1, 12}, {1, 13}}, {{11, 22}, {12, 22}, {12, 13}}},
		},
		{
			depth: 3,
			want:  [][]rule{{{1, 11}, {1, 12}, {1, 13}}, {{11, 22}, {12, 22}, {12, 13}}, {{22, 99}}},
		},
	} {
		t.Run(fmt.Sprintf("depth=%v", x.depth), func(t *testing.T) {
			var got []rule
			g.Neighbours(class(1), x.depth, func(l *Line) { got = append(got, RuleFor(l).(rule)) })
			assertComponentOrder(t, x.want, got)
		})
	}
}
