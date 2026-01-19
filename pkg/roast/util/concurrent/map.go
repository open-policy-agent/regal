package concurrent

import (
	"maps"
	"sync"
)

// Map provides a simple concurrent map implementation.
type Map[K comparable, V any] struct {
	m    map[K]V
	murw sync.RWMutex
}

// ValueTransformer is a function that transforms a value in the map.
type ValueTransformer[V any] func(V) V

// MapOf creates a new concurrent map wrapping the given map.
func MapOf[K comparable, V any](src map[K]V) *Map[K, V] {
	return &Map[K, V]{
		m:    src,
		murw: sync.RWMutex{},
	}
}

// Get returns the value associated with the given key, and a boolean indicating
// whether the key was found.
func (cm *Map[K, V]) Get(k K) (V, bool) {
	cm.murw.RLock()

	v, ok := cm.m[k]

	cm.murw.RUnlock()

	return v, ok
}

// GetUnchecked returns the value associated with the given key without checking
// for its existence (nil if not found).
func (cm *Map[K, V]) GetUnchecked(k K) V {
	cm.murw.RLock()

	v := cm.m[k]

	cm.murw.RUnlock()

	return v
}

// Set sets the value associated with the given key.
func (cm *Map[K, V]) Set(k K, v V) {
	cm.murw.Lock()

	cm.m[k] = v

	cm.murw.Unlock()
}

// Delete removes the value associated with the given key.
func (cm *Map[K, V]) Delete(k K) {
	cm.murw.Lock()

	delete(cm.m, k)

	cm.murw.Unlock()
}

func (cm *Map[K, V]) RenameKey(oldKey, newKey K) {
	cm.murw.Lock()

	if v, ok := cm.m[oldKey]; ok {
		cm.m[newKey] = v
		delete(cm.m, oldKey)
	}

	cm.murw.Unlock()
}

// Keys returns a slice of all keys in the map.
func (cm *Map[K, V]) Keys() []K {
	cm.murw.RLock()

	keys := make([]K, 0, len(cm.m))

	for k := range cm.m {
		keys = append(keys, k)
	}

	cm.murw.RUnlock()

	return keys
}

// Values returns a slice of all values in the map.
func (cm *Map[K, V]) Values() []V {
	cm.murw.RLock()

	vs := make([]V, len(cm.m))
	i := 0

	for _, v := range cm.m {
		vs[i] = v
		i++
	}

	cm.murw.RUnlock()

	return vs
}

// Len returns the number of elements in the map.
func (cm *Map[K, V]) Len() int {
	if cm == nil {
		return 0
	}

	cm.murw.RLock()

	l := len(cm.m)

	cm.murw.RUnlock()

	return l
}

// Clone returns a shallow copy of the map.
func (cm *Map[K, V]) Clone() map[K]V {
	cm.murw.RLock()

	m := maps.Clone(cm.m)

	cm.murw.RUnlock()

	return m
}

// Clear removes all elements from the map.
func (cm *Map[K, V]) Clear() {
	cm.murw.Lock()

	clear(cm.m)

	cm.murw.Unlock()
}

// UpdateValue updates the value associated with the given key using the provided
// transformer function.
func (cm *Map[K, V]) UpdateValue(key K, transformer ValueTransformer[V]) {
	cm.murw.Lock()

	var v V

	if vo, ok := cm.m[key]; ok {
		v = vo
	}

	cm.m[key] = transformer(v)

	cm.murw.Unlock()
}
