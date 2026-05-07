// Package funsafe collects Regal operations that requires "unsafe" access to OPA internals.
// Currently, this is used only to be able to traverse AST objects without allocating memory,
// which is of course not something that should be needed, but will have to do until OPA provides
// a native way to do this.
package funsafe

import (
	"sync"
	"unsafe"

	"github.com/open-policy-agent/opa/v1/ast"
)

type (
	Object struct {
		elems     map[int]*objectElem
		keys      objectElemSlice
		ground    int
		hash      int
		sortGuard *sync.Once
	}

	eface struct {
		rtype unsafe.Pointer
		data  unsafe.Pointer
	}

	objectElem struct {
		key   *ast.Term
		value *ast.Term
		next  *objectElem //nolint:unused
	}

	objectElemSlice []*objectElem
)

func ObjectElems(o ast.Object) objectElemSlice {
	ef := (*eface)(unsafe.Pointer(&o))
	obj := (*Object)(ef.data)

	return obj.keys
}

// Elems returns a slice of object elements, without heap allocations.
// Use Key() and Value() to access the key and value of each element.
func (o *Object) Elems() objectElemSlice {
	return o.keys
}

func (oe *objectElem) Key() *ast.Term {
	return oe.key
}

func (oe *objectElem) Value() *ast.Term {
	return oe.value
}
