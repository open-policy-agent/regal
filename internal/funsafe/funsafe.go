// Package funsafe collects Regal operations that requires "unsafe" access to OPA internals.
// Currently, this is used only to be able to traverse AST objects without allocating memory,
// which is of course not something that should be needed, but will have to do until OPA provides
// a native way to do this.
package funsafe

import (
	"sync"
	"unsafe"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/debug"
)

var _ debug.Variable = DebugVar{}

type (
	Object struct {
		elems     map[int]*objectElem
		keys      objectElemSlice
		ground    int
		hash      int
		sortGuard *sync.Once
	}
	// DebugVar is a temporary mirror of [debug.Variable] to avoid the default implementation's
	// truncating of the variable value to 100 characters, and to allow access to the underlying [ast.Value].
	DebugVar struct {
		name     string
		value    string
		astValue ast.Value
		varRef   int
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
	namedVar        struct {
		name  string
		value ast.Value
	}
)

// ToDebugVar converts a [debug.Variable] to a [DebugVar]. See [DebugVar] for details on why this is needed.
func ToDebugVar(v debug.Variable) DebugVar {
	ef := (*eface)(unsafe.Pointer(&v))
	nv := *(*namedVar)(ef.data)

	return DebugVar{
		name:     nv.name,
		value:    nv.value.String(),
		astValue: nv.value,
		varRef:   int(v.VariablesReference()),
	}
}

func (dv DebugVar) Name() string {
	return dv.name
}

func (dv DebugVar) Type() string {
	return ast.ValueName(dv.astValue)
}

func (dv DebugVar) Value() string {
	return dv.value
}

func (dv DebugVar) VariablesReference() debug.VarRef {
	return debug.VarRef(dv.varRef)
}

func (dv DebugVar) ASTValue() ast.Value {
	return dv.astValue
}

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
