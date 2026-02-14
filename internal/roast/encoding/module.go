package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	encutil "github.com/open-policy-agent/regal/internal/roast/encoding/write"
	"github.com/open-policy-agent/regal/internal/util"
)

type moduleCodec struct{}

func (*moduleCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*moduleCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	mod := *((*ast.Module)(ptr))

	encutil.ObjectStart(stream, nil)

	if mod.Package != nil {
		if len(mod.Annotations) > 0 {
			stream.Attachment = util.Filter(mod.Annotations, notDocumentOrRuleScope)
		}

		encutil.Val(stream, "package", mod.Package)
		stream.Attachment = nil
	}

	if len(mod.Imports) > 0 {
		encutil.ValsArrayAttr(stream, "imports", mod.Imports)
	}

	if len(mod.Rules) > 0 {
		encutil.ValsArrayAttr(stream, "rules", mod.Rules)
	}

	if len(mod.Comments) > 0 {
		encutil.ValsArrayAttr(stream, "comments", mod.Comments)
	}

	encutil.ObjectEnd(stream)
}

func notDocumentOrRuleScope(a *ast.Annotations) bool {
	return a.Scope != "document" && a.Scope != "rule"
}
