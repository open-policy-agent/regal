package encoding

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/roast/encoding/write"
)

func TermMarshalToFn(enc *jsontext.Encoder, term *ast.Term) error {
	write.ObjectStart(enc, term.Location)

	if term.Value != nil {
		write.StringField(enc, "type", ast.ValueName(term.Value))
		enc.WriteToken(jsontext.String("value"))

		if err := ValueMarshalToFn(enc, term.Value); err != nil {
			return err
		}
	}

	return enc.WriteToken(jsontext.EndObject)
}

func ValueMarshalToFn(enc *jsontext.Encoder, value ast.Value) (err error) {
	switch t := value.(type) {
	case ast.Null:
		return enc.WriteToken(jsontext.Null)
	case ast.Number:
		return enc.WriteValue(util.StringToByteSlice(t))
	case ast.Boolean:
		if t {
			return enc.WriteToken(jsontext.True)
		}

		return enc.WriteToken(jsontext.False)
	case ast.String:
		return enc.WriteToken(jsontext.String(string(t)))
	case ast.Var:
		return enc.WriteToken(jsontext.String(string(t)))
	case ast.Object:
		return ObjectMarshalToFn(enc, t)
	case *ast.Array:
		return ArrayMarshalToFn(enc, t)
	case ast.Set:
		return SetMarshalToFn(enc, t)
	case *ast.Not:
		return NotMarshalToFn(enc, t)
	case ast.Ref:
		return RefMarshalToFn(enc, t)
	case ast.Call:
		return CallMarshalToFn(enc, t)
	case *ast.TemplateString:
		return TemplateStringMarshalToFn(enc, t)
	case *ast.ArrayComprehension:
		return ArrayComprehensionMarshalToFn(enc, t)
	case *ast.SetComprehension:
		return SetComprehensionMarshalToFn(enc, t)
	case *ast.ObjectComprehension:
		return ObjectComprehensionMarshalToFn(enc, t)
	default:
		return json.MarshalEncode(enc, t)
	}
}

func NumberUnmarshalFromFunc(dec *jsontext.Decoder, num *ast.Number) error {
	val, err := dec.ReadValue()
	if err == nil {
		*num = ast.Number(string(val))
	}

	return err
}
