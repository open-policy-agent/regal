package util

import (
	"fmt"
	"iter"
	"maps"
)

// Set is a generic set implementation.
type Set[T comparable] struct {
	elements map[T]struct{}
}

// NewSet creates a new set from the supplied items.
func NewSet[T comparable](items ...T) *Set[T] {
	s := &Set[T]{elements: make(map[T]struct{}, len(items))}

	s.Add(items...)

	return s
}

// NewSetFromKeys creates a new set from the keys of map m.
func NewSetFromKeys[T comparable, V any](m map[T]V) *Set[T] {
	s := &Set[T]{elements: make(map[T]struct{}, len(m))}
	for k := range m {
		s.elements[k] = struct{}{}
	}

	return s
}

// Add adds one or more items to the set.
func (s *Set[T]) Add(items ...T) {
	for _, item := range items {
		s.elements[item] = struct{}{}
	}
}

// Remove removes one or more items from the set.
func (s *Set[T]) Remove(items ...T) {
	for _, item := range items {
		delete(s.elements, item)
	}
}

// Contains checks if all given items are in the set.
func (s *Set[T]) Contains(items ...T) bool {
	for _, item := range items {
		if _, exists := s.elements[item]; !exists {
			return false
		}
	}

	return true
}

// Size returns the number of elements in the set.
func (s *Set[T]) Size() int {
	return len(s.elements)
}

// Items returns all elements as a slice.
func (s *Set[T]) Items() []T {
	items := make([]T, 0, len(s.elements))
	for item := range s.elements {
		items = append(items, item)
	}

	return items
}

// Diff returns a new set containing items from the current set that are not in the given set B.
func (s *Set[T]) Diff(b *Set[T]) *Set[T] {
	diffSet := NewSet[T]()

	for item := range s.elements {
		if !b.Contains(item) {
			diffSet.Add(item)
		}
	}

	return diffSet
}

// Intersect returns a new set containing items that are present in both sets.
func (s *Set[T]) Intersect(b *Set[T]) *Set[T] {
	intersects := NewSet[T]()

	for item := range s.elements {
		if b.Contains(item) {
			intersects.Add(item)
		}
	}

	return intersects
}

func (s *Set[T]) Equal(b *Set[T]) bool {
	if s.Size() != b.Size() {
		return false
	}

	for item := range s.elements {
		if !b.Contains(item) {
			return false
		}
	}

	return true
}

// Values returns an iterator of all items in the set.
func (s *Set[T]) Values() iter.Seq[T] {
	return maps.Keys(s.elements)
}

func (s *Set[T]) String() string {
	return fmt.Sprintf("%v", s.Items())
}
