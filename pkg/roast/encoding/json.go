package encoding

import (
	"log"

	jsoniter "github.com/json-iterator/go"

	_ "github.com/open-policy-agent/regal/internal/roast/encoding"
	_ "github.com/open-policy-agent/regal/pkg/roast/intern"
)

// Fallback config in case the faster number handling fails.
// See: https://github.com/open-policy-agent/regal/issues/1592
var SafeNumberConfig = jsoniter.Config{
	UseNumber:                     true,
	EscapeHTML:                    false,
	MarshalFloatWith6Digits:       true,
	ObjectFieldMustBeSimpleString: true,
}.Froze()

// JSON returns the fastest jsoniter configuration
// It is preferred using this function instead of jsoniter.ConfigFastest directly
// as there as the init function needs to be called to register the custom types,
// which will happen automatically on import.
func JSON() jsoniter.API {
	return jsoniter.ConfigFastest
}

// JSONRoundTrip convert any value to JSON and back again.
func JSONRoundTrip(from any, to any) error {
	bs, err := jsoniter.ConfigFastest.Marshal(from)
	if err != nil {
		return err
	}

	if err = jsoniter.ConfigFastest.Unmarshal(bs, to); err != nil {
		return SafeNumberConfig.Unmarshal(bs, to)
	}

	return nil
}

func JSONRoundTripTo[T any](from any) (T, error) {
	var to T

	err := JSONRoundTrip(from, &to)

	return to, err
}

// MustJSONRoundTrip convert any value to JSON and back again, exit on failure.
func MustJSONRoundTrip(from any, to any) {
	if err := JSONRoundTrip(from, to); err != nil {
		log.Fatal(err)
	}
}
