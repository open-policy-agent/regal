package encoding

import (
	"encoding/base64"
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type commentCodec struct{}

func (*commentCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*commentCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	comment := *((*ast.Comment)(ptr))

	write.ObjectStart(stream, comment.Location)

	stream.WriteObjectField("text")
	stream.SetBuffer(append(base64.StdEncoding.AppendEncode(append(stream.Buffer(), '"'), comment.Text), '"'))
	stream.WriteObjectEnd()
}
