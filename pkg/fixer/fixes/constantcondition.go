//nolint:dupl // Same implementation as redundant-existence-check fixer. Could consider refactoring later.
package fixes

import (
	"errors"
	"strings"
)

type ConstantCondition struct{}

func (*ConstantCondition) Name() string {
	return "constant-condition"
}

func (p *ConstantCondition) Fix(fc *FixCandidate, opts *RuntimeOptions) ([]FixResult, error) {
	if opts == nil {
		return nil, errors.New("missing runtime options")
	}

	lines := strings.Split(fc.Contents, "\n")
	fixed := false

	for _, loc := range opts.Locations {
		line := lines[loc.Row-1]

		if loc.Row > len(lines) || loc.Column-1 < 0 || loc.Column-1 >= len(line) {
			continue
		}

		startIndex := loc.Column - 1
		endIndex := loc.End.Column - 1

		lines[loc.Row-1] = line[0:startIndex] + line[endIndex:]

		fixed = true
	}

	if !fixed {
		return nil, nil
	}

	return []FixResult{{Title: p.Name(), Root: opts.BaseDir, Contents: strings.Join(lines, "\n")}}, nil
}
