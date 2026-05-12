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

// ObjectElems returns a slice of object elements, without heap allocations.
// Use Key() and Value() to access the key and value of each element.
func ObjectElems(o ast.Object) objectElemSlice {
	// Since ast.Object is an interface, we must first convert it to the internal
	// form of an interface to be able to then access the underlying concrete type
	// (*ast.object).
	return (*Object)((*eface)(unsafe.Pointer(&o)).data).keys
}

func (oe *objectElem) Key() *ast.Term {
	return oe.key
}

func (oe *objectElem) Value() *ast.Term {
	return oe.value
}
