package encoding

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"strconv"
	"strings"

	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/util"

	_ "github.com/open-policy-agent/regal/pkg/roast/intern"
)

// JSONUnmarshalTo is a silly convenience function over json.Unmarshal.
func JSONUnmarshalTo[T any](bs []byte, opts ...json.Options) (to T, err error) {
	err = json.Unmarshal(bs, &to, opts...)

	return to, err
}

// JSONRoundTrip convert any value to JSON and back again.
func JSONRoundTrip(from, to any, opts ...json.Options) error {
	bs, err := json.Marshal(from, opts...)
	if err != nil {
		return err
	}

	// For now, use OPA's unmarshalling for this as we don't
	// want to replicate AST unmarshalling here
	// For now, use OPA's unmarshalling for this as we don't
	// want to replicate AST unmarshalling here
	return json.Unmarshal(bs, to, opts...)
}

// JSONRoundTripTo convert any value to JSON and back again, returning the new value or an error.
func JSONRoundTripTo[T any](from any, opts ...json.Options) (to T, err error) {
	err = JSONRoundTrip(from, &to, opts...)

	return to, err
}

// MustJSONRoundTrip convert any value to JSON and back again, exit on failure.
func MustJSONRoundTrip(from, to any, opts ...json.Options) {
	util.PanicOnErr(JSONRoundTrip(from, to, opts...))
}

func Get(value jsontext.Value, ptr jsontext.Pointer) jsontext.Value {
	path := strings.TrimPrefix(string(ptr), "/")
	key, rest, _ := strings.Cut(path, "/")

	switch value.Kind() {
	case jsontext.KindBeginObject:
		return Get(getInObject(value, key), jsontext.Pointer(rest))
	case jsontext.KindBeginArray:
		return nil // TODO: implement
	default:
		return value
	}
}

func GetString(value jsontext.Value, ptr jsontext.Pointer) (string, bool) {
	if val := Get(value, ptr); val.Kind() == jsontext.KindString {
		str, _ := strconv.Unquote(string(val))

		return str, true
	}

	return "", false
}

// MarshalWriteLn is a convenience function json.MarshalWrite followed by a newline.
func MarshalWriteLn(w io.Writer, v any, opts ...json.Options) (err error) {
	if err = json.MarshalWrite(w, v, opts...); err == nil {
		_, err = w.Write(outil.StringToByteSlice("\n"))
	}

	return err
}

func getInObject(obj jsontext.Value, key string) (res jsontext.Value) {
	dec := jsontext.NewDecoder(bytes.NewReader(obj))
	tok, err := dec.ReadToken() //nolint

	for err == nil {
		if tok, err = dec.ReadToken(); err != nil || tok.Kind() == '}' {
			break
		}

		if tok.String() == key {
			res, _ = dec.ReadValue()

			break
		}

		err = dec.SkipValue()
	}

	return res.Clone()
}
