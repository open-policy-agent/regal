package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type termCodec struct{}

func (*termCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*termCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	term := *((*ast.Term)(ptr))

	util.ObjectStart(stream, term.Location)

	if term.Value != nil {
		util.WriteString(stream, strType, ast.ValueName(term.Value))
		util.WriteVal(stream, strValue, term.Value)
	}

	util.ObjectEnd(stream)
}
