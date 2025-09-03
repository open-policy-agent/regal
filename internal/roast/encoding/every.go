package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type everyCodec struct{}

func (*everyCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*everyCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	every := *((*ast.Every)(ptr))

	util.ObjectStart(stream, every.Location)
	util.WriteVal(stream, strKey, every.Key)
	util.WriteVal(stream, strValue, every.Value)
	util.WriteVal(stream, strDomain, every.Domain)
	util.WriteVal(stream, strBody, every.Body)
	util.ObjectEnd(stream)
}
