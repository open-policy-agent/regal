package write

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"
)

var locationKeyArr = [...]byte{'"', 'l', 'o', 'c', 'a', 't', 'i', 'o', 'n', '"'}

type marshalToFn[T any] func(*jsontext.Encoder, T) error

func ArrayField[T any](enc *jsontext.Encoder, name string, vals []T) error {
	return ArrayFieldFn(enc, name, vals, marshalEncodeUnvariadic)
}

func ArrayFieldFn[T any](enc *jsontext.Encoder, name string, vals []T, f marshalToFn[T]) error {
	enc.WriteToken(jsontext.String(name))

	return ArrayToFn(enc, vals, f)
}

func ArrayToFn[T any](enc *jsontext.Encoder, arr []T, f marshalToFn[T]) error {
	enc.WriteToken(jsontext.BeginArray)

	for _, item := range arr {
		if err := f(enc, item); err != nil {
			return err
		}
	}

	return enc.WriteToken(jsontext.EndArray)
}

func ObjectStart(enc *jsontext.Encoder, loc *ast.Location) (err error) {
	enc.WriteToken(jsontext.BeginObject)

	if loc != nil {
		enc.WriteValue(locationKeyArr[:])

		endRow, endCol := loc.End()

		buf := append(enc.AvailableBuffer(), '"')
		buf = append(util.AppendInt(buf, loc.Row), ':')
		buf = append(util.AppendInt(buf, loc.Col), ':')
		buf = append(util.AppendInt(buf, endRow), ':')
		buf = append(util.AppendInt(buf, endCol), '"')

		return enc.WriteValue(buf)
	}

	return err
}

func Tokens(enc *jsontext.Encoder, tokens ...jsontext.Token) error {
	for _, token := range tokens {
		if err := enc.WriteToken(token); err != nil {
			return err
		}
	}

	return nil
}

func WhenTrue(enc *jsontext.Encoder, condition bool, name string) error {
	if !condition {
		return nil
	}

	return Tokens(enc, jsontext.String(name), jsontext.True)
}

func StringField(enc *jsontext.Encoder, name, val string) error {
	return Tokens(enc, jsontext.String(name), jsontext.String(val))
}

func Field[T any](enc *jsontext.Encoder, name string, val T) error {
	return FieldFn(enc, name, val, marshalEncodeUnvariadic)
}

func FieldFn[T any](enc *jsontext.Encoder, name string, val T, f marshalToFn[T]) error {
	enc.WriteToken(jsontext.String(name))

	return f(enc, val)
}

func marshalEncodeUnvariadic[T any](enc *jsontext.Encoder, val T) error {
	return json.MarshalEncode(enc, val)
}
