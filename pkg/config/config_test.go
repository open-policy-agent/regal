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
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/internal/util"
)

const levelError = "error"

func TestFindRegalDirectory(t *testing.T) {
	t.Parallel()

	fs := map[string]string{filepath.FromSlash("/foo/bar/baz/p.rego"): ""}

	test.WithTempFS(fs, func(root string) {
		testutil.MustMkdirAll(t, root, ".regal")
		testutil.Must(FindRegalDirectory(filepath.Join(root, "foo", "bar", "baz")))(t)
	})

	fs = map[string]string{filepath.FromSlash("/foo/bar/baz/p.rego"): "", filepath.FromSlash("/foo/bar/bax.json"): ""}

	test.WithTempFS(fs, func(root string) {
		if _, err := FindRegalDirectory(filepath.Join(root, "foo", "bar", "baz")); err == nil {
			t.Errorf("expected no config file to be found")
		}
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
		locations := testutil.Must(FindBundleRootDirectories(root))(t)
		if len(locations) != 5 {
			t.Errorf("expected 5 locations, got %d", len(locations))
		}

		expected := util.Map([]string{"", ".regal/rules", "baz", "bundle", "foo/bar"}, util.FilepathJoiner(root))
		if !slices.Equal(expected, locations) {
			t.Errorf("expected\n%s\ngot\n%s", strings.Join(expected, "\n"), strings.Join(locations, "\n"))
		}
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
		locations := testutil.Must(FindBundleRootDirectories(root))(t)
		if len(locations) != 4 {
			t.Errorf("expected 5 locations, got %d", len(locations))
		}

		expected := util.Map([]string{"", "baz", "bundle", "foo/bar"}, util.FilepathJoiner(root))
		if !slices.Equal(expected, locations) {
			t.Errorf("expected\n%s\ngot\n%s", strings.Join(expected, "\n"), strings.Join(locations, "\n"))
		}
	})
}

func TestMarshalConfig(t *testing.T) {
	t.Parallel()

	conf := Config{
		// ignore is empty and so should not be marshalled
		Ignore: Ignore{
			Files: []string{},
		},
		Rules: map[string]Category{
			"testing": {
				"foo": Rule{
					Level: "error",
					Ignore: &Ignore{
						Files: []string{"foo.rego"},
					},
					Extra: ExtraAttributes{
						"bar":    "baz",
						"ignore": "this should be removed by the marshaller",
					},
				},
			},
		},
	}

	bs := testutil.Must(yaml.Marshal(conf))(t)

	expect := `rules:
    testing:
        foo:
            bar: baz
            ignore:
                files:
                    - foo.rego
            level: error
`

	if string(bs) != expect {
		t.Errorf("expected:\n%sgot:\n%s", expect, string(bs))
	}
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
      level: error
`)

	originalConfig := testutil.MustUnmarshalYAML[Config](t, bs)

	if originalConfig.Defaults.Global.Level != "ignore" {
		t.Errorf("expected global default to be level ignore")
	}

	if _, unexpected := originalConfig.Rules["bugs"]["default"]; unexpected {
		t.Errorf("erroneous rule parsed, bugs.default should not exist")
	}

	if originalConfig.Defaults.Categories["bugs"].Level != levelError {
		t.Errorf("expected bugs default to be level error")
	}

	if originalConfig.Rules["testing"]["print-or-trace-call"].Level != levelError {
		t.Errorf("expected for testing.print-or-trace-call to be level error")
	}

	originalConfig.Capabilities = nil

	roundTrippedConfig := testutil.MustUnmarshalYAML[Config](t, testutil.Must(yaml.Marshal(originalConfig))(t))

	if roundTrippedConfig.Defaults.Global.Level != "ignore" {
		t.Errorf("expected global default to be level ignore")
	}

	if roundTrippedConfig.Defaults.Categories["bugs"].Level != levelError {
		t.Errorf("expected bugs default to be level error")
	}
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
      - name: http.send
`)

	conf := testutil.MustUnmarshalYAML[Config](t, bs)

	if conf.Rules["testing"]["foo"].Level != "error" {
		t.Errorf("expected level to be error")
	}

	if conf.Rules["testing"]["foo"].Ignore == nil {
		t.Errorf("expected ignore attribute to be set")
	}

	if len(conf.Rules["testing"]["foo"].Ignore.Files) != 1 {
		t.Errorf("expected ignore files to be set")
	}

	if conf.Rules["testing"]["foo"].Ignore.Files[0] != "foo.rego" {
		t.Errorf("expected ignore files to contain foo.rego")
	}

	if conf.Rules["testing"]["foo"].Extra["bar"] != "baz" {
		t.Errorf("expected extra attribute 'bar' to be baz")
	}

	if conf.Rules["testing"]["foo"].Extra["ignore"] != nil {
		t.Errorf("expected extra attribute 'ignore' to be removed")
	}

	if conf.Rules["testing"]["foo"].Extra["level"] != nil {
		t.Errorf("expected extra attribute 'level' to be removed")
	}

	if exp, got := 183, len(conf.Capabilities.Builtins); exp != got {
		t.Errorf("expected %d builtins, got %d", exp, got)
	}

	expBuiltins := util.NewSet("regex.match", "ldap.query")
	actBuiltins := util.NewSetFromKeys(conf.Capabilities.Builtins)

	if !expBuiltins.Equal(expBuiltins.Intersect(actBuiltins)) {
		t.Errorf("expected builtins %s in capabilities set", expBuiltins)
	}

	if actBuiltins.Contains("http.send") {
		t.Errorf("expected builtin http.send to be removed from capabilities set")
	}
}

func TestUnmarshalConfigWithBuiltinsFile(t *testing.T) {
	t.Parallel()

	bs := []byte(`rules: {}
capabilities:
  from:
    file: "./fixtures/caps.json"
`)

	conf := testutil.MustUnmarshalYAML[Config](t, bs)

	if exp, got := 1, len(conf.Capabilities.Builtins); exp != got {
		t.Errorf("expected %d builtins, got %d", exp, got)
	}

	if !slices.Contains(outil.Keys(conf.Capabilities.Builtins), "wow") {
		t.Errorf("expected builtin 'wow' to be found")
	}
}

func TestUnmarshalConfigDefaultCapabilities(t *testing.T) {
	t.Parallel()

	conf := testutil.MustUnmarshalYAML[Config](t, []byte("rules: {}\n"))
	caps := io.Capabilities()

	if exp, got := len(caps.Builtins), len(conf.Capabilities.Builtins); exp != got {
		t.Errorf("expected %d builtins, got %d", exp, got)
	}

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
	version1 := 1
	expRoots := []Root{{Path: "foo/bar"}, {Path: "baz"}, {Path: "bar/baz"}, {Path: "v1", RegoVersion: &version1}}
	roots := *conf.Project.Roots

	if len(roots) != len(expRoots) {
		t.Errorf("expected %d roots, got %d", len(expRoots), len(roots))
	}

	for i, expRoot := range expRoots {
		if roots[i].Path != expRoot.Path {
			t.Errorf("expected root path %v, got %v", expRoot.Path, roots[i].Path)
		}

		if expRoot.RegoVersion != nil {
			if roots[i].RegoVersion == nil {
				t.Errorf("expected root %v to have a rego version", expRoot.Path)
			} else if *roots[i].RegoVersion != *expRoot.RegoVersion {
				t.Errorf(
					"expected root %v rego version %v, got %v",
					expRoot.Path, *expRoot.RegoVersion, *roots[i].RegoVersion,
				)
			}
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
				versions := testutil.Must(AllRegoVersions(root, conf))(t)
				if !maps.Equal(versions, testData.Expected) {
					t.Errorf("expected %v, got %v", testData.Expected, versions)
				}
			})
		})
	}
}
