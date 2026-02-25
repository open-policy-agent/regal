package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
)

type numberCodec struct{}

func (*numberCodec) IsEmpty(ptr unsafe.Pointer) bool {
	return *((*ast.Number)(ptr)) == ""
}

func (*numberCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	num := *((*ast.Number)(ptr))
	buf := stream.Buffer()

	buf, _ = num.AppendText(buf)

	stream.SetBuffer(buf)
}
