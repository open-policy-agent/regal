package fixer

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	outil "github.com/open-policy-agent/opa/v1/util"
)

var re = regexp.MustCompile(`^(.*)_(\d+)$`)

// renameCandidate takes a filename and produces a new name with an incremented
// numeric suffix. It correctly handles test files by inserting the increment
// before the "_test" suffix and preserves the original directory.
func renameCandidate(oldName string) string {
	dir, baseWithExt := filepath.Split(oldName)
	ext := filepath.Ext(baseWithExt)
	base, isTest := strings.CutSuffix(strings.TrimSuffix(baseWithExt, ext), "_test")

	var suffix string
	if isTest {
		suffix = "_test"
	}

	matches := re.FindStringSubmatch(base)
	if len(matches) == 3 {
		baseName := matches[1]
		num, _ := outil.Atoi(matches[2])
		num++
		base = fmt.Sprintf("%s_%d", baseName, num)
	} else {
		base += "_1"
	}

	return filepath.Join(dir, base+suffix+ext)
}
