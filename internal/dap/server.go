package dap

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/open-policy-agent/opa/v1/debug"

	"github.com/open-policy-agent/regal/internal/dap/evaluate"
	"github.com/open-policy-agent/regal/internal/util"
)

type Server struct {
	Listener        net.Listener
	evaluateHandler evaluate.Handler
	addr            string
	log             *DebugLogger
	port            uint16
}

// NewServer creates a new debug server that listens for incoming connections on addr,
// represented as "host:port". If the port is missing or 0, a random port will be picked.
func NewServer(addr string, logger *DebugLogger) *Server {
	return &Server{addr: addr, log: logger, port: parsePort(addr)}
}

func (s *Server) WithEvaluateHandler(f evaluate.Handler) *Server {
	s.evaluateHandler = f

	return s
}

func (s *Server) Port() uint16 {
	return s.port
}

func (s *Server) Start(ctx context.Context) (err error) {
	lc := &net.ListenConfig{}
	if s.Listener, err = lc.Listen(ctx, "tcp4", s.addr); err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.port = parsePort(s.Listener.Addr().String())
	s.log.Local.Info("starting dap server on 127.0.0.1:%d", s.port)

	for {
		if conn, err := s.Listener.Accept(); err == nil {
			s.log.Local.Info("new connection from %s", conn.RemoteAddr())

			protoManager := NewProtocolManager(s.log.Local)
			s.log.ProtocolManager = protoManager

			state := NewState(
				protoManager,
				debug.NewDebugger(debug.SetEventHandler(NewEventHandler(protoManager)), debug.SetLogger(s.log)),
				s.log,
			).WithEvaluateHandler(s.evaluateHandler)

			if err := protoManager.Start(ctx, conn, state.HandleMessage); err != nil {
				s.log.Local.Error("failed to handle connection: %v", err)
			}
		} else {
			s.log.Local.Error("failed to accept connection: %v", err)
		}
	}
}

func (s *Server) Close() (err error) {
	if s.Listener != nil {
		err = s.Listener.Close()
	}

	return util.WrapErr(err, "failed to close debug server")
}

func parsePort(addr string) (port uint16) {
	if strings.Contains(addr, ":") {
		if portint, _ := strconv.Atoi(strings.SplitN(addr, ":", 2)[1]); portint > 0 && portint <= 65535 {
			port = uint16(portint)
		}
	}

	return port
}

func NewEventHandler(pm *ProtocolManager) debug.EventHandler {
	return func(e debug.Event) {
		switch e.Type {
		case debug.ExceptionEventType:
			pm.SendEvent(NewStoppedExceptionEvent(e.Thread, e.Message))
		case debug.StdoutEventType:
			pm.SendEvent(NewOutputEvent("stdout", e.Message))
		case debug.StoppedEventType:
			pm.SendEvent(NewStoppedEvent(e.Message, e.Thread, nil, "", ""))
		case debug.TerminatedEventType:
			pm.SendEvent(NewTerminatedEvent())
		case debug.ThreadEventType:
			pm.SendEvent(NewThreadEvent(e.Thread, e.Message))
		}
	}
}
