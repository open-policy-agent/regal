//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	equals      = makeVerifier("equals", false, func(act, exp string) bool { return act == exp })
	hasPrefix   = makeVerifier("prefix", false, strings.HasPrefix)
	hasSuffix   = makeVerifier("suffix", false, strings.HasSuffix)
	contains    = makeVerifier("contains", false, strings.Contains)
	notContains = makeVerifier("not contains", true, strings.Contains)
)

type runner struct {
	cmd       []string
	stdout    *bytes.Buffer
	stderr    *bytes.Buffer
	stdin     io.Reader
	exp       result
	dir       string
	skipFn    func(t *testing.T)
	cleanFn   func()
	stdoutFns []verifier
	stderrFns []verifier
	filesFns  []verifier
	t         *testing.T // only to keep track of t.Parallel being set
}

type result struct {
	exitCode int
	stdout   string
	stderr   string
}

type exitStatus interface {
	ExitStatus() int
}

type verifier func(*testing.T, string, string)

func regal(args ...string) runner {
	return runner{
		stdout:    &bytes.Buffer{},
		stderr:    &bytes.Buffer{},
		cmd:       args,
		stdoutFns: []verifier{equals("")}, // default expectation is no output on stdout
		stderrFns: []verifier{equals("")}, // default expectation is no output on stderr
	}
}

// regal as method — only used when running multiple regal commands in a single test,
// and we need to keep the original test context in order to not have multiple calls
// to t.Parallel(), as that causes panics.
func (r runner) regal(args ...string) runner {
	nr := regal(args...)
	nr.t = r.t // keep the original test context
	return nr
}

func (r runner) cleanup(f func()) runner {
	r.cleanFn = f
	return r
}

func (r runner) stdinFrom(in io.Reader) runner {
	r.stdin = in
	return r
}

func (r runner) expectExitCode(code int) runner {
	r.exp.exitCode = code
	return r
}

func (r runner) inDirectory(path string) runner {
	r.dir = path
	return r
}

func (r runner) skip(f func(t *testing.T)) runner {
	r.skipFn = f
	return r
}

func (r runner) expectStdout(f ...verifier) runner {
	r.stdoutFns = f
	return r
}

func (r runner) expectStderr(f ...verifier) runner {
	r.stderrFns = f
	return r
}

func (r runner) expectFiles(f ...verifier) runner {
	r.filesFns = f
	return r
}

func makeVerifier(name string, negated bool, f func(act, exp string) bool) func(exp string, vars ...any) verifier {
	return func(exp string, vars ...any) verifier {
		return func(t *testing.T, context string, act string) {
			t.Helper()

			if len(vars) > 0 {
				exp = fmt.Sprintf(exp, vars...)
			}

			if f(act, exp) == negated {
				if exp == "" {
					exp = "<no output>"
				}
				t.Errorf("\n%s: %s check failed\nexpected:\n%s\ngot:\n%s", context, name, exp, act)
			}
		}
	}
}

func notEmpty() verifier {
	return func(t *testing.T, context, act string) {
		t.Helper()

		if act == "" {
			t.Errorf("%s: not empty check failed\nexpected some output, got nothing", context)
		}
	}
}

func unmarshalsTo(v any) verifier {
	return func(t *testing.T, context, act string) {
		t.Helper()

		if err := json.Unmarshal([]byte(act), v); err != nil {
			t.Errorf("%s: JSON unmarshal failed for input %s\n%v", context, act, err)
		}
	}
}

func exists(path ...string) verifier {
	return func(t *testing.T, context, act string) {
		t.Helper()

		if _, err := os.Stat(filepath.Join(path...)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("%s: expected file %s to exist, but it does not", context, path)
			} else {
				t.Fatalf("%s: unexpected error checking file %s: %v", context, path, err)
			}
		}
	}
}

func notExists(path ...string) verifier {
	return func(t *testing.T, context, act string) {
		t.Helper()

		if _, err := os.Stat(filepath.Join(path...)); err == nil {
			t.Errorf("%s: expected file or directory %s to not exist, but it does", context, path)
		}
	}
}

func hasContent(path, content string) verifier {
	return func(t *testing.T, context, act string) {
		t.Helper()

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: unexpected error reading file %s: %v", context, path, err)
		}
		if string(data) != content {
			t.Errorf("%s: expected file %s with content\n%q, but got\n%s", context, path, content, string(data))
		}
	}
}

func all(verifiers ...verifier) verifier {
	return func(t *testing.T, context, act string) {
		t.Helper()

		for _, v := range verifiers {
			v(t, context, act)
		}
	}
}

func contentMatchesMap(root string, m map[string]string) verifier {
	return func(t *testing.T, context, act string) {
		t.Helper()

		for file, content := range m {
			path := filepath.Join(root, file)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: unexpected error reading file %s: %v", context, path, err)
			}
			if string(data) != content {
				t.Errorf("%s: expected file %s with content\n%q, but got\n%s", context, path, content, string(data))
			}
		}
	}
}

// test allows passing check directly to (*testing.T).Run, et. al.
func (r runner) test(t *testing.T) {
	t.Helper()
	r.verify(t)
}

func (r runner) verify(t *testing.T) runner {
	t.Helper()

	if r.t == nil {
		t.Parallel()
		r.t = t
	}

	if r.cleanFn != nil {
		t.Cleanup(r.cleanFn)
	}

	if r.skipFn != nil {
		r.skipFn(t)
	}

	c := exec.Command(r.binary(), r.cmd...)

	c.Dir = r.dir
	c.Stdin = r.stdin
	c.Stdout = r.stdout
	c.Stderr = r.stderr

	res := result{exitCode: exitCode(c.Run()), stdout: r.stdout.String(), stderr: r.stderr.String()}

	if res.exitCode != r.exp.exitCode {
		t.Errorf("expected exit status %d, got %d\nstdout:\n%s\nstderr:\n%s",
			r.exp.exitCode, res.exitCode, r.stdout, r.stderr,
		)
	}

	for _, f := range r.stdoutFns {
		f(t, "stdout", res.stdout)
	}
	for _, f := range r.stderrFns {
		f(t, "stderr", res.stderr)
	}
	for _, f := range r.filesFns {
		f(t, "files", "") // "" because there is no "act"-ual output for files
	}

	return r
}

func (r runner) binary() string {
	location := "../regal"
	if r.dir != "" {
		location = filepath.Join(r.dir, "regal")
	}

	if runtime.GOOS == "windows" {
		location += ".exe"
	}

	if b := os.Getenv("REGAL_BIN"); b != "" {
		location = b
	}

	if _, err := os.Stat(location); errors.Is(err, os.ErrNotExist) {
		log.Fatal("regal binary not found — make sure to run go build before running the e2e tests")
	} else if err != nil {
		log.Fatal(err)
	}

	return location
}

// exitCode returns the exit status of exec.ExitError's or errors implementing
// exitCode() int, 0 if it is nil and panics on different errors.
func exitCode(err error) int {
	switch e := err.(type) { //nolint:errorlint // We know the errors that can happen here, the switch is enough.
	case nil:
		return 0
	case exitStatus:
		return e.ExitStatus()
	case *exec.ExitError:
		if ex, ok := e.Sys().(exitStatus); ok {
			return ex.ExitStatus()
		}
	}

	panic(err)
}
