package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type notCodec struct{}

func (*notCodec) IsEmpty(_ unsafe.Pointer) bool { return false }

func (*notCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	not := *((*ast.Not)(ptr))

	write.ObjectStart(stream, not.Location)

	write.Val(stream, "type", "not")

	if not.ExplicitBody {
		write.Bool(stream, "explicit_body", not.ExplicitBody)
	}

	write.Val(stream, "body", not.Body)

	write.ObjectEnd(stream)
}
