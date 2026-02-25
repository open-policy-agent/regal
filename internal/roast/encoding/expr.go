package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type exprCodec struct{}

func (*exprCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*exprCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	expr := *((*ast.Expr)(ptr))

	write.ObjectStart(stream, expr.Location)

	if expr.Negated {
		write.Bool(stream, "negated", expr.Negated)
	}

	if expr.Generated {
		write.Bool(stream, "generated", expr.Generated)
	}

	if stream.Attachment != nil {
		if s, ok := stream.Attachment.(string); ok && s == "interpolated" {
			write.Bool(stream, "interpolated", true)
			stream.Attachment = nil
		}
	}

	if len(expr.With) > 0 {
		write.ValsArrayAttr(stream, "with", expr.With)
	}

	if expr.Terms != nil {
		stream.WriteObjectField("terms")

		switch t := expr.Terms.(type) {
		case *ast.Term:
			write.Term(stream, t)
		case []*ast.Term:
			write.ValsArray(stream, t)
		case *ast.SomeDecl:
			stream.WriteVal(t)
		case *ast.Every:
			stream.WriteVal(t)
		}
	}

	write.ObjectEnd(stream)
}
