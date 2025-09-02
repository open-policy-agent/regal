package encoding

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	encutil "github.com/open-policy-agent/regal/internal/roast/encoding/util"
	"github.com/open-policy-agent/regal/internal/util"
)

type moduleCodec struct{}

func (*moduleCodec) IsEmpty(_ unsafe.Pointer) bool {
	return false
}

func (*moduleCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	mod := *((*ast.Module)(ptr))

	stream.WriteObjectStart()

	hasWritten := false

	if mod.Package != nil {
		stream.WriteObjectField(strPackage)

		if len(mod.Annotations) > 0 {
			stream.Attachment = util.Filter(mod.Annotations, notDocumentOrRuleScope)
		}

		stream.WriteVal(mod.Package)

		stream.Attachment = nil
		hasWritten = true
	}

	if len(mod.Imports) > 0 {
		if hasWritten {
			stream.WriteMore()
		}

		stream.WriteObjectField(strImports)
		encutil.WriteValsArray(stream, mod.Imports)

		hasWritten = true
	}

	if len(mod.Rules) > 0 {
		if hasWritten {
			stream.WriteMore()
		}

		stream.WriteObjectField(strRules)
		encutil.WriteValsArray(stream, mod.Rules)

		hasWritten = true
	}

	if len(mod.Comments) > 0 {
		if hasWritten {
			stream.WriteMore()
		}

		stream.WriteObjectField(strComments)
		encutil.WriteValsArray(stream, mod.Comments)
	}

	stream.WriteObjectEnd()
}

func notDocumentOrRuleScope(a *ast.Annotations) bool {
	return a.Scope != "document" && a.Scope != "rule"
}
