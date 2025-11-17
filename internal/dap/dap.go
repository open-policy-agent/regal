package dap

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	godap "github.com/google/go-dap"

	"github.com/open-policy-agent/opa/v1/debug"
	"github.com/open-policy-agent/opa/v1/logging"
)

type MessageHandler func(ctx context.Context, request godap.Message) (bool, godap.ResponseMessage, error)

type ProtocolManager struct {
	inChan  chan godap.Message
	outChan chan godap.Message
	logger  logging.Logger
	seq     int
	seqLock sync.Mutex
}

func NewProtocolManager(logger logging.Logger) *ProtocolManager {
	return &ProtocolManager{
		inChan:  make(chan godap.Message),
		outChan: make(chan godap.Message),
		logger:  logger,
	}
}

func (pm *ProtocolManager) Start(ctx context.Context, conn io.ReadWriteCloser, handle MessageHandler) error {
	reader := bufio.NewReader(conn)
	done := make(chan error)

	go func() {
		for resp := range pm.outChan {
			if pm.logger.GetLevel() == logging.Debug {
				if respData, err := json.Marshal(resp); respData != nil && err == nil {
					pm.logger.Debug("Sending %T\n%s", resp, respData)
				} else {
					pm.logger.Debug("Sending %T", resp)
				}
			}

			if err := godap.WriteProtocolMessage(conn, resp); err != nil {
				done <- err

				return
			}
		}
	}()

	go func() {
		for {
			pm.logger.Debug("Waiting for message...")

			req, err := godap.ReadProtocolMessage(reader)
			if err != nil {
				done <- err

				return
			}

			if pm.logger.GetLevel() == logging.Debug {
				if reqData, err := json.Marshal(req); reqData != nil && err == nil {
					pm.logger.Debug("Received %T\n%s", req, reqData)
				} else {
					pm.logger.Debug("Received %T", req)
				}
			}

			stop, resp, err := handle(ctx, req)
			if err != nil {
				pm.logger.Warn("Error handling request: %v", err)
			}

			pm.SendResponse(resp, req, err)

			if stop {
				done <- err

				return
			}
		}
	}()

	for {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return fmt.Errorf("context closed: %w", ctx.Err())
		}
	}
}

func (pm *ProtocolManager) SendEvent(e godap.EventMessage) {
	e.GetEvent().Seq = pm.nextSeq()
	pm.outChan <- e
}

func (pm *ProtocolManager) SendResponse(resp godap.ResponseMessage, req godap.Message, err error) {
	if resp == nil {
		return
	}

	if r := resp.GetResponse(); r != nil {
		r.Success = err == nil
		if err != nil {
			r.Message = err.Error()
		}

		r.Seq = pm.nextSeq()
		if req != nil {
			r.RequestSeq = req.GetSeq()
		}
	}

	pm.outChan <- resp
}

func (pm *ProtocolManager) Close() {
	close(pm.outChan)
	close(pm.inChan)
}

func (pm *ProtocolManager) nextSeq() int {
	if pm == nil {
		return 0
	}

	pm.seqLock.Lock()
	defer pm.seqLock.Unlock()

	pm.seq++

	return pm.seq
}

func NewContinueResponse() *godap.ContinueResponse {
	return &godap.ContinueResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "continue",
			Success: true,
		},
	}
}

func NewNextResponse() *godap.NextResponse {
	return &godap.NextResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "next",
			Success: true,
		},
	}
}

func NewStepInResponse() *godap.StepInResponse {
	return &godap.StepInResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "stepIn",
			Success: true,
		},
	}
}

func NewStepOutResponse() *godap.StepOutResponse {
	return &godap.StepOutResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "stepOut",
			Success: true,
		},
	}
}

func NewInitializeResponse(capabilities godap.Capabilities) *godap.InitializeResponse {
	return &godap.InitializeResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "initialize",
			Success: true,
		},
		Body: capabilities,
	}
}

func NewAttachResponse() *godap.AttachResponse {
	return &godap.AttachResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "attach",
			Success: true,
		},
	}
}

func NewBreakpointLocationsResponse(breakpoints []godap.BreakpointLocation) *godap.BreakpointLocationsResponse {
	return &godap.BreakpointLocationsResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "breakpointLocations",
			Success: true,
		},
		Body: godap.BreakpointLocationsResponseBody{
			Breakpoints: breakpoints,
		},
	}
}

func NewSetBreakpointsResponse(breakpoints []godap.Breakpoint) *godap.SetBreakpointsResponse {
	return &godap.SetBreakpointsResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "setBreakpoints",
			Success: true,
		},
		Body: godap.SetBreakpointsResponseBody{
			Breakpoints: breakpoints,
		},
	}
}

func NewConfigurationDoneResponse() *godap.ConfigurationDoneResponse {
	return &godap.ConfigurationDoneResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "configurationDone",
			Success: true,
		},
	}
}

func NewDisconnectResponse() *godap.DisconnectResponse {
	return &godap.DisconnectResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "disconnect",
			Success: true,
		},
	}
}

func NewEvaluateResponse(value string) *godap.EvaluateResponse {
	return &godap.EvaluateResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "evaluate",
			Success: true,
		},
		Body: godap.EvaluateResponseBody{
			Result: value,
		},
	}
}

func NewLaunchResponse() *godap.LaunchResponse {
	return &godap.LaunchResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "launch",
			Success: true,
		},
	}
}

func NewScopesResponse(scopes []godap.Scope) *godap.ScopesResponse {
	return &godap.ScopesResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "scopes",
			Success: true,
		},
		Body: godap.ScopesResponseBody{
			Scopes: scopes,
		},
	}
}

func NewStackTraceResponse(stack []godap.StackFrame) *godap.StackTraceResponse {
	return &godap.StackTraceResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "stackTrace",
			Success: true,
		},
		Body: godap.StackTraceResponseBody{
			StackFrames: stack,
			TotalFrames: len(stack),
		},
	}
}

func NewTerminateResponse() *godap.TerminateResponse {
	return &godap.TerminateResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "terminate",
			Success: true,
		},
	}
}

func NewThreadsResponse(threads []godap.Thread) *godap.ThreadsResponse {
	return &godap.ThreadsResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "threads",
			Success: true,
		},
		Body: godap.ThreadsResponseBody{
			Threads: threads,
		},
	}
}

func NewVariablesResponse(variables []godap.Variable) *godap.VariablesResponse {
	return &godap.VariablesResponse{
		Response: godap.Response{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "response",
			},
			Command: "variables",
			Success: true,
		},
		Body: godap.VariablesResponseBody{
			Variables: variables,
		},
	}
}

// Events

func NewInitializedEvent() *godap.InitializedEvent {
	return &godap.InitializedEvent{
		Event: godap.Event{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "event",
			},
			Event: "initialized",
		},
	}
}

func NewOutputEvent(category, output string) *godap.OutputEvent {
	return &godap.OutputEvent{
		Event: godap.Event{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "event",
			},
			Event: "output",
		},
		Body: godap.OutputEventBody{
			Output:   output,
			Category: category,
		},
	}
}

func NewThreadEvent(threadID debug.ThreadID, reason string) *godap.ThreadEvent {
	return &godap.ThreadEvent{
		Event: godap.Event{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "event",
			},
			Event: "thread",
		},
		Body: godap.ThreadEventBody{
			Reason:   reason,
			ThreadId: int(threadID),
		},
	}
}

func NewTerminatedEvent() *godap.TerminatedEvent {
	return &godap.TerminatedEvent{
		Event: godap.Event{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "event",
			},
			Event: "terminated",
		},
	}
}

func NewStoppedEntryEvent(threadID debug.ThreadID) *godap.StoppedEvent {
	return NewStoppedEvent("entry", threadID, nil, "", "")
}

func NewStoppedExceptionEvent(threadID debug.ThreadID, text string) *godap.StoppedEvent {
	return NewStoppedEvent("exception", threadID, nil, "", text)
}

func NewStoppedResultEvent(threadID debug.ThreadID) *godap.StoppedEvent {
	return NewStoppedEvent("result", threadID, nil, "", "")
}

func NewStoppedBreakpointEvent(threadID debug.ThreadID, bp *godap.Breakpoint) *godap.StoppedEvent {
	return NewStoppedEvent("breakpoint", threadID, []int{bp.Id}, "", "")
}

func NewStoppedEvent(reason string, threadID debug.ThreadID, bps []int, description string,
	text string,
) *godap.StoppedEvent {
	return &godap.StoppedEvent{
		Event: godap.Event{
			ProtocolMessage: godap.ProtocolMessage{
				Type: "event",
			},
			Event: "stopped",
		},
		Body: godap.StoppedEventBody{
			Reason:            reason,
			ThreadId:          int(threadID),
			Text:              text,
			Description:       description,
			AllThreadsStopped: true,
			HitBreakpointIds:  bps,
			PreserveFocusHint: false,
		},
	}
}
