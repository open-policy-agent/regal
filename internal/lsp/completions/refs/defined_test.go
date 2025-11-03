package refs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-policy-agent/regal/internal/lsp/rego"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	rparse "github.com/open-policy-agent/regal/internal/parse"
)

func TestForModule_Package(t *testing.T) {
	t.Parallel()

	mod := rparse.MustParseModule(`# METADATA
# title: An awesome package
# description: A package that's for things
# scope: package
# related_resources:
# - "https://example.com"
# - ref: "https://example.com/foobar"
#   description: "A related resource"
# authors:
# - Example Name
# - name: Foo
#   email: bar@example.com
# organizations:
# - Example Org
# custom:
#  foo: bar
#  tags: [a, b, c]
package example
`)

	bis := rego.BuiltinsForDefaultCapabilities()
	items := DefinedInModule(mod, bis)

	expectedRefs := map[string]types.Ref{
		"data.example": {
			Label: "data.example",
			Kind:  types.Package,
			Description: fmt.Sprintf(`# An awesome package

**Description**:

A package that's for things

**Authors**:

* Example Name
* Foo <bar@example.com>

**Organizations**:

* Example Org

**Related Resources**:

* [https://example.com](https://example.com)
* [A related resource](https://example.com/foobar)

**Custom**:

%s
foo: bar
tags:
    - a
    - b
    - c
%s
`, "```yaml", "```"),
		},
	}

	for key, item := range expectedRefs {
		if _, ok := items[key]; !ok {
			t.Errorf("missing expected item %s", key)

			continue
		}

		if items[key].Label != item.Label {
			t.Errorf("expected label %q, got %q", item.Label, items[key].Label)
		}

		if items[key].Kind != item.Kind {
			t.Errorf("expected documentation kind %q, got %q", item.Description, items[key].Description)
		}

		if items[key].Description != item.Description {
			t.Errorf("expected description\n%s\ngot\n%s", item.Description, items[key].Description)
		}
	}

	if len(items) != len(expectedRefs) {
		t.Errorf("expected %d items, got %d", len(expectedRefs), len(items))
	}
}

func TestRefsForModule_Rules(t *testing.T) {
	t.Parallel()

	mod := rparse.MustParseModule(`package example

# METADATA
# title: An allow rule
# description: A rule for things that should be allowed
# scope: rule
default allow := false

allow if input.foo

# METADATA
# title: A funky function
# description: A function that's really funky
# scope: rule
funkyfunc(x) := true

deny contains "strings" if true

pi := 3.14
`)
	bis := rego.BuiltinsForDefaultCapabilities()
	items := DefinedInModule(mod, bis)

	expectedRefs := map[string]types.Ref{
		"data.example": {
			Label:       "data.example",
			Kind:        types.Package,
			Detail:      "Package",
			Description: "# example",
		},
		"data.example.allow": {
			Label:  "data.example.allow",
			Kind:   types.Rule,
			Detail: "single-value rule (boolean)",
			Description: `# An allow rule

**Description**:

A rule for things that should be allowed
`,
		},
		"data.example.funkyfunc": {
			Label:  "data.example.funkyfunc",
			Kind:   types.Function,
			Detail: "function(x)",
			Description: `# A funky function

**Description**:

A function that's really funky`,
		},
		"data.example.deny": {
			Label:       "data.example.deny",
			Kind:        types.Rule,
			Detail:      "multi-value rule",
			Description: "# deny",
		},
		"data.example.pi": {
			Label:       "data.example.pi",
			Kind:        types.ConstantRule,
			Detail:      "single-value rule (number)",
			Description: "# pi",
		},
	}

	for key, item := range expectedRefs {
		if _, ok := items[key]; !ok {
			t.Errorf("missing expected item %s", key)

			continue
		}

		if items[key].Label != item.Label {
			t.Errorf("expected label %q, got %q", item.Label, items[key].Label)
		}

		if items[key].Kind != item.Kind {
			t.Errorf("expected kind %q, got %q", item.Kind, items[key].Kind)
		}

		if items[key].Detail != item.Detail {
			t.Errorf("expected detail %q, got %q", item.Detail, items[key].Detail)
		}

		if !strings.Contains(items[key].Description, item.Description) {
			t.Errorf("expected description\n%s\ngot\n%s", item.Description, items[key].Description)
		}
	}

	if len(items) != len(expectedRefs) {
		t.Errorf("expected %d items, got %d", len(expectedRefs), len(items))
	}
}
