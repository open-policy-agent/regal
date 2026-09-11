package command

import (
	"context"
	"time"
)

// rpcTimeout allows requests to complete independently from the server's ctx,
// supporting graceful shutdown rather than immediate cancellation.
var rpcTimeout = 3 * time.Second

type Command interface {
	Run(context.Context) error
}
