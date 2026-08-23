package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateBuiltinDocsIncludesConfigurationFileLocations(t *testing.T) {
	t.Parallel()

	output := t.TempDir()

	err := createBuiltinDocs(newRuleCommandParams{
		category: "naming",
		name:     "foo-bar-baz",
		output:   output,
	})
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(output, "docs", "rules", "naming", "foo-bar-baz.md")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	doc := string(content)
	for _, expected := range []string{
		"[configuration options](https://www.openpolicyagent.org/projects/regal/configuration)",
		"```yaml title=\".regal/config.yaml or .regal.yaml\"",
	} {
		if !strings.Contains(doc, expected) {
			t.Errorf("generated documentation does not contain %q", expected)
		}
	}
}
