package encoding

import (
	"encoding"
	"errors"
	"io"
	"log"
	"strconv"

	jsoniter "github.com/json-iterator/go"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/funsafe"
	"github.com/open-policy-agent/regal/pkg/roast/rast"

	_ "github.com/open-policy-agent/regal/pkg/roast/intern"
)

// ValueMarshaller provides the most efficient methods for encoding and decoding
// OPA's ast.Values matching JSON types, as plain JSON. Use this for when you do
// **not** want RoAST.
type ValueMarshaller struct {
	config jsoniter.Config
	json   jsoniter.API
}

type Options struct {
	UseNumber bool
}

var (
	// SafeNumberConfig config in case the faster number handling fails.
	// See: https://github.com/open-policy-agent/regal/issues/1592
	SafeNumberConfig = safeNumberConfig.Froze()
	safeNumberConfig = jsoniter.Config{
		UseNumber:                     true,
		EscapeHTML:                    false,
		MarshalFloatWith6Digits:       true,
		ObjectFieldMustBeSimpleString: true,
	}
)

// JSON returns the fastest jsoniter configuration
// It is preferred using this function instead of jsoniter.ConfigFastest directly
// as there as the init function needs to be called to register the custom types,
// which will happen automatically on import.
func JSON() jsoniter.API {
	return jsoniter.ConfigFastest
}

// JSONUnmarshalTo unmarshals JSON into the provided type T.
func JSONUnmarshalTo[T any](bs []byte) (to T, err error) {
	if err = jsoniter.ConfigFastest.Unmarshal(bs, &to); err != nil {
		err = SafeNumberConfig.Unmarshal(bs, &to)
	}

	return to, err
}

// JSONRoundTrip convert any value to JSON and back again.
func JSONRoundTrip(from, to any) error {
	bs, err := jsoniter.ConfigFastest.Marshal(from)
	if err != nil {
		return err
	}

	if err = jsoniter.ConfigFastest.Unmarshal(bs, to); err != nil {
		return SafeNumberConfig.Unmarshal(bs, to)
	}

	return nil
}

// JSONRoundTripTo convert any value to JSON and back again, returning the new value or an error.
func JSONRoundTripTo[T any](from any) (to T, err error) {
	err = JSONRoundTrip(from, &to)

	return to, err
}

// MustJSONRoundTrip convert any value to JSON and back again, exit on failure.
func MustJSONRoundTrip(from, to any) {
	if err := JSONRoundTrip(from, to); err != nil {
		log.Fatal(err)
	}
}

// NewIndentEncoder creates a new JSON encoder with the specified prefix and indent, encoding to w.
func NewIndentEncoder(w io.Writer, prefix, indent string) *jsoniter.Encoder {
	enc := JSON().NewEncoder(w)
	enc.SetIndent(prefix, indent)

	return enc
}

func OfValue() ValueMarshaller {
	return ValueMarshaller{config: safeNumberConfig, json: SafeNumberConfig}
}

func (m ValueMarshaller) Encode(out io.Writer, value ast.Value) error {
	stream := m.json.BorrowStream(out)
	defer m.json.ReturnStream(stream)

	if stream.Error != nil {
		return stream.Error
	}

	m.toJSON(value, stream)

	return stream.Flush()
}

func (m ValueMarshaller) Decode(bs []byte) (val ast.Value, err error) {
	iter := m.json.BorrowIterator(bs)
	defer m.json.ReturnIterator(iter)

	value := m.toValue(iter, iter.WhatIsNext())
	if iter.Error != nil && !errors.Is(iter.Error, io.EOF) {
		err = iter.Error
	}

	return value, err
}

func (m ValueMarshaller) toJSON(value ast.Value, stream *jsoniter.Stream) {
	switch v := value.(type) {
	case ast.Null:
		stream.WriteNil()
	case ast.Boolean:
		stream.WriteBool(bool(v))
	case ast.Number:
		stream.WriteRaw(string(v))
	case ast.String:
		stream.WriteString(string(v))
	case *ast.Array:
		stream.WriteArrayStart()

		l := v.Len()
		for i := range l {
			m.toJSON(v.Elem(i).Value, stream)

			if i != l-1 {
				stream.WriteMore()
			}
		}

		stream.WriteArrayEnd()
	case ast.Set:
		stream.WriteArrayStart()

		l := v.Len()
		for i, term := range v.Slice() {
			m.toJSON(term.Value, stream)

			if i != l-1 {
				stream.WriteMore()
			}
		}

		stream.WriteArrayEnd()
	case ast.Object:
		stream.WriteObjectStart()

		if v.Len() > 0 {
			// Use of "funsafe" to allow traversing the object without
			// allocating, which isn't possible to do with any of the
			// public AST methods.. but needless to say, this should be
			// fixed in OPA rather than via hacks like this
			for i, elem := range funsafe.ObjectElems(v) {
				switch key := elem.Key().Value.(type) {
				case ast.String:
					stream.WriteObjectField(string(key))
				case encoding.TextAppender:
					if bs, err := key.AppendText(stream.Buffer()); err == nil {
						stream.SetBuffer(bs)
					} else {
						stream.Error = err
					}
				default:
					stream.WriteObjectField(key.String())
				}

				m.toJSON(elem.Value().Value, stream)

				if i != v.Len()-1 {
					stream.WriteMore()
				}
			}
		}

		stream.WriteObjectEnd()
	default:
		panic("can't encode value of type " + ast.ValueName(v))
	}
}

func (m ValueMarshaller) toValue(iter *jsoniter.Iterator, valueType jsoniter.ValueType) ast.Value {
	switch valueType {
	case jsoniter.StringValue:
		// NOTE: sadly, this allocates :/ which it wouldn't have to do if
		// iter.ReadStringAsSlice() actually worked... but it fails for all
		// strings with escapes, as for some reason it'll then stop at the
		// first **escaped quote** instead of the actual end of the string.
		// perhaps there's some elaborate way to work around this, but since
		// we'll be ditching jsoniter for json/v2 at our first opportunity,
		// not spending more time here.
		return ast.InternedTerm(iter.ReadString()).Value
	case jsoniter.NumberValue:
		if m.config.UseNumber {
			// NOTE: this always allocates for reading the number as a string,
			// and contrary to actual strings, there is no way to read this as a byte slice :(
			jsonNum := iter.ReadNumber()
			if interned := ast.InternedIntNumberTermFromString(string(jsonNum)); interned != nil {
				return interned.Value
			}

			return ast.Number(jsonNum)
		}
		// NOTE: this, on the other hand, is not safe if e.g. "uncommon" or huge numbers are expected,
		// so only use this when you know the input is safe, and never on arbitrary user input
		jsonFloat := iter.ReadFloat64()
		if jsonInt := int(jsonFloat); jsonFloat == float64(jsonInt) {
			return ast.InternedValue(jsonInt)
		}

		return ast.Number(strconv.FormatFloat(jsonFloat, 'g', -1, 64))
	case jsoniter.NilValue:
		_ = iter.ReadNil()

		return ast.NullValue
	case jsoniter.BoolValue:
		return ast.InternedValue(iter.ReadBool())
	case jsoniter.ArrayValue:
		var terms []*ast.Term // we don't know the length here so no pre-alloc :/

		iter.ReadArrayCB(func(iter *jsoniter.Iterator) bool {
			elem := m.toValue(iter, iter.WhatIsNext())
			if elem == nil {
				return false
			}

			if terms == nil {
				terms = make([]*ast.Term, 0, 8) // pure guess, but better than appending to nil slice
			}

			terms = append(terms, internedValueToTerm(elem))

			return true
		})

		if terms == nil {
			return ast.InternedEmptyArrayValue
		}

		return ast.NewArray(terms...)
	case jsoniter.ObjectValue:
		var items [][2]*ast.Term // same as above, no pre-alloc

		iter.ReadObjectCB(func(iter *jsoniter.Iterator, field string) bool {
			value := m.toValue(iter, iter.WhatIsNext())
			if value == nil {
				return false
			}

			if items == nil {
				items = make([][2]*ast.Term, 0, 8) // as above, pure guess
			}

			items = append(items, rast.Item(field, internedValueToTerm(value)))

			return true
		})

		if items == nil {
			return ast.InternedEmptyObjectValue
		}

		return ast.NewObject(items...)
	case jsoniter.InvalidValue:
		iter.ReportError("Read", "invalid JSON value")
	}

	return nil
}

func internedValueToTerm(value ast.Value) *ast.Term {
	switch v := value.(type) {
	case ast.Null:
		return ast.InternedNullTerm
	case ast.Boolean:
		return ast.InternedTerm(bool(v))
	case ast.Number:
		if interned := ast.InternedIntNumberTermFromString(string(v)); interned != nil {
			return interned
		}
	case ast.String:
		return ast.InternedTerm(string(v))
	case *ast.Array:
		if v != nil && v.Len() == 0 {
			return ast.InternedEmptyArray
		}
	case ast.Object:
		if v != nil && v.Len() == 0 {
			return ast.InternedEmptyObject
		}
	}

	return ast.NewTerm(value)
}
