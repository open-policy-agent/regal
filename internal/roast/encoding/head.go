package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type headCodec struct{}

func (*headCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*headCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	head := *((*ast.Head)(ptr))

	write.ObjectStart(stream, head.Location)

	if head.Reference != nil {
		write.ValsArrayAttr(stream, "ref", head.Reference)
	}

	if len(head.Args) > 0 {
		write.ValsArrayAttr(stream, "args", head.Args)
	}

	if head.Assign {
		write.Bool(stream, "assign", head.Assign)
	}

	if head.Key != nil {
		stream.WriteObjectField("key")
		write.Term(stream, head.Key)
	}

	if head.Value != nil {
		// Strip location from generated `true` values, as they don't have one
		if head.Value.Location != nil && head.Location != nil {
			if head.Value.Location.Row == head.Location.Row && head.Value.Location.Col == head.Location.Col {
				head.Value.Location = nil
			}
		}

		if head.Key != nil {
			stream.WriteMore()
		}

		stream.WriteObjectField("value")
		write.Term(stream, head.Value)
	}

	write.ObjectEnd(stream)
}
