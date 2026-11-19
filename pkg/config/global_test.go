package config_test

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestFindGlobal(t *testing.T) {
	// Not parallel, as the home directory is set.
	home, project := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // the home directory on Windows

	globalFile := filepath.Join(home, ".config", "regal", "config.yaml")
	projectFile := filepath.Join(project, ".regal.yaml")

	found, err := config.Find(project, true)
	must.Equal(t, nil, err)
	assert.Equal(t, "", found, "no config at all")

	// A global config directory without a config file is the same as having no config.
	must.MkdirAll(t, filepath.Dir(globalFile))

	found, err = config.Find(project, true)
	must.Equal(t, nil, err)
	assert.Equal(t, "", found, "no global config file")

	must.WriteFile(t, globalFile, []byte("rules: {}"))

	// The global config is only found when asked for, so the ignore rule command does not edit it.
	found, err = config.Find(project, true)
	must.Equal(t, nil, err)
	assert.Equal(t, globalFile, found)

	found, err = config.Find(project, false)
	must.Equal(t, nil, err)
	assert.Equal(t, "", found, "global config left out")

	must.WriteFile(t, projectFile, []byte("rules: {}"))

	for _, global := range []bool{true, false} {
		found, err = config.Find(project, global)
		must.Equal(t, nil, err)
		assert.Equal(t, projectFile, found, "project config preferred")
	}

	// A problem with the project config is an error, and not a reason to use the global config.
	must.MkdirAll(t, project, ".regal")
	must.WriteFile(t, filepath.Join(project, ".regal", "config.yaml"), []byte("rules: {}"))

	_, err = config.Find(project, true)
	must.NotEqual(t, nil, err)
	assert.StringContains(t, err.Error(), "conflicting config files")
}
