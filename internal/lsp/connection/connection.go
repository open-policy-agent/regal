// The implementation of logMessages, is a heavily modified version of the original implementation
// in https://github.com/sourcegraph/jsonrpc2
// The original license for that code is as follows:
// Copyright (c) 2016 Sourcegraph Inc
//
// # MIT License
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
// documentation files (the "Software"), to deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or substantial portions
// of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO
// THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package connection

import (
	"cmp"
	"context"
	"os"
	"slices"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

type (
	HandlerFunc func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (result any, err error)
	logHandler  func(*jsonrpc2.Request, *jsonrpc2.Response)

	LoggingConfig struct {
		Logger *log.Logger

		// IncludeMethods is a list of methods to include in the request log.
		// If empty, all methods are included. IncludeMethods takes precedence
		// over ExcludeMethods.
		IncludeMethods []string
		// ExcludeMethods is a list of methods to exclude from the request log.
		ExcludeMethods []string

		LogInbound  bool
		LogOutbound bool
	}

	Options struct {
		LoggingConfig LoggingConfig
	}
)

func (cfg *LoggingConfig) ShouldLog(method string) bool {
	if len(cfg.IncludeMethods) > 0 {
		return slices.Contains(cfg.IncludeMethods, method)
	}

	return !slices.Contains(cfg.ExcludeMethods, method)
}

func New(ctx context.Context, handler HandlerFunc, opts *Options) *jsonrpc2.Conn {
	stream := jsonrpc2.NewBufferedStream(io.NewReadWriteCloser(os.Stdin, os.Stdout), jsonrpc2.VSCodeObjectCodec{})
	asynch := jsonrpc2.AsyncHandler(jsonrpc2.HandlerWithError(handler))

	return jsonrpc2.NewConn(ctx, stream, asynch, logMessages(opts.LoggingConfig))
}

func logMessages(cfg LoggingConfig) jsonrpc2.ConnOpt {
	return func(c *jsonrpc2.Conn) {
		// Remember reqs received so that we can show the request method in responses.
		reqMethods := concurrent.MapOf(make(map[jsonrpc2.ID]string))

		if cfg.LogInbound {
			jsonrpc2.OnRecv(buildRecvHandler(reqMethods.Set, cfg))(c)
		}

		if cfg.LogOutbound {
			jsonrpc2.OnSend(buildSendHandler(reqMethods.GetUnchecked, reqMethods.Delete, cfg))(c)
		}
	}
}

func buildRecvHandler(setMethod func(jsonrpc2.ID, string), cfg LoggingConfig) logHandler {
	return func(req *jsonrpc2.Request, resp *jsonrpc2.Response) {
		switch {
		case req != nil && resp == nil:
			setMethod(req.ID, req.Method)
			logRequest(cfg, req)
		case resp != nil:
			method := "(no matching request)"
			if req != nil {
				method = req.Method
			}

			logResponse(cfg, resp, method)
		}
	}
}

func buildSendHandler(getFn func(jsonrpc2.ID) string, deleteFn func(jsonrpc2.ID), cfg LoggingConfig) logHandler {
	return func(req *jsonrpc2.Request, resp *jsonrpc2.Response) {
		switch {
		case req != nil && resp == nil:
			logRequest(cfg, req)
		case resp != nil:
			deleteFn(resp.ID)
			logResponse(cfg, resp, cmp.Or(getFn(resp.ID), "(no previous request)"))
		}
	}
}

func logRequest(cfg LoggingConfig, req *jsonrpc2.Request) {
	if !cfg.ShouldLog(req.Method) {
		return
	}

	params, _ := encoding.JSON().Marshal(req.Params)
	if req.Notif {
		cfg.Logger.Message("--> notif: %s: %s\n", req.Method, params)
	} else {
		cfg.Logger.Message("--> request #%s: %s: %s\n", req.ID, req.Method, params)
	}
}

func logResponse(cfg LoggingConfig, resp *jsonrpc2.Response, method string) {
	if !cfg.ShouldLog(method) {
		return
	}

	if resp.Result != nil {
		result, _ := encoding.JSON().Marshal(resp.Result)
		cfg.Logger.Message("<-- response #%s: %s: %s\n", resp.ID, method, result)
	} else {
		errBs, _ := encoding.JSON().Marshal(resp.Error)
		cfg.Logger.Message("<-- response error #%s: %s: %s\n", resp.ID, method, errBs)
	}
}
