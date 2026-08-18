package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

type logicalAndCodec struct{}

func (*logicalAndCodec) IsEmpty(_ unsafe.Pointer) bool { return false }

func (*logicalAndCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	and := *((*ast.LogicalAnd)(ptr))

	writeLogical(stream, "and", and.Location, and.Lhs, and.Rhs, and.ExplicitLhs, and.ExplicitRhs)
}

type logicalOrCodec struct{}

func (*logicalOrCodec) IsEmpty(_ unsafe.Pointer) bool { return false }

func (*logicalOrCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	or := *((*ast.LogicalOr)(ptr))

	writeLogical(stream, "or", or.Location, or.Lhs, or.Rhs, or.ExplicitLhs, or.ExplicitRhs)
}

// writeLogical encodes the operands of an `and`/`or` expression. The explicit_lhs
// and explicit_rhs attributes tell whether the operand was written as a brace
// enclosed body (`{ x; y } or z`) rather than as a single expression.
func writeLogical(
	stream *jsoniter.Stream,
	op string,
	location *ast.Location,
	lhs, rhs ast.Body,
	explicitLhs, explicitRhs bool,
) {
	write.ObjectStart(stream, location)

	write.Val(stream, "type", op)

	if explicitLhs {
		write.Bool(stream, "explicit_lhs", explicitLhs)
	}

	if explicitRhs {
		write.Bool(stream, "explicit_rhs", explicitRhs)
	}

	write.Val(stream, "lhs", lhs)
	write.Val(stream, "rhs", rhs)

	write.ObjectEnd(stream)
}
