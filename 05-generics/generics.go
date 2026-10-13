// Package generics covers type parameters (Go 1.18+): write a function or type once, use it with many types.
package generics

import (
	"cmp"
	"errors"
)

// Map transforms every element. [T, R any] are TYPE PARAMETERS; Go infers them from the arguments.
func Map[T, R any](items []T, f func(T) R) []R {
	out := make([]R, len(items))
	for i, item := range items {
		out[i] = f(item)
	}
	return out
}

func Filter[T any](items []T, keep func(T) bool) []T {
	var out []T
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func Reduce[T, A any](items []T, initial A, f func(A, T) A) A {
	acc := initial
	for _, item := range items {
		acc = f(acc, item)
	}
	return acc
}

// Number is a CONSTRAINT: the set of types allowed. ~int also allows types whose underlying type is int.
type Number interface {
	~int | ~int64 | ~float64
}

func Sum[T Number](numbers ...T) T {
	var total T
	for _, n := range numbers {
		total += n
	}
	return total
}

// MaxBy works for any element type with a key that can be ordered (cmp.Ordered: ints, floats, strings).
func MaxBy[T any, K cmp.Ordered](items []T, key func(T) K) (T, bool) {
	var best T
	if len(items) == 0 {
		return best, false
	}
	best = items[0]
	for _, item := range items[1:] {
		if key(item) > key(best) {
			best = item
		}
	}
	return best, true
}

// Stack is a GENERIC TYPE: a type-safe container, no casting from `any`.
type Stack[T any] struct {
	items []T
}

var ErrEmpty = errors.New("stack is empty")

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, error) {
	var zero T
	if len(s.items) == 0 {
		return zero, ErrEmpty
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, nil
}

func (s *Stack[T]) Len() int { return len(s.items) }

// Cache is a generic key-value store; comparable = types usable as map keys.
type Cache[K comparable, V any] struct {
	data map[K]V
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{data: make(map[K]V)}
}

// GetOrLoad returns the cached value or computes, stores and returns it.
func (c *Cache[K, V]) GetOrLoad(key K, load func(K) (V, error)) (V, error) {
	if v, ok := c.data[key]; ok {
		return v, nil
	}
	v, err := load(key)
	if err != nil {
		return v, err
	}
	c.data[key] = v
	return v, nil
}
