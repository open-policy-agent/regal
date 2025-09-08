package web

import (
	"bytes"
	"strings"
	"testing"

	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestTemplateFoundAndParsed(t *testing.T) {
	t.Parallel()

	buf := bytes.Buffer{}
	testutil.NoErr(tpl.ExecuteTemplate(&buf, mainTemplate, state{Code: "package main\n\nimport rego.v1\n"}))(t)

	if !strings.HasPrefix(buf.String(), "<!DOCTYPE html>") {
		t.Fatalf("expected HTML document, got %s", buf.String())
	}
}
