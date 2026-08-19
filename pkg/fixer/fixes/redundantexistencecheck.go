package fixes

import (
	"errors"
	"strings"
)

type RedundantExistenceCheck struct{}

func (*RedundantExistenceCheck) Name() string {
	return "redundant-existence-check"
}

func (p *RedundantExistenceCheck) Fix(fc *FixCandidate, opts *RuntimeOptions) ([]FixResult, error) {
	if opts == nil {
		return nil, errors.New("missing runtime options")
	}

	lines, fixed := removeLocations(strings.Split(fc.Contents, "\n"), opts.Locations)
	if !fixed {
		return nil, nil
	}

	return []FixResult{{Title: p.Name(), Root: opts.BaseDir, Contents: strings.Join(lines, "\n")}}, nil
}
