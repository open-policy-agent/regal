package types

import "strconv"

const (
	ErrorMessage Message = iota + 1
	WarningMessage
	InfoMessage
	LogMessage
)

type Message uint8

func (m Message) AppendText(buf []byte) []byte {
	return strconv.AppendUint(buf, uint64(m), 10)
}
