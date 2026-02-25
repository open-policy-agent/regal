package write

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
)

func ValsArray[T any](stream *jsoniter.Stream, vals []T) {
	stream.WriteArrayStart()

	for i, val := range vals {
		if i > 0 {
			stream.WriteMore()
		}

		stream.WriteVal(val)
	}

	stream.WriteArrayEnd()
}

func ValsArrayAttr[T any](stream *jsoniter.Stream, name string, vals []T) {
	stream.WriteObjectField(name)
	ValsArray(stream, vals)
	stream.WriteMore()
}

func Object[V any](stream *jsoniter.Stream, name string, obj map[string]V) {
	stream.WriteObjectField(name)
	stream.WriteObjectStart()

	i := 0

	for key, value := range obj {
		if i > 0 {
			stream.WriteMore()
		}

		stream.WriteObjectField(key)
		stream.WriteVal(value)

		i++
	}

	stream.WriteObjectEnd()
	stream.WriteMore()
}

func Val(stream *jsoniter.Stream, field string, val any) {
	stream.WriteObjectField(field)
	stream.WriteVal(val)
	stream.WriteMore()
}

func String(stream *jsoniter.Stream, field, val string) {
	stream.WriteObjectField(field)
	stream.WriteString(val)
	stream.WriteMore()
}

func Bool(stream *jsoniter.Stream, field string, val bool) {
	stream.WriteObjectField(field)
	stream.WriteBool(val)
	stream.WriteMore()
}

func ObjectStart(stream *jsoniter.Stream, loc *ast.Location) {
	stream.WriteObjectStart()

	if loc != nil {
		Val(stream, "location", loc)
	}
}

func ObjectEnd(stream *jsoniter.Stream) {
	stream.SetBuffer(bytes.TrimRight(stream.Buffer(), ",\n "))
	stream.WriteObjectEnd()
}

func Term(stream *jsoniter.Stream, term *ast.Term) {
	ObjectStart(stream, term.Location)

	if term.Value != nil {
		String(stream, "type", ast.ValueName(term.Value))
		Val(stream, "value", term.Value)
	}

	ObjectEnd(stream)
}
