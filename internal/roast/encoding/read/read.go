package read

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

func Token(dec *jsontext.Decoder, kind jsontext.Kind) (jsontext.Token, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return jsontext.Null, err
	}

	if tok.Kind() != kind {
		return jsontext.Null, fmt.Errorf("expected token kind %v, got %v", kind, tok.Kind())
	}

	return tok, nil
}

func String(dec *jsontext.Decoder) (string, error) {
	tok, err := Token(dec, jsontext.KindString)
	if err != nil {
		return "", err
	}

	return tok.String(), nil
}

func Object(dec *jsontext.Decoder, fn func(*jsontext.Decoder, string) error) error {
	tok, err := Token(dec, '{') //nolint // 'this value of tok is never used' — yes it is
	for err == nil {
		if tok, err = dec.ReadToken(); err != nil || tok.Kind() == '}' {
			break
		}

		err = fn(dec, tok.String())
	}

	return err
}

func Unknown(dec *jsontext.Decoder, key string) error {
	if reject, _ := json.GetOption(dec.Options(), json.RejectUnknownMembers); reject {
		return fmt.Errorf("rejecting unknown key: %s", key)
	}

	return dec.SkipValue()
}
