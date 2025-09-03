package util

import (
	"bytes"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"
)

func WriteValsArray[T any](stream *jsoniter.Stream, vals []T) {
	stream.WriteArrayStart()

	for i, val := range vals {
		if i > 0 {
			stream.WriteMore()
		}

		stream.WriteVal(val)
	}

	stream.WriteArrayEnd()
}

func WriteValsArrayAttr[T any](stream *jsoniter.Stream, name string, vals []T) {
	stream.WriteObjectField(name)
	WriteValsArray(stream, vals)
	stream.WriteMore()
}

func WriteObject[V any](stream *jsoniter.Stream, name string, obj map[string]V) {
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

func WriteVal(stream *jsoniter.Stream, field string, val any) {
	stream.WriteObjectField(field)
	stream.WriteVal(val)
	stream.WriteMore()
}

func WriteString(stream *jsoniter.Stream, field string, val string) {
	stream.WriteObjectField(field)
	stream.WriteString(val)
	stream.WriteMore()
}

func WriteBool(stream *jsoniter.Stream, field string, val bool) {
	stream.WriteObjectField(field)
	stream.WriteBool(val)
	stream.WriteMore()
}

func ObjectStart(stream *jsoniter.Stream, loc *ast.Location) {
	stream.WriteObjectStart()

	if loc != nil {
		WriteVal(stream, "location", loc)
	}
}

func ObjectEnd(stream *jsoniter.Stream) {
	stream.SetBuffer(bytes.TrimRight(stream.Buffer(), ",\n "))
	stream.WriteObjectEnd()
}
