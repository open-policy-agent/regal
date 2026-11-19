package config_test

import (
	"path/filepath"
	"testing"

	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/pkg/config"
)

func TestLoadUser(t *testing.T) {
	// Not parallel, as the home directory is set.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // the home directory on Windows

	const (
		valid      = "rules: {}"
		invalid    = "rules: [x"
		standalone = ".regal.yaml"
		dirConfig  = ".regal/config.yaml"
	)

	// Files and paths are relative to the project.
	tests := map[string]struct {
		files   map[string]string
		given   string
		want    string
		wantErr string
	}{
		"no config":            {},
		"given config":         {files: map[string]string{"given.yaml": valid}, given: "given.yaml", want: "given.yaml"},
		"given config missing": {given: "missing.yaml", want: "missing.yaml", wantErr: "missing.yaml"},
		"found config":         {files: map[string]string{standalone: valid}, want: standalone},
		"empty config":         {files: map[string]string{standalone: ""}, want: standalone},
		"conflicting config":   {files: map[string]string{standalone: valid, dirConfig: valid}, wantErr: "conflicting"},
		"invalid config":       {files: map[string]string{standalone: invalid}, want: standalone, wantErr: standalone},
	}

	for desc, tc := range tests { //nolint:paralleltest // the home directory is set
		t.Run(desc, func(t *testing.T) {
			project := t.TempDir()

			for file, content := range tc.files {
				must.MkdirAll(t, project, filepath.Dir(file))
				must.WriteFile(t, filepath.Join(project, file), []byte(content))
			}

			var given, want string
			if tc.given != "" {
				given = filepath.Join(project, tc.given)
			}

			if tc.want != "" {
				want = filepath.Join(project, tc.want)
			}

			cfg, path, err := config.LoadUser(given, project, false)

			assert.Equal(t, want, path)
			assert.Equal(t, want != "" && tc.wantErr == "", cfg != nil, "config loaded")

			if tc.wantErr == "" {
				must.Equal(t, nil, err)
			} else {
				must.NotEqual(t, nil, err)
				assert.StringContains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
