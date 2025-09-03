package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
)

type headCodec struct{}

func (*headCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*headCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	head := *((*ast.Head)(ptr))

	util.ObjectStart(stream, head.Location)

	if head.Reference != nil {
		util.WriteVal(stream, strRef, head.Reference)
	}

	if len(head.Args) > 0 {
		util.WriteValsArrayAttr(stream, strArgs, head.Args)
	}

	if head.Assign {
		util.WriteBool(stream, strAssign, head.Assign)
	}

	if head.Key != nil {
		util.WriteVal(stream, strKey, head.Key)
	}

	if head.Value != nil {
		// Strip location from generated `true` values, as they don't have one
		if head.Value.Location != nil && head.Location != nil {
			if head.Value.Location.Row == head.Location.Row && head.Value.Location.Col == head.Location.Col {
				head.Value.Location = nil
			}
		}

		util.WriteVal(stream, strValue, head.Value)
	}

	util.ObjectEnd(stream)
}
