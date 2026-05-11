package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

type commentCodec struct{}

func (*commentCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*commentCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	comment := *((*ast.Comment)(ptr))

	// Use location string — text is retrieved dynamically via regal.file.lines
	buf := stream.Buffer()
	buf = append(buf, '"')
	buf = rast.AppendLocation(buf, comment.Location)
	buf = append(buf, '"')

	stream.SetBuffer(buf)
}
