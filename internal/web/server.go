package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/arl/statsviz"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/util"
)

type Server struct {
	log     *log.Logger
	baseURL string
}

var pprofEndpoints = os.Getenv("REGAL_DEBUG") != "" || os.Getenv("REGAL_DEBUG_PPROF") != ""

func NewServer(logger *log.Logger) *Server {
	return &Server{log: logger}
}

func (s *Server) GetBaseURL() string {
	return s.baseURL
}

// SetBaseURL sets the base URL for the server
// NOTE: This is normally set by the server itself, and this method is provided only for testing purposes.
func (s *Server) SetBaseURL(baseURL string) {
	s.baseURL = baseURL
}

func (s *Server) Start(context.Context) {
	mux := http.NewServeMux()
	if err := statsviz.Register(mux); err != nil {
		s.log.Message("failed to register statsviz handler: %v", err)
	}

	if pprofEndpoints {
		mux.HandleFunc("GET /debug/pprof/", pprof.Index)
		mux.Handle("GET /debug/pprof/allocs", pprof.Handler("allocs"))
		mux.Handle("GET /debug/pprof/block", pprof.Handler("block"))
		mux.Handle("GET /debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("GET /debug/pprof/mutex", pprof.Handler("mutex"))
		mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	}

	// root handler for those looking for what the server is
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		body := `
<h1>Regal Language Server</h1>
<ul>`

		if pprofEndpoints {
			body += `<li><a href="/debug/pprof/">pprof</a></li>
<li><a href="/debug/statsviz">statsviz</a></li>
</ul>`
		} else {
			body += `Start server with REGAL_DEBUG or REGAL_DEBUG_PPROF set to enable pprof endpoints`
		}

		if _, err := w.Write([]byte(body)); err != nil {
			s.log.Message("failed to write response: %v", err)
		}
	})

	freePort, err := util.FreePort(5052, 5053, 5054)
	if err != nil {
		s.log.Message("preferred web server ports are not available, using random port")

		freePort = 0
	}

	var lc net.ListenConfig

	//nolint:contextcheck
	listener, err := lc.Listen(context.Background(), "tcp", fmt.Sprintf("localhost:%d", freePort))
	if err != nil {
		s.log.Message("failed to start web server: %v", err)

		return
	}

	s.baseURL = "http://" + listener.Addr().String()

	s.log.Message("starting web server for docs on %s", s.baseURL)

	//nolint:gosec // this is a local server, no timeouts needed
	if err = http.Serve(listener, mux); err != nil {
		s.log.Message("failed to serve web server: %v", err)
	}
}
