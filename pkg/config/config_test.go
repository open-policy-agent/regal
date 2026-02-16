package config

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"
	"github.com/open-policy-agent/opa/v1/util/test"

	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/test/assert"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
)

func TestFindRegalDirectory(t *testing.T) {
	t.Parallel()

	fs := map[string]string{filepath.FromSlash("/foo/bar/baz/p.rego"): ""}

	test.WithTempFS(fs, func(root string) {
		must.MkdirAll(t, root, ".regal")
		must.Return(FindRegalDirectory(filepath.Join(root, "foo", "bar", "baz")))(t)
	})

	fs = map[string]string{filepath.FromSlash("/foo/bar/baz/p.rego"): "", filepath.FromSlash("/foo/bar/bax.json"): ""}

	test.WithTempFS(fs, func(root string) {
		_, err := FindRegalDirectory(filepath.Join(root, "foo", "bar", "baz"))
		assert.NotNil(t, err, "expected no config file found")
	})
}

func TestFindConfig(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		FS           map[string]string
		Error        string
		ExpectedName string
	}{
		"no config file": {
			FS: map[string]string{
				filepath.FromSlash("/foo/bar/baz/p.rego"): "",
				filepath.FromSlash("/foo/bar/bax.json"):   "",
			},
			Error: "could not find Regal config",
		},
		".regal/config.yaml": {
			FS: map[string]string{
				filepath.FromSlash("/foo/bar/baz/p.rego"):         "",
				filepath.FromSlash("/foo/bar/.regal/config.yaml"): "",
			},
			ExpectedName: filepath.FromSlash("/foo/bar/.regal/config.yaml"),
		},
		".regal/ dir missing config file": {
			FS: map[string]string{
				filepath.FromSlash("/foo/bar/baz/p.rego"):   "",
				filepath.FromSlash("/foo/bar/.regal/.keep"): "", // .keep file to ensure the dir is present
			},
			Error: "config file was not found in .regal directory",
		},
		".regal.yaml": {
			FS: map[string]string{
				filepath.FromSlash("/foo/bar/baz/p.rego"):  "",
				filepath.FromSlash("/foo/bar/.regal.yaml"): "",
			},
			ExpectedName: filepath.FromSlash("/foo/bar/.regal.yaml"),
		},
		".regal.yaml and .regal/config.yaml": {
			FS: map[string]string{
				filepath.FromSlash("/foo/bar/baz/p.rego"):         "",
				filepath.FromSlash("/foo/bar/.regal.yaml"):        "",
				filepath.FromSlash("/foo/bar/.regal/config.yaml"): "",
			},
			Error: "conflicting config files: both .regal directory and .regal.yaml found",
		},
		".regal.yaml with .regal/config.yaml at higher directory": {
			FS: map[string]string{
				filepath.FromSlash("/foo/bar/baz/p.rego"):  "",
				filepath.FromSlash("/foo/bar/.regal.yaml"): "",
				filepath.FromSlash("/.regal/config.yaml"):  "",
			},
			ExpectedName: filepath.FromSlash("/foo/bar/.regal.yaml"),
		},
		".regal/config.yaml with .regal.yaml at higher directory": {
			FS: map[string]string{
				filepath.FromSlash("/foo/bar/baz/p.rego"):         "",
				filepath.FromSlash("/foo/bar/.regal/config.yaml"): "",
				filepath.FromSlash("/.regal.yaml"):                "",
			},
			ExpectedName: filepath.FromSlash("/foo/bar/.regal/config.yaml"),
		},
	}

	for testName, testData := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			test.WithTempFS(testData.FS, func(root string) {
				configFile, err := Find(filepath.Join(root, "foo", "bar", "baz"))
				if testData.Error != "" {
					testutil.ErrMustContain(err, testData.Error)(t)
				} else if err != nil {
					t.Fatalf("expected no error, got %s", err)
				}

				if testData.ExpectedName != "" {
					if got, exp := strings.TrimPrefix(configFile.Name(), root), filepath.FromSlash(testData.ExpectedName); got != exp {
						t.Fatalf("expected config file %q, got %q", exp, got)
					}
				}
			})
		})
	}
}

func TestFindBundleRootDirectories(t *testing.T) {
	t.Parallel()

	cfg := `
project:
  roots:
  - foo/bar
  - baz
`

	fs := map[string]string{
		filepath.FromSlash("/.regal/config.yaml"):       cfg, // root from config
		filepath.FromSlash("/.regal/rules/policy.rego"): "",  // custom rules directory
		filepath.FromSlash("/bundle/.manifest"):         "",  // bundle from .manifest
		filepath.FromSlash("/foo/bar/baz/policy.rego"):  "",  // foo/bar from config
		filepath.FromSlash("/baz"):                      "",  // baz from config
	}

	test.WithTempFS(fs, func(root string) {
		locations := must.Return(FindBundleRootDirectories(root))(t)
		assert.Equal(t, 5, len(locations), "locations")

		expected := util.Map([]string{"", ".regal/rules", "baz", "bundle", "foo/bar"}, util.FilepathJoiner(root))
		assert.SlicesEqual(t, expected, locations, "bundle root directories")
	})
}

func TestFindBundleRootDirectoriesWithStandaloneConfig(t *testing.T) {
	t.Parallel()

	cfg := `
project:
  roots:
  - foo/bar
  - baz
`

	fs := map[string]string{
		filepath.FromSlash("/.regal.yaml"):             cfg, // root from config
		filepath.FromSlash("/bundle/.manifest"):        "",  // bundle from .manifest
		filepath.FromSlash("/foo/bar/baz/policy.rego"): "",  // foo/bar from config
		filepath.FromSlash("/baz"):                     "",  // baz from config
	}

	test.WithTempFS(fs, func(root string) {
		locations := must.Return(FindBundleRootDirectories(root))(t)
		assert.Equal(t, 4, len(locations), "locations")

		expected := util.Map([]string{"", "baz", "bundle", "foo/bar"}, util.FilepathJoiner(root))
		assert.SlicesEqual(t, expected, locations, "bundle root directories")
	})
}

func TestMarshalConfig(t *testing.T) {
	t.Parallel()

	conf := Config{
		// ignore is empty and so should not be marshalled
		Ignore: Ignore{Files: []string{}},
		Rules: map[string]Category{
			"testing": {
				"foo": Rule{
					Level:  "error",
					Ignore: &Ignore{Files: []string{"foo.rego"}},
					Extra: ExtraAttributes{
						"bar":    "baz",
						"ignore": "this should be removed by the marshaller",
					},
				},
			},
		},
	}

	bs := must.Return(yaml.Marshal(conf))(t)
	expect := `rules:
    testing:
        foo:
            bar: baz
            ignore:
                files:
                    - foo.rego
            level: error
`

	assert.Equal(t, expect, string(bs), "marshalled config")
}

func TestUnmarshalMarshalConfigWithDefaultRuleConfigs(t *testing.T) {
	t.Parallel()

	bs := []byte(`
rules:
  default:
    level: ignore
  bugs:
    default:
      level: error
    constant-condition:
      level: ignore
  testing:
    print-or-trace-call:
      level: error`)

	originalConfig := testutil.MustUnmarshalYAML[Config](t, bs)

	assert.KeyMissing(t, originalConfig.Rules["bugs"], "default")
	assert.Equal(t, "ignore", originalConfig.Defaults.Global.Level, "global default level")
	assert.Equal(t, "error", originalConfig.Defaults.Categories["bugs"].Level, "bugs default level")
	assert.Equal(t, "error", originalConfig.Rules["testing"]["print-or-trace-call"].Level, "print-or-trace-call level")

	originalConfig.Capabilities = nil

	roundTrippedConfig := testutil.MustUnmarshalYAML[Config](t, must.Return(yaml.Marshal(originalConfig))(t))

	assert.Equal(t, "ignore", roundTrippedConfig.Defaults.Global.Level, "global default level post round trip")
	assert.Equal(t, "error", roundTrippedConfig.Defaults.Categories["bugs"].Level, "bugs default level post round trip")
}

func TestUnmarshalConfig(t *testing.T) {
	t.Parallel()

	bs := []byte(`rules:
  testing:
    foo:
      bar: baz
      ignore:
        files:
          - foo.rego
      level: error
capabilities:
  from:
    engine: opa
    version: v0.45.0
  plus:
    builtins:
      - name: ldap.query
        type: function
        decl:
          args:
            - type: string
        result:
          type: object
  minus:
    builtins:
      - name: http.send`)

	conf := testutil.MustUnmarshalYAML[Config](t, bs)
	fooRule := conf.Rules["testing"]["foo"]

	assert.Equal(t, "error", fooRule.Level, "rule level")
	must.NotEqual(t, nil, fooRule.Ignore, "rule attribute")

	assert.Equal(t, 1, len(fooRule.Ignore.Files), "rule ignore files")
	assert.Equal(t, "foo.rego", fooRule.Ignore.Files[0], "rule ignore file")

	assert.Equal(t, "baz", fooRule.Extra["bar"], "extra attribute")
	assert.Equal(t, nil, fooRule.Extra["ignore"], "extra attribute 'ignore'")
	assert.Equal(t, nil, fooRule.Extra["level"], "extra attribute 'level'")

	assert.Equal(t, 183, len(conf.Capabilities.Builtins), "number of builtins in capabilities set")

	expBuiltins := util.NewSet("regex.match", "ldap.query")
	actBuiltins := util.NewSetFromKeys(conf.Capabilities.Builtins)

	assert.True(t, expBuiltins.Equal(expBuiltins.Intersect(actBuiltins)), "expected builtins %s", expBuiltins)
	assert.False(t, actBuiltins.Contains("http.send"), "http.send should be removed")
}

func TestUnmarshalConfigWithBuiltinsFile(t *testing.T) {
	t.Parallel()

	bs := []byte(`rules: {}
capabilities:
  from:
    file: "./fixtures/caps.json"`)

	conf := testutil.MustUnmarshalYAML[Config](t, bs)

	assert.Equal(t, 1, len(conf.Capabilities.Builtins), "number of builtins in capabilities set")
	assert.True(t, slices.Contains(outil.Keys(conf.Capabilities.Builtins), "wow"), "builtin 'wow' not found")
}

func TestUnmarshalConfigDefaultCapabilities(t *testing.T) {
	t.Parallel()

	conf := testutil.MustUnmarshalYAML[Config](t, []byte("rules: {}\n"))
	caps := io.Capabilities()

	assert.Equal(t, len(caps.Builtins), len(conf.Capabilities.Builtins), "number of builtins")

	// choose the first built-ins to check for to keep the test fast
	expectedBuiltins := []string{caps.Builtins[0].Name, caps.Builtins[1].Name}

	for _, expectedBuiltin := range expectedBuiltins {
		if !slices.Contains(outil.Keys(conf.Capabilities.Builtins), expectedBuiltin) {
			t.Errorf("expected builtin %s to be found", expectedBuiltin)
		}
	}
}

func TestUnmarshalConfigWithNumericOPAVersion(t *testing.T) {
	t.Parallel()

	bs := []byte(`
capabilities:
  from:
    engine: opa
    version: 68
`)

	testutil.ErrMustContain(yaml.Unmarshal(bs, &Config{}), "capabilities: from.version must be a string")(t)
}

func TestUnmarshalConfigWithMissingVPrefixOPAVersion(t *testing.T) {
	t.Parallel()

	bs := []byte(`
capabilities:
  from:
    engine: opa
    version: 0.68.0
`)

	testutil.ErrMustContain(
		yaml.Unmarshal(bs, &Config{}), "capabilities: from.version must be a valid OPA version (with a 'v' prefix)")(t)
}

func TestUnmarshalProjectRootsAsStringOrObject(t *testing.T) {
	t.Parallel()

	bs := []byte(`project:
  roots:
    - foo/bar
    - baz
    - path: bar/baz
    - path: v1
      rego-version: 1
`)

	conf := testutil.MustUnmarshalYAML[Config](t, bs)
	expRoots := []Root{{Path: "foo/bar"}, {Path: "baz"}, {Path: "bar/baz"}, {Path: "v1", RegoVersion: new(1)}}
	roots := *conf.Project.Roots

	assert.Equal(t, len(expRoots), len(roots), "number of project roots")

	for i, expRoot := range expRoots {
		assert.Equal(t, expRoot.Path, roots[i].Path, "root path")

		if expRoot.RegoVersion != nil {
			assert.DereferenceEqual(t, *expRoot.RegoVersion, roots[i].RegoVersion, "root rego version")
		}
	}
}

func TestAllRegoVersions(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		Config   string
		FS       map[string]string
		Expected map[string]ast.RegoVersion
	}{
		"values from config": {
			Config: `project:
  rego-version: 0
  roots:
    - path: foo
      rego-version: 1
`,
			FS: map[string]string{filepath.FromSlash("bar/baz/.manifest"): `{"rego_version": 1}`},
			Expected: map[string]ast.RegoVersion{
				"":                            ast.RegoV0,
				filepath.FromSlash("bar/baz"): ast.RegoV1,
				"foo":                         ast.RegoV1,
			},
		},
		"no config": {
			Config:   "",
			FS:       map[string]string{filepath.FromSlash("bar/baz/.manifest"): `{"rego_version": 1}`},
			Expected: map[string]ast.RegoVersion{},
		},
	}

	for testName, testData := range testCases {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			var conf *Config
			if testData.Config != "" {
				conf = testutil.MustUnmarshalYAML[*Config](t, []byte(testData.Config))
			}

			test.WithTempFS(testData.FS, func(root string) {
				versions := must.Return(AllRegoVersions(root, conf))(t)
				assert.True(t, maps.Equal(versions, testData.Expected))
			})
		})
	}
}
