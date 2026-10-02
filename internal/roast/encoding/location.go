package encoding

import (
	"encoding/json/jsontext"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"
)

func LocationMarshalToFn(enc *jsontext.Encoder, location *ast.Location) (err error) {
	endRow, endCol := location.End()

	buf := append(enc.AvailableBuffer(), '"')
	buf = append(util.AppendInt(buf, location.Row), ':')
	buf = append(util.AppendInt(buf, location.Col), ':')
	buf = append(util.AppendInt(buf, endRow), ':')
	buf = append(util.AppendInt(buf, endCol), '"')

	return enc.WriteValue(buf)
}
