package util_test

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/util"
)

func TestNewSet(t *testing.T) {
	t.Parallel()

	s := util.NewSet(1, 2, 3)
	must.Equal(t, 3, s.Size(), "size of set")
	assert.True(t, s.Contains(1, 2, 3), "set should contain initial elements")
	assert.False(t, s.Contains(4), "set should not contain missing element")
}

func TestAdd(t *testing.T) {
	t.Parallel()

	s := util.NewSet[int]()

	s.Add(10)
	assert.True(t, s.Contains(10), "set should contain added element")

	s.Add(20, 30, 40)
	assert.True(t, s.Contains(20, 30, 40), "set should contain added elements")

	initialSize := s.Size()
	s.Add(10, 20)
	assert.Equal(t, initialSize, s.Size(), "size should not change when adding duplicate elements")
}

func TestRemove(t *testing.T) {
	t.Parallel()

	s := util.NewSet(1, 2, 3, 4, 5)

	s.Remove(2)
	assert.False(t, s.Contains(2), "set should not contain removed element")

	s.Remove(1, 3, 5)
	s.Remove(100, 200) // Removing random elements should be ok too

	assert.False(t, s.Contains(1, 3, 5), "set should not contain removed elements")
}

func TestContains(t *testing.T) {
	t.Parallel()

	s := util.NewSet("apple", "banana", "cherry")

	assert.True(t, s.Contains("banana"), "set should contain 'banana'")
	assert.True(t, s.Contains("apple", "banana"), "set should contain 'apple' and 'banana'")
	assert.False(t, s.Contains("grape"), "set should not contain 'grape'")
	assert.False(t, s.Contains("banana", "grape"), "set should not contain 'grape' even if it contains 'banana'")
}

func TestSize(t *testing.T) {
	t.Parallel()

	s := util.NewSet(1, 2, 3)
	must.Equal(t, 3, s.Size(), "initial size of set")

	s.Add(4, 5)
	must.Equal(t, 5, s.Size(), "size after adding elements")

	s.Remove(2, 3)
	must.Equal(t, 3, s.Size(), "size after removing elements")
}

func TestItems(t *testing.T) {
	t.Parallel()

	s := util.NewSet(1, 2, 3, 4)
	assert.SlicesEqual(t, []int{1, 2, 3, 4}, util.Sorted(s.Items()))

	s.Remove(2, 3)
	assert.SlicesEqual(t, []int{1, 4}, util.Sorted(s.Items()))
}

func TestEmptySet(t *testing.T) {
	t.Parallel()

	s := util.NewSet[int]()
	assert.Equal(t, 0, s.Size(), "size of empty set should be 0")
	assert.Equal(t, 0, len(s.Items()), "Items of empty set should be empty")
}

func TestDiff(t *testing.T) {
	t.Parallel()

	diff := util.NewSet(1, 2, 3, 4, 5).Diff(util.NewSet(3, 4, 5, 6, 7))
	assert.SlicesEqual(t, []int{1, 2}, util.Sorted(diff.Items()))
}
