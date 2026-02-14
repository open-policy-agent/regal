package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type importCodec struct{}

func (*importCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*importCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	imp := *((*ast.Import)(ptr))

	write.ObjectStart(stream, imp.Location)

	if imp.Path != nil {
		write.Val(stream, "path", imp.Path)

		if imp.Alias != "" {
			write.Val(stream, "alias", imp.Alias)
		}
	}

	write.ObjectEnd(stream)
}
