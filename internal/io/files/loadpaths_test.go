package files

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/open-policy-agent/opa/v1/loader"

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

// The expanded paths must load into the same data layout as handing the
// directory to the OPA loader directly, minus the skipped directories.
func TestFilterOPALoadPathsKeepsDataLayout(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		files  map[string]string
		prefix string
	}{
		"sibling directories": {
			files: map[string]string{"a/data.json": `{"x": 1}`, "b/data.json": `{"y": 2}`},
		},
		"root and nested data": {
			files: map[string]string{
				"data.json":          `{"top": true}`,
				"a/b/data.yaml":      "deep: 1\n",
				"a/other.json":       `{"other": 2}`,
				"a/b/c/data.yml":     "z: 3\n",
				"policy/policy.rego": "package p\n\nallow := data.a.b.deep == 1\n",
			},
		},
		"modules and data together": {
			files: map[string]string{
				"p/policy.rego": "package p\n\nallow := true\n",
				"p/data.json":   `{"enabled": true}`,
			},
		},
		"user prefix": {
			files:  map[string]string{"a/data.json": `{"x": 1}`, "data.json": `{"y": 2}`},
			prefix: "outer.inner:",
		},
		"directory name with a dot": {
			files: map[string]string{"a.b/data.json": `{"x": 1}`, "c/data.json": `{"y": 2}`},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := testutil.TempDirectoryOf(t, tc.files)
			arg := tc.prefix + root

			want, err := loader.NewFileLoader().Filtered([]string{arg}, nil)
			if err != nil {
				t.Fatal(err)
			}

			paths, err := FilterOPALoadPaths([]string{arg})
			if err != nil {
				t.Fatal(err)
			}

			got, err := loader.NewFileLoader().Filtered(paths, nil)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(want.Documents, got.Documents) {
				t.Fatalf("documents: want %v, got %v", want.Documents, got.Documents)
			}

			if !reflect.DeepEqual(moduleNames(want), moduleNames(got)) {
				t.Fatalf("modules: want %v, got %v", moduleNames(want), moduleNames(got))
			}
		})
	}
}

// Skipped directories stay skipped even when they hold data files.
func TestFilterOPALoadPathsSkipsDataInIrrelevantDirectories(t *testing.T) {
	t.Parallel()

	root := testutil.TempDirectoryOf(t, map[string]string{
		"a/data.json":                   `{"x": 1}`,
		"node_modules/pkg/package.json": `{"name": "pkg"}`,
	})

	paths, err := FilterOPALoadPaths([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	got, err := loader.NewFileLoader().Filtered(paths, nil)
	if err != nil {
		t.Fatal(err)
	}

	clean := testutil.TempDirectoryOf(t, map[string]string{"a/data.json": `{"x": 1}`})

	want, err := loader.NewFileLoader().Filtered([]string{clean}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(want.Documents, got.Documents) {
		t.Fatalf("want %v, got %v", want.Documents, got.Documents)
	}
}

func moduleNames(r *loader.Result) []string {
	names := make([]string, 0, len(r.Modules))
	for _, m := range r.Modules {
		names = append(names, m.Name)
	}

	slices.Sort(names)

	return names
}
