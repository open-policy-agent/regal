package files

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestFilterOPALoadPathsSkipsIrrelevantDirectories(t *testing.T) {
	t.Parallel()

	root := testutil.TempDirectoryOf(t, map[string]string{
		"policy.rego":                      "package p\n",
		"data.json":                        "{}\n",
		"README.md":                        "docs\n",
		".git/nested/ignored.rego":         "package ignored\n",
		".idea/nested/ignored.rego":        "package ignored\n",
		"node_modules/nested/ignored.rego": "package ignored\n",
	})

	got, err := FilterOPALoadPaths([]string{root})
	if err != nil {
		t.Fatalf("FilterOPALoadPaths: %v", err)
	}

	rels := make([]string, 0, len(got))
	for _, p := range got {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatalf("filepath.Rel(%q, %q): %v", root, p, err)
		}

		rels = append(rels, rel)
	}

	slices.Sort(rels)

	if want := []string{"data.json", "policy.rego"}; !slices.Equal(rels, want) {
		t.Fatalf("expected %v, got %v", want, rels)
	}
}

func TestFilterOPALoadPathsKeepsExplicitFiles(t *testing.T) {
	t.Parallel()

	root := testutil.TempDirectoryOf(t, map[string]string{"policy.rego": "package p\n"})
	file := filepath.Join(root, "policy.rego")

	got, err := FilterOPALoadPaths([]string{file})
	if err != nil {
		t.Fatalf("FilterOPALoadPaths: %v", err)
	}

	if want := []string{file}; !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
