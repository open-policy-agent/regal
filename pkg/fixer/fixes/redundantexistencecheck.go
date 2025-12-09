package fixes

import (
	"errors"
	"slices"
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

	lines := strings.Split(fc.Contents, "\n")
	fixed := false

	var newLines []string

	removedLines := make([]int, 0, 100)

	for _, loc := range opts.Locations {
		index := loc.Row - 1
		line := lines[index]

		if loc.Row > len(lines) || loc.Column-1 < 0 || loc.Column-1 >= len(line) {
			continue
		}

		removedLines = append(removedLines, index)
	}

	if len(removedLines) == 0 {
		newLines = lines
	} else {
		for line := range lines {
			if !slices.Contains(removedLines, line) {
				newLines = append(newLines, lines[line])
				fixed = true
			}
		}
	}

	if !fixed {
		return nil, nil
	}

	return []FixResult{{Title: p.Name(), Root: opts.BaseDir, Contents: strings.Join(newLines, "\n")}}, nil
}
