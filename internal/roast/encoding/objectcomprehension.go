package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type objectComprehensionCodec struct{}

func (*objectComprehensionCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*objectComprehensionCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	oc := *((*ast.ObjectComprehension)(ptr))

	util.ObjectStart(stream, nil)
	util.WriteVal(stream, strKey, oc.Key)
	util.WriteVal(stream, strValue, oc.Value)
	util.WriteVal(stream, strBody, oc.Body)
	util.ObjectEnd(stream)
}
