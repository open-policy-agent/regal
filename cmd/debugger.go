package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"

	godap "github.com/google/go-dap"
	"github.com/spf13/cobra"

	"github.com/open-policy-agent/opa/v1/ast/location"
	"github.com/open-policy-agent/opa/v1/debug"
	"github.com/open-policy-agent/opa/v1/logging"

	"github.com/open-policy-agent/regal/internal/dap"
	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/builtins"
)

func init() {
	verboseLogging, serverMode := false, false
	address := "localhost:4712"

	debug := &cobra.Command{
		Use:   "debug",
		Short: "Run the Regal OPA Debugger",
		Long:  `Start the Regal OPA debugger and listen on stdin/stdout for client editor messages.`,

		RunE: wrapProfiling(func([]string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			logger := dap.NewDebugLogger(logging.New(), logging.Debug)
			if verboseLogging {
				logger.Local.SetLevel(logging.Debug)
			}

			if serverMode {
				return startServer(ctx, address, logger)
			}

			return startCmd(ctx, logger)
		}),
	}

	debug.Flags().BoolVarP(&verboseLogging, "verbose", "v", verboseLogging, "Enable verbose logging")
	debug.Flags().BoolVarP(&serverMode, "server", "s", serverMode, "Start the debugger in server mode")
	debug.Flags().StringVarP(&address, "address", "a", address, "Address to listen on. For use with --server flag.")

	RootCommand.AddCommand(debug)
}

func startCmd(ctx context.Context, logger *dap.DebugLogger) error {
	protoManager := dap.NewProtocolManager(logger.Local)
	logger.ProtocolManager = protoManager

	debugger := debug.NewDebugger(debug.SetEventHandler(newEventHandler(protoManager)), debug.SetLogger(logger))
	conn := io.NewReadWriteCloser(os.Stdin, os.Stdout)
	s := newState(protoManager, debugger, logger)

	return util.WrapErr(protoManager.Start(ctx, conn, s.messageHandler), "failed to handle connection")
}

func startServer(ctx context.Context, address string, logger *dap.DebugLogger) error {
	var lc net.ListenConfig

	l, err := lc.Listen(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("could not listen: %w", err)
	}

	logger.Local.Info("Listening on %s", address)

	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("could not accept: %w", err)
		}

		logger.Local.Info("New connection from %s", conn.RemoteAddr())

		go func() {
			defer func() {
				if err := conn.Close(); err != nil {
					logger.Local.Error("Error closing connection: %v", err)
				}

				logger.Local.Info("Connection closed")
			}()

			protoManager := dap.NewProtocolManager(logger.Local)
			logger.ProtocolManager = protoManager
			debugger := debug.NewDebugger(debug.SetEventHandler(newEventHandler(protoManager)), debug.SetLogger(logger))

			s := newState(protoManager, debugger, logger)
			if err := protoManager.Start(ctx, conn, s.messageHandler); err != nil {
				logger.Local.Error("Failed to handle connection: %v", err)
			}

			logger.Local.Info("Closing connection...")
		}()
	}
}

type state struct {
	protocolManager    *dap.ProtocolManager
	debugger           debug.Debugger
	session            debug.Session
	logger             *dap.DebugLogger
	serverCapabilities *godap.Capabilities
	clientCapabilities *godap.InitializeRequestArguments
}

func newState(protocolManager *dap.ProtocolManager, debugger debug.Debugger, logger *dap.DebugLogger) *state {
	return &state{
		protocolManager: protocolManager,
		debugger:        debugger,
		logger:          logger,
		serverCapabilities: &godap.Capabilities{
			SupportsBreakpointLocationsRequest:    true,
			SupportsCancelRequest:                 true,
			SupportsConfigurationDoneRequest:      true,
			SupportsSingleThreadExecutionRequests: true,
			SupportSuspendDebuggee:                true,
			SupportTerminateDebuggee:              true,
			SupportsTerminateRequest:              true,
		},
	}
}

func newEventHandler(pm *dap.ProtocolManager) debug.EventHandler {
	return func(e debug.Event) {
		switch e.Type {
		case debug.ExceptionEventType:
			pm.SendEvent(dap.NewStoppedExceptionEvent(e.Thread, e.Message))
		case debug.StdoutEventType:
			pm.SendEvent(dap.NewOutputEvent("stdout", e.Message))
		case debug.StoppedEventType:
			pm.SendEvent(dap.NewStoppedEvent(e.Message, e.Thread, nil, "", ""))
		case debug.TerminatedEventType:
			pm.SendEvent(dap.NewTerminatedEvent())
		case debug.ThreadEventType:
			pm.SendEvent(dap.NewThreadEvent(e.Thread, e.Message))
		}
	}
}

func (s *state) messageHandler(ctx context.Context, message godap.Message) (bool, godap.ResponseMessage, error) {
	var (
		resp godap.ResponseMessage
		err  error
	)

	switch request := message.(type) {
	case *godap.AttachRequest:
		resp = dap.NewAttachResponse()
		err = errors.New("attach not supported")
	case *godap.BreakpointLocationsRequest:
		resp = s.breakpointLocations(request)
	case *godap.ConfigurationDoneRequest:
		err = s.start()
		resp = dap.NewConfigurationDoneResponse()
	case *godap.ContinueRequest:
		resp, err = s.resume(request)
	case *godap.DisconnectRequest:
		return true, dap.NewDisconnectResponse(), nil
	case *godap.EvaluateRequest:
		resp, err = s.evaluate(request)
	case *godap.InitializeRequest:
		resp = s.initialize(request)
	case *godap.LaunchRequest:
		resp, err = s.launch(ctx, request)
	case *godap.NextRequest:
		resp, err = s.next(request)
	case *godap.ScopesRequest:
		resp, err = s.scopes(request)
	case *godap.SetBreakpointsRequest:
		resp, err = s.setBreakpoints(request)
	case *godap.StackTraceRequest:
		resp, err = s.stackTrace(request)
	case *godap.StepInRequest:
		resp, err = s.stepIn(request)
	case *godap.StepOutRequest:
		resp, err = s.stepOut(request)
	case *godap.TerminateRequest:
		resp, err = s.terminate(request)
	case *godap.ThreadsRequest:
		resp, err = s.threads(request)
	case *godap.VariablesRequest:
		resp, err = s.variables(request)
	default:
		s.logger.Warn("Handler not found for request: %T", message)
		err = fmt.Errorf("handler not found for request: %T", message)
	}

	return false, resp, err
}

func (s *state) initialize(r *godap.InitializeRequest) *godap.InitializeResponse {
	if args, err := json.Marshal(r.Arguments); err == nil {
		s.logger.Info("Initializing: %s", args)
	} else {
		s.logger.Info("Initializing")
	}

	s.clientCapabilities = &r.Arguments

	return dap.NewInitializeResponse(*s.serverCapabilities)
}

type launchProperties struct {
	Command  string `json:"command"`
	LogLevel string `json:"logLevel"`
}

func (s *state) launch(ctx context.Context, r *godap.LaunchRequest) (*godap.LaunchResponse, error) {
	var props launchProperties
	if err := json.Unmarshal(r.Arguments, &props); err != nil {
		return dap.NewLaunchResponse(), fmt.Errorf("invalid launch properties: %w", err)
	}

	if props.LogLevel != "" {
		s.logger.SetLevelFromString(props.LogLevel)
	} else {
		s.logger.SetRemoteEnabled(false)
	}

	s.logger.Info("Launching: %s", props)

	var err error

	switch props.Command {
	case "eval":
		var evalProps debug.LaunchEvalProperties
		if err := json.Unmarshal(r.Arguments, &evalProps); err != nil {
			return dap.NewLaunchResponse(), fmt.Errorf("invalid launch eval properties: %w", err)
		}

		funcs := util.Map(builtins.RegalBuiltinRegoFuncs, debug.RegoOption)

		// FIXME: Should we protect this with a mutex?
		s.session, err = s.debugger.LaunchEval(ctx, evalProps, funcs...)
	case "test":
		err = errors.New("test not supported")
	case "":
		err = errors.New("missing launch command")
	default:
		err = fmt.Errorf("unsupported launch command: '%s'", props.Command)
	}

	if err == nil {
		//	err = s.session.ResumeAll()
		s.protocolManager.SendEvent(dap.NewInitializedEvent())
	}

	return dap.NewLaunchResponse(), err
}

func (s *state) start() error {
	return util.WrapErr(s.session.ResumeAll(), "failed to start debug session")
}

func (*state) evaluate(_ *godap.EvaluateRequest) (*godap.EvaluateResponse, error) {
	return dap.NewEvaluateResponse(""), errors.New("evaluate not supported")
}

func (s *state) resume(r *godap.ContinueRequest) (*godap.ContinueResponse, error) {
	return dap.NewContinueResponse(), s.session.Resume(debug.ThreadID(r.Arguments.ThreadId))
}

func (s *state) next(r *godap.NextRequest) (*godap.NextResponse, error) {
	return dap.NewNextResponse(), s.session.StepOver(debug.ThreadID(r.Arguments.ThreadId))
}

func (s *state) stepIn(r *godap.StepInRequest) (*godap.StepInResponse, error) {
	return dap.NewStepInResponse(), s.session.StepIn(debug.ThreadID(r.Arguments.ThreadId))
}

func (s *state) stepOut(r *godap.StepOutRequest) (*godap.StepOutResponse, error) {
	return dap.NewStepOutResponse(), s.session.StepOut(debug.ThreadID(r.Arguments.ThreadId))
}

func (s *state) threads(_ *godap.ThreadsRequest) (*godap.ThreadsResponse, error) {
	var threads []godap.Thread

	ts, err := s.session.Threads()
	if err == nil {
		for _, t := range ts {
			threads = append(threads, godap.Thread{Id: int(t.ID()), Name: t.Name()})
		}
	}

	return dap.NewThreadsResponse(threads), err
}

func (s *state) stackTrace(r *godap.StackTraceRequest) (*godap.StackTraceResponse, error) {
	var stackFrames []godap.StackFrame

	fs, err := s.session.StackTrace(debug.ThreadID(r.Arguments.ThreadId))
	if err == nil {
		for _, f := range fs {
			var source *godap.Source

			source, line, col, endLine, endCol := pos(f.Location())
			stackFrames = append(stackFrames, godap.StackFrame{
				Id:               int(f.ID()),
				Name:             f.Name(),
				Source:           source,
				Line:             line,
				Column:           col,
				EndLine:          endLine,
				EndColumn:        endCol,
				PresentationHint: "normal",
			})
		}
	}

	return dap.NewStackTraceResponse(stackFrames), err
}

func pos(loc *location.Location) (source *godap.Source, line, col, endLine, endCol int) {
	if loc == nil {
		return nil, 1, 0, 1, 0
	}

	if loc.File != "" {
		source = &godap.Source{Path: loc.File}
	}

	lines := bytes.Split(loc.Text, []byte("\n"))

	// vs-code will select text if multiple lines are present; avoid this
	// endLine = loc.Row + len(lines) - 1
	// endCol = col + len(lines[len(lines)-1])
	return source, loc.Row, loc.Col, loc.Row, loc.Col + len(lines[0])
}

func (s *state) scopes(r *godap.ScopesRequest) (*godap.ScopesResponse, error) {
	var scopes []godap.Scope

	ss, err := s.session.Scopes(debug.FrameID(r.Arguments.FrameId))
	if err == nil {
		for _, s := range ss {
			var source *godap.Source

			line := 1

			if loc := s.Location(); loc != nil {
				line = loc.Row

				if loc.File != "" {
					source = &godap.Source{Path: loc.File}
				}
			}

			scopes = append(scopes, godap.Scope{
				Name:               s.Name(),
				NamedVariables:     s.NamedVariables(),
				VariablesReference: int(s.VariablesReference()),
				Source:             source,
				Line:               line,
			})
		}
	}

	return dap.NewScopesResponse(scopes), err
}

func (s *state) variables(r *godap.VariablesRequest) (*godap.VariablesResponse, error) {
	var variables []godap.Variable

	vs, err := s.session.Variables(debug.VarRef(r.Arguments.VariablesReference))
	if err == nil {
		for _, v := range vs {
			variables = append(variables, godap.Variable{
				Name:               v.Name(),
				Value:              v.Value(),
				Type:               v.Type(),
				VariablesReference: int(v.VariablesReference()),
			})
		}
	}

	return dap.NewVariablesResponse(variables), err
}

func (s *state) breakpointLocations(request *godap.BreakpointLocationsRequest) *godap.BreakpointLocationsResponse {
	line := request.Arguments.Line
	s.logger.Debug("Breakpoint locations requested for: %s:%d", request.Arguments.Source.Name, line)

	// TODO: Actually assert where breakpoints can be placed.
	return dap.NewBreakpointLocationsResponse([]godap.BreakpointLocation{{
		Line:   line,
		Column: 1,
	}})
}

func (s *state) setBreakpoints(request *godap.SetBreakpointsRequest) (*godap.SetBreakpointsResponse, error) {
	bps, err := s.session.Breakpoints()
	if err != nil {
		return dap.NewSetBreakpointsResponse(nil), err
	}

	// Remove all breakpoints for the given source.
	for _, bp := range bps {
		if bp.Location().File != request.Arguments.Source.Path {
			continue
		}

		if _, err := s.session.RemoveBreakpoint(bp.ID()); err != nil {
			return dap.NewSetBreakpointsResponse(nil), err
		}
	}

	breakpoints := make([]godap.Breakpoint, 0, len(request.Arguments.Breakpoints))

	for _, sbp := range request.Arguments.Breakpoints {
		loc := location.Location{File: request.Arguments.Source.Path, Row: sbp.Line}

		bp, err := s.session.AddBreakpoint(loc)
		if err != nil {
			return dap.NewSetBreakpointsResponse(breakpoints), err
		}

		breakpoints = append(breakpoints, godap.Breakpoint{
			Id:       int(bp.ID()),
			Source:   &godap.Source{Path: loc.File},
			Line:     bp.Location().Row,
			Verified: true,
		})
	}

	return dap.NewSetBreakpointsResponse(breakpoints), err
}

func (s *state) terminate(_ *godap.TerminateRequest) (*godap.TerminateResponse, error) {
	return dap.NewTerminateResponse(), s.session.Terminate()
}
