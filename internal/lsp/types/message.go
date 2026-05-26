package types

import (
	"github.com/open-policy-agent/regal/internal/util"
)

const (
	ErrorMessage Message = iota + 1
	WarningMessage
	InfoMessage
	LogMessage
)

type Message uint8

func (m Message) AppendText(buf []byte) []byte {
	return util.AppendUint(buf, m)
}
