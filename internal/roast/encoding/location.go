package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

type locationCodec struct{}

func (*locationCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*locationCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	location := (*ast.Location)(ptr)

	stream.SetBuffer(append(rast.AppendLocation(append(stream.Buffer(), '"'), location), '"'))
}
