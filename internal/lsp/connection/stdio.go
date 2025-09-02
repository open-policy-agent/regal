package connection

import (
	"os"

	"github.com/open-policy-agent/regal/internal/util"
)

type StdOutReadWriteCloser struct{}

func (StdOutReadWriteCloser) Read(p []byte) (int, error) {
	return util.Wrap(os.Stdin.Read(p))("failed to read from stdin")
}

func (StdOutReadWriteCloser) Write(p []byte) (int, error) {
	return util.Wrap(os.Stdout.Write(p))("failed to write to stdout")
}

func (StdOutReadWriteCloser) Close() error {
	return nil
}
