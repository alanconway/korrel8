// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique

// List of unique comparable values, maintains order.
type List[T any] struct {
	seen map[any]struct{}
	List []T
}

func NewList[T any](values ...T) List[T] {
	l := List[T]{seen: map[any]struct{}{}}
	l.Append(values...)
	return l
}

// Add a value if not already present, return true if the value was added.
func (l *List[T]) Add(v T) bool {
	_, seen := l.seen[v]
	if !seen {
		l.seen[v] = struct{}{}
		l.List = append(l.List, v)
	}
	return !seen
}

func (l *List[T]) Has(v T) bool {
	_, ok := l.seen[v]
	return ok
}

func (l *List[T]) Append(values ...T) {
	for _, v := range values {
		if _, ok := l.seen[v]; !ok {
			_ = l.Add(v)
		}
	}
}
