package hints

import (
	"testing"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
)

func TestHints(t *testing.T) {
	t.Parallel()

	_, err := parse.Module("test.rego", "package foo\n\nincomplete")
	must.NotEqual(t, nil, err, "expected error")

	hints := must.Return(GetForError(err))(t)

	assert.SlicesEqual(t, []string{"rego-parse-error/var-cannot-be-used-for-rule-name"}, hints, "hints")
}
