package fixes

import (
	"errors"
	"strings"
)

type PreferEqualsComparison struct{}

func (*PreferEqualsComparison) Name() string {
	return "prefer-equals-comparison"
}

func (p *PreferEqualsComparison) Fix(fc *FixCandidate, opts *RuntimeOptions) ([]FixResult, error) {
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

		// unexpected character at location column, skipping
		if line[loc.Column-1] != '=' {
			continue
		}

		lines[loc.Row-1] = line[0:loc.Column-1] + "=" + line[loc.Column-1:]
		fixed = true
	}

	if !fixed {
		return nil, nil
	}

	return []FixResult{{Title: p.Name(), Root: opts.BaseDir, Contents: strings.Join(lines, "\n")}}, nil
}
