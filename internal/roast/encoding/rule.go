package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

type ruleCodec struct{}

func (*ruleCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*ruleCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	rule := *((*ast.Rule)(ptr))

	write.ObjectStart(stream, rule.Location)

	if len(rule.Annotations) > 0 {
		write.ValsArrayAttr(stream, "annotations", rule.Annotations)
	}

	if rule.Default {
		write.Bool(stream, "default", rule.Default)
	}

	if rule.Head != nil {
		write.Val(stream, "head", rule.Head)
	}

	if !rast.IsBodyGenerated(&rule) {
		write.Val(stream, "body", rule.Body)
	}

	if rule.Else != nil {
		write.Val(stream, "else", rule.Else)
	}

	write.ObjectEnd(stream)
}
