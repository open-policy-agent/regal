package concurrent

import (
	"slices"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
)

type Object struct {
	o    ast.Object
	mu   sync.RWMutex
	keys map[string]*ast.Term
}

func NewObject() *Object {
	return &Object{
		o:    ast.NewObject(),
		mu:   sync.RWMutex{},
		keys: make(map[string]*ast.Term),
	}
}

func (co *Object) Get(key string) (term *ast.Term) {
	co.mu.RLock()

	if keyTerm, ok := co.keys[key]; ok {
		term = co.o.Get(keyTerm)
	}

	co.mu.RUnlock()

	return term
}

func (co *Object) Set(key string, value *ast.Term) {
	co.mu.Lock()

	keyTerm, ok := co.keys[key]
	if !ok {
		keyTerm = ast.InternedTerm(key)
		co.keys[key] = keyTerm
	}

	co.o.Insert(keyTerm, value)

	co.mu.Unlock()
}

func (co *Object) Delete(key string) {
	co.mu.Lock()

	co.o = deleteKey(co.o, key)

	delete(co.keys, key)

	co.mu.Unlock()
}

func (co *Object) Keep(keys ...string) ast.Object {
	co.mu.RLock()

	newObj, _ := co.o.Map(func(k, v *ast.Term) (*ast.Term, *ast.Term, error) {
		if slices.Contains(keys, stringValue(k)) {
			return nil, nil, nil
		}

		return k, v, nil
	})

	co.mu.RUnlock()

	return newObj
}

func (co *Object) Reset(o ast.Object) {
	co.mu.Lock()

	clear(co.keys)

	for _, key := range o.Keys() {
		if s := stringValue(key); s != "" {
			co.keys[s] = key
		}
	}

	co.o = o

	co.mu.Unlock()
}

func (co *Object) RenameKey(oldKey, newKey string) {
	co.mu.Lock()

	if oldKeyTerm, ok := co.keys[oldKey]; ok {
		if value := co.o.Get(oldKeyTerm); value != nil {
			co.keys[newKey] = ast.InternedTerm(newKey)
			co.o.Insert(co.keys[newKey], value)
			delete(co.keys, oldKey)
		}
	}

	co.mu.Unlock()
}

func (co *Object) UnsafeObject() ast.Object {
	return co.o
}

func stringValue(t *ast.Term) string {
	if s, ok := t.Value.(ast.String); ok {
		return string(s)
	}

	return ""
}

func deleteKey(o ast.Object, key string) ast.Object {
	n, _ := o.Map(func(t1, t2 *ast.Term) (*ast.Term, *ast.Term, error) {
		if stringValue(t1) == key {
			return nil, nil, nil
		}

		return t1, t2, nil
	})

	return n
}
