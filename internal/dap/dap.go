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
			pm.logDebug("Sending", resp)

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

			pm.logDebug("Received", req)

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
		if r.Success = err == nil; !r.Success {
			r.Message = err.Error()
		}

		if r.Seq = pm.nextSeq(); req != nil {
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

func (pm *ProtocolManager) logDebug(event string, message godap.Message) {
	if pm.logger.GetLevel() == logging.Debug {
		if msgData, err := json.Marshal(message); msgData != nil && err == nil {
			pm.logger.Debug("%s %T\n%s", event, message, msgData)
		} else {
			pm.logger.Debug("%s %T", event, message)
		}
	}
}

func NewContinueResponse() *godap.ContinueResponse {
	return &godap.ContinueResponse{Response: createResponse("continue", true)}
}

func NewNextResponse() *godap.NextResponse {
	return &godap.NextResponse{Response: createResponse("next", true)}
}

func NewStepInResponse() *godap.StepInResponse {
	return &godap.StepInResponse{Response: createResponse("stepIn", true)}
}

func NewStepOutResponse() *godap.StepOutResponse {
	return &godap.StepOutResponse{Response: createResponse("stepOut", true)}
}

func NewInitializeResponse(capabilities godap.Capabilities) *godap.InitializeResponse {
	return &godap.InitializeResponse{
		Response: createResponse("initialize", true),
		Body:     capabilities,
	}
}

func NewAttachResponse() *godap.AttachResponse {
	return &godap.AttachResponse{Response: createResponse("attach", true)}
}

func NewBreakpointLocationsResponse(breakpoints []godap.BreakpointLocation) *godap.BreakpointLocationsResponse {
	return &godap.BreakpointLocationsResponse{
		Response: createResponse("breakpointLocations", true),
		Body:     godap.BreakpointLocationsResponseBody{Breakpoints: breakpoints},
	}
}

func NewSetBreakpointsResponse(breakpoints []godap.Breakpoint) *godap.SetBreakpointsResponse {
	return &godap.SetBreakpointsResponse{
		Response: createResponse("setBreakpoints", true),
		Body:     godap.SetBreakpointsResponseBody{Breakpoints: breakpoints},
	}
}

func NewConfigurationDoneResponse() *godap.ConfigurationDoneResponse {
	return &godap.ConfigurationDoneResponse{Response: createResponse("configurationDone", true)}
}

func NewDisconnectResponse() *godap.DisconnectResponse {
	return &godap.DisconnectResponse{Response: createResponse("disconnect", true)}
}

func NewLaunchResponse() *godap.LaunchResponse {
	return &godap.LaunchResponse{Response: createResponse("launch", true)}
}

func NewScopesResponse(scopes []godap.Scope) *godap.ScopesResponse {
	return &godap.ScopesResponse{
		Response: createResponse("scopes", true),
		Body:     godap.ScopesResponseBody{Scopes: scopes},
	}
}

func NewStackTraceResponse(stack []godap.StackFrame) *godap.StackTraceResponse {
	return &godap.StackTraceResponse{
		Response: createResponse("stackTrace", true),
		Body:     godap.StackTraceResponseBody{StackFrames: stack, TotalFrames: len(stack)},
	}
}

func NewTerminateResponse() *godap.TerminateResponse {
	return &godap.TerminateResponse{Response: createResponse("terminate", true)}
}

func NewThreadsResponse(threads []godap.Thread) *godap.ThreadsResponse {
	return &godap.ThreadsResponse{
		Response: createResponse("threads", true),
		Body:     godap.ThreadsResponseBody{Threads: threads},
	}
}

func NewVariablesResponse(variables []godap.Variable) *godap.VariablesResponse {
	return &godap.VariablesResponse{
		Response: createResponse("variables", true),
		Body:     godap.VariablesResponseBody{Variables: variables},
	}
}

// Events

func NewInitializedEvent() *godap.InitializedEvent {
	return &godap.InitializedEvent{Event: createEvent("initialized")}
}

func NewOutputEvent(category, output string) *godap.OutputEvent {
	return &godap.OutputEvent{
		Event: createEvent("output"),
		Body:  godap.OutputEventBody{Output: output, Category: category},
	}
}

func NewThreadEvent(threadID debug.ThreadID, reason string) *godap.ThreadEvent {
	return &godap.ThreadEvent{
		Event: createEvent("thread"),
		Body:  godap.ThreadEventBody{Reason: reason, ThreadId: int(threadID)},
	}
}

func NewTerminatedEvent() *godap.TerminatedEvent {
	return &godap.TerminatedEvent{Event: createEvent("terminated")}
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

func NewStoppedEvent(reason string, id debug.ThreadID, bps []int, description, text string) *godap.StoppedEvent {
	return &godap.StoppedEvent{
		Event: godap.Event{
			ProtocolMessage: godap.ProtocolMessage{Type: "event"},
			Event:           "stopped",
		},
		Body: godap.StoppedEventBody{
			Reason:            reason,
			ThreadId:          int(id),
			Text:              text,
			Description:       description,
			AllThreadsStopped: true,
			HitBreakpointIds:  bps,
			PreserveFocusHint: false,
		},
	}
}

func createResponse(command string, success bool) godap.Response { //nolint:unparam
	return godap.Response{
		ProtocolMessage: godap.ProtocolMessage{Type: "response"},
		Command:         command,
		Success:         success,
	}
}

func createEvent(event string) godap.Event {
	return godap.Event{
		ProtocolMessage: godap.ProtocolMessage{Type: "event"},
		Event:           event,
	}
}
