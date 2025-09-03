package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/roast/encoding/util"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

type ruleCodec struct{}

func (*ruleCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*ruleCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	rule := *((*ast.Rule)(ptr))

	util.ObjectStart(stream, rule.Location)

	if len(rule.Annotations) > 0 {
		util.WriteValsArrayAttr(stream, strAnnotations, rule.Annotations)
	}

	if rule.Default {
		util.WriteBool(stream, strDefault, rule.Default)
	}

	if rule.Head != nil {
		util.WriteVal(stream, strHead, rule.Head)
	}

	if !rast.IsBodyGenerated(&rule) {
		util.WriteVal(stream, strBody, rule.Body)
	}

	if rule.Else != nil {
		util.WriteVal(stream, strElse, rule.Else)
	}

	util.ObjectEnd(stream)
}
