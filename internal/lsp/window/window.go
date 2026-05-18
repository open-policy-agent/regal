package window

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

// Window provides convenience methods for sending window-related notifications and requests to the client.
// Errors are logged using the provided logger but they are *not* returned, as there is typically nothing
// meaningful the server can do with them other than to log them, and handling them clutter the call sites.
type Window struct {
	conn *jsonrpc2.Conn
	log  *log.Logger
}

func New(conn *jsonrpc2.Conn, log *log.Logger) *Window {
	return &Window{conn: conn, log: log}
}

// ShowMessage sends a message to be displayed in the user's editor.
func (w *Window) ShowMessage(ctx context.Context, typ types.Message, msg string) {
	if err := w.conn.Notify(ctx, "window/showMessage", rawMessageParams(typ, msg)); err != nil {
		w.log.Message("window/showMessage notify failed: %s", err)
	}
}

// ShowMessageRequest sends a message to be displayed in the user's editor, along with a set of actions
// for them to choose from. The returned string is the title of the action selected.
func (w *Window) ShowMessageRequest(ctx context.Context, typ types.Message, msg string, actions ...string) string {
	var res struct {
		Title string `json:"title"`
	}
	if err := w.conn.Call(ctx, "window/showMessageRequest", rawMessageParams(typ, msg, actions...), &res); err != nil {
		w.log.Message("window/showMessageRequest call failed: %s", err)
	}

	return res.Title
}

// ShowDocument attempts to open the given URI in the user's editor.
func (w *Window) ShowDocument(ctx context.Context, uri string, takeFocus bool) bool {
	var res struct {
		Success bool `json:"success"`
	}

	if err := w.conn.Call(ctx, "window/showDocument", rawShowDocumentParams(uri, takeFocus), &res); err != nil {
		w.log.Message("window/showDocument failed: %s", err)
	}

	return res.Success
}

func rawShowDocumentParams(uri string, takeFocus bool) *json.RawMessage {
	alloc := 10 + len(uri) // {"uri":""}
	if takeFocus {
		alloc += 17 // ,"takeFocus":true
	}

	buf := strconv.AppendQuote(append(make([]byte, 0, alloc), `{"uri":`...), uri)
	if takeFocus {
		buf = append(buf, `,"takeFocus":true`...)
	}

	return new(json.RawMessage(append(buf, '}')))
}

func rawMessageParams(typ types.Message, msg string, actions ...string) *json.RawMessage {
	alloc := 23 + len(msg)
	if len(actions) > 0 {
		alloc += 12 // ,"actions":[], minus one comma
		for _, action := range actions {
			alloc += len(action) + 13 // ,{"title":""}
		}
	}

	buf := typ.AppendText(append(make([]byte, 0, alloc), `{"type":`...))
	buf = strconv.AppendQuote(append(buf, `,"message":`...), msg)

	if len(actions) > 0 {
		buf = append(strconv.AppendQuote(append(append(buf, `,"actions":[`...), `{"title":`...), actions[0]), '}')
		for _, action := range actions[1:] {
			buf = append(strconv.AppendQuote(append(buf, `,{"title":`...), action), '}')
		}

		buf = append(buf, ']')
	}

	return new(json.RawMessage(append(buf, '}')))
}
