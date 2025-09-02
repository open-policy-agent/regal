package util

import (
	"slices"
	"testing"
)

func TestNewSet(t *testing.T) {
	t.Parallel()

	s := NewSet(1, 2, 3)

	if got, expected := s.Size(), 3; got != expected {
		t.Fatalf("unexpected size, got: %v, want: %v", got, expected)
	}

	if got, expected := s.Contains(1, 2, 3), true; got != expected {
		t.Fatalf("unexpected Contains result, got: %v, want: %v", got, expected)
	}

	if got, expected := s.Contains(4), false; got != expected {
		t.Fatalf("unexpected Contains result for missing element, got: %v, want: %v", got, expected)
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()

	s := NewSet[int]()
	s.Add(10)

	if got, expected := s.Contains(10), true; got != expected {
		t.Fatalf("unexpected Contains result after Add, got: %v, want: %v", got, expected)
	}

	s.Add(20, 30, 40)

	if got, expected := s.Contains(20, 30, 40), true; got != expected {
		t.Fatalf("unexpected Contains result after adding multiple elements, got: %v, want: %v", got, expected)
	}

	initialSize := s.Size()
	s.Add(10, 20)

	if got, expected := s.Size(), initialSize; got != expected {
		t.Fatalf("size changed after adding duplicate elements, got: %v, want: %v", got, expected)
	}
}

func TestRemove(t *testing.T) {
	t.Parallel()

	s := NewSet(1, 2, 3, 4, 5)
	s.Remove(2)

	if got, expected := s.Contains(2), false; got != expected {
		t.Fatalf("unexpected Contains result after Remove, got: %v, want: %v", got, expected)
	}

	s.Remove(1, 3, 5)
	s.Remove(100, 200) // Removing random elements should be ok too

	if got, expected := s.Contains(1, 3, 5), false; got != expected {
		t.Fatalf("unexpected Contains result after removing multiple elements, got: %v, want: %v", got, expected)
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	s := NewSet("apple", "banana", "cherry")

	if got, expected := s.Contains("banana"), true; got != expected {
		t.Fatalf("unexpected Contains result for existing element, got: %v, want: %v", got, expected)
	}

	if got, expected := s.Contains("apple", "banana"), true; got != expected {
		t.Fatalf("unexpected Contains result for multiple existing elements, got: %v, want: %v", got, expected)
	}

	if got, expected := s.Contains("grape"), false; got != expected {
		t.Fatalf("unexpected Contains result for missing element, got: %v, want: %v", got, expected)
	}

	if got, expected := s.Contains("banana", "grape"), false; got != expected {
		t.Fatalf("unexpected Contains result when at least one element is missing, got: %v, want: %v", got, expected)
	}
}

func TestSize(t *testing.T) {
	t.Parallel()

	s := NewSet(1, 2, 3)

	if got, expected := s.Size(), 3; got != expected {
		t.Fatalf("unexpected size, got: %v, want: %v", got, expected)
	}

	s.Add(4, 5)

	if got, expected := s.Size(), 5; got != expected {
		t.Fatalf("unexpected size after Add, got: %v, want: %v", got, expected)
	}

	s.Remove(2, 3)

	if got, expected := s.Size(), 3; got != expected {
		t.Fatalf("unexpected size after Remove, got: %v, want: %v", got, expected)
	}
}

func TestItems(t *testing.T) {
	t.Parallel()

	s := NewSet(1, 2, 3, 4)
	items := s.Items()
	expected := []int{1, 2, 3, 4}

	slices.Sort(items)
	slices.Sort(expected)

	if got := slices.Equal(expected, items); !got {
		t.Fatalf("unexpected Items result, got: %v, want: %v", items, expected)
	}

	s.Remove(2, 3)

	expected = []int{1, 4}
	items = s.Items()

	slices.Sort(items)
	slices.Sort(expected)

	if got := slices.Equal(expected, items); !got {
		t.Fatalf("unexpected Items result after Remove, got: %v, want: %v", items, expected)
	}
}

func TestEmptySet(t *testing.T) {
	t.Parallel()

	s := NewSet[int]()

	if got, expected := s.Size(), 0; got != expected {
		t.Fatalf("unexpected size for empty set, got: %v, want: %v", got, expected)
	}

	if got := len(s.Items()); got != 0 {
		t.Fatalf("unexpected Items result for empty set, got: %v, want: %v", got, 0)
	}
}

func TestDiff(t *testing.T) {
	t.Parallel()

	a := NewSet(1, 2, 3, 4, 5)
	b := NewSet(3, 4, 5, 6, 7)

	diffSet := a.Diff(b)
	expected := []int{1, 2}

	items := diffSet.Items()
	slices.Sort(items)
	slices.Sort(expected)

	if got := slices.Equal(expected, items); !got {
		t.Fatalf("unexpected Diff result, got: %v, want: %v", items, expected)
	}
}
