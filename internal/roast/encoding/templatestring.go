package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
)

type templateStringCodec struct{}

func (*templateStringCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*templateStringCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	sc := *((*ast.TemplateString)(ptr))

	stream.WriteObjectStart()

	if sc.MultiLine {
		stream.WriteObjectField("multi_line")
		stream.WriteBool(sc.MultiLine)
		stream.WriteMore()
	}

	stream.WriteObjectField("parts")
	stream.WriteArrayStart()

	for i, part := range sc.Parts {
		if i > 0 {
			stream.WriteMore()
		}

		if _, ok := part.(*ast.Expr); ok {
			stream.Attachment = "interpolated"
		}

		stream.WriteVal(part)
	}

	stream.WriteArrayEnd()
	stream.WriteObjectEnd()
}
