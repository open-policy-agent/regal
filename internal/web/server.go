package web

import (
	"cmp"
	"context"
	"embed"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"slices"
	"strings"

	"github.com/arl/statsviz"

	"github.com/open-policy-agent/regal/internal/explorer"
	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/util"
)

type (
	Server struct {
		cache        *cache.Cache
		log          *log.Logger
		workspaceURI string
		baseURL      string
		client       clients.Identifier
	}

	stringResult struct {
		Class  string
		Stage  string
		Output string
		Show   bool
	}

	state struct {
		Code   string
		Result []stringResult
	}
)

const mainTemplate = "main.tpl"

var (
	//go:embed assets/*
	assets         embed.FS
	tpl            = template.Must(template.New(mainTemplate).ParseFS(assets, "assets/main.tpl"))
	pprofEndpoints = os.Getenv("REGAL_DEBUG") != "" || os.Getenv("REGAL_DEBUG_PPROF") != ""
)

func NewServer(cache *cache.Cache, logger *log.Logger) *Server {
	return &Server{cache: cache, log: logger}
}

func (s *Server) SetWorkspaceURI(uri string) {
	s.workspaceURI = uri
}

func (s *Server) SetClient(client clients.Identifier) {
	s.client = client
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

	mux.HandleFunc("GET /explorer/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/explorer/")

		policy, ok := s.cache.GetFileContents(s.workspaceURI + "/" + path)
		if !ok {
			http.NotFound(w, r)

			return
		}

		var (
			enableStrict, enableAnnotationProcessing, enablePrint bool // TODO(sr): expose
			hideIdentical                                         bool
			tmpl                                                  string
		)

		if err := r.ParseForm(); err == nil {
			enableStrict = r.Form.Get("strict") == "on"
			enableAnnotationProcessing = r.Form.Get("annotations") == "on"
			enablePrint = r.Form.Get("print") == "on"
			hideIdentical = r.Form.Get("hide_identical") == "on"
			tmpl = cmp.Or(r.Form.Get("tmpl"), mainTemplate)
		}

		cs := explorer.CompilerStages(path, policy, enableStrict, enableAnnotationProcessing, enablePrint)
		n := len(cs)
		st := state{Code: policy, Result: make([]stringResult, n+1)}

		for i := range cs {
			show := i == 0 || st.Result[i-1].Output != cs[i].Result.String()

			st.Result[i] = stringResult{Stage: cs[i].Stage, Show: show}

			if cs[i].Error != "" {
				st.Result[i].Output = cs[i].Error
				st.Result[i].Class = "bad"
			} else {
				st.Result[i].Output = cs[i].Result.String()
			}

			if st.Result[i].Class == "" {
				if show {
					st.Result[i].Class = "ok"
				} else {
					st.Result[i].Class = "plain"
				}
			}
		}

		st.Result[n] = stringResult{Stage: "Plan", Show: true}
		if p, err := explorer.Plan(r.Context(), path, policy, enablePrint); err != nil {
			st.Result[n].Class = "bad"
			st.Result[n].Output = err.Error()
		} else {
			st.Result[n].Class = "ok"
			st.Result[n].Output = p
		}

		if hideIdentical {
			st.Result = slices.CompactFunc(st.Result, func(a, b stringResult) bool {
				return a.Output == b.Output
			})
		}

		if err := tpl.ExecuteTemplate(w, tmpl, st); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.Handle("/assets/", http.FileServer(http.FS(assets)))

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
