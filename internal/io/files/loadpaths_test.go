package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterOPALoadPathsSkipsIrrelevantDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "policy.rego"), []byte("package p\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "data.json"), []byte("{}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("docs\n"), 0o600))

	for _, dir := range []string{".git", ".idea", "node_modules"} {
		nested := filepath.Join(root, dir, "nested")
		require.NoError(t, os.MkdirAll(nested, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(nested, "ignored.rego"), []byte("package ignored\n"), 0o600))
	}

	got, err := FilterOPALoadPaths([]string{root})
	require.NoError(t, err)

	var rels []string
	for _, p := range got {
		rel, err := filepath.Rel(root, p)
		require.NoError(t, err)
		rels = append(rels, rel)
	}

	assert.ElementsMatch(t, []string{"policy.rego", "data.json"}, rels)
}

func TestFilterOPALoadPathsKeepsExplicitFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "policy.rego")
	require.NoError(t, os.WriteFile(file, []byte("package p\n"), 0o600))

	got, err := FilterOPALoadPaths([]string{file})
	require.NoError(t, err)
	assert.Equal(t, []string{file}, got)
}
