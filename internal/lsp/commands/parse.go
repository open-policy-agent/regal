package commands

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/pkg/report"
)

type ParseOptions struct {
	TargetArgIndex int
	RowArgIndex    int
	ColArgIndex    int
}

type ParseResult struct {
	Location *report.Location
	Target   string
}

// Parse is responsible for extracting the target and location from the given params command params sent from the client
// after acting on a Code Action.
func Parse(params types.ExecuteCommandParams, opts ParseOptions) (*ParseResult, error) {
	numArgs := len(params.Arguments)
	if numArgs == 0 {
		return nil, errors.New("no args supplied")
	}

	target := ""
	if opts.TargetArgIndex < numArgs {
		target = fmt.Sprintf("%s", params.Arguments[opts.TargetArgIndex])
	}

	// we can't extract a location from the same location as the target, so location arg positions
	// must not have been set in the opts.
	if opts.RowArgIndex == opts.TargetArgIndex {
		return &ParseResult{Target: target}, nil
	}

	var loc *report.Location

	if opts.RowArgIndex < numArgs && opts.ColArgIndex < numArgs {
		var row, col int

		switch v := params.Arguments[opts.RowArgIndex].(type) {
		case int:
			row = v
		case string:
			var err error
			if row, err = strconv.Atoi(v); err != nil {
				return nil, fmt.Errorf("failed to parse row: %w", err)
			}
		default:
			return nil, fmt.Errorf("unexpected type for row: %T", params.Arguments[opts.RowArgIndex])
		}

		switch v := params.Arguments[opts.ColArgIndex].(type) {
		case int:
			col = v
		case string:
			var err error
			if col, err = strconv.Atoi(v); err != nil {
				return nil, fmt.Errorf("failed to parse col: %w", err)
			}
		default:
			return nil, fmt.Errorf("unexpected type for col: %T", params.Arguments[opts.ColArgIndex])
		}

		loc = &report.Location{Row: row, Column: col}
	}

	return &ParseResult{
		Target:   target,
		Location: loc,
	}, nil
}
