package fixes

import (
	"errors"
	"strings"
)

type NonRawRegexPattern struct{}

func (*NonRawRegexPattern) Name() string {
	return "non-raw-regex-pattern"
}

func (u *NonRawRegexPattern) Fix(fc *FixCandidate, opts *RuntimeOptions) ([]FixResult, error) {
	if opts == nil {
		return nil, errors.New("missing runtime options")
	}

	if len(opts.Locations) == 0 {
		return []FixResult{}, nil
	}

	lines := strings.Split(fc.Contents, "\n")
	fileChanged := false

	for _, loc := range opts.Locations {
		if loc.Row-1 < 0 || loc.Row-1 >= len(lines) {
			continue
		}

		line := []rune(lines[loc.Row-1])
		startIdx := loc.Column - 1
		endIdx := loc.End.Column - 2

		if startIdx < 0 || endIdx > len(line) || startIdx >= endIdx {
			continue
		}

		if line[startIdx] == '"' {
			line[startIdx] = '`'
			fileChanged = true
		}

		if line[endIdx] == '"' {
			line[endIdx] = '`'
			fileChanged = true
		}

		// Replace "\\" with "\" between startIdx and endIdx
		segment := strings.ReplaceAll(string(line[startIdx:endIdx]), `\\`, `\`)
		replacement := []rune(segment)

		lines[loc.Row-1] = string(append(line[:startIdx], append(replacement, line[endIdx:]...)...))
	}

	if !fileChanged {
		return []FixResult{}, nil
	}

	return []FixResult{{Title: u.Name(), Root: opts.BaseDir, Contents: strings.Join(lines, "\n")}}, nil
}
