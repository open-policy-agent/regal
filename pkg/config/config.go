package config

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/types"
	outil "github.com/open-policy-agent/opa/v1/util"

	rbundle "github.com/open-policy-agent/regal/bundle"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

type (
	Config struct { //nolint:recvcheck // conflicting with other linter on YAML marshalling methods
		// Defaults state is loaded from configuration under rules and so is not (un)marshalled
		// in the same way.
		Defaults        Defaults      `json:"-"                          yaml:"-"`
		Rules           Rules         `json:"rules"                      yaml:"rules"`
		Capabilities    *Capabilities `json:"capabilities,omitempty"     yaml:"capabilities,omitempty"`
		Features        *Features     `json:"features,omitempty"         yaml:"features,omitempty"`
		Project         *Project      `json:"project,omitempty"          yaml:"project,omitempty"`
		CapabilitiesURL string        `json:"capabilities_url,omitempty" yaml:"capabilities_url,omitempty"`
		Ignore          Ignore        `json:"ignore,omitzero"            yaml:"ignore,omitempty"`
	}
	Rules map[string]Category
	Root  struct {
		Path string
		// Note that contrary to ast.RegoVersion, we'll only accept 0 or 1 here currently.
		// This aligns with the Rego versioning scheme used in .manifest files, which is
		// the alternative way to provide this for a specific directory.
		RegoVersion *int `json:"rego-version,omitempty" yaml:"rego-version,omitempty"`
	}
	Project struct {
		Roots *[]Root `json:"roots,omitempty" yaml:"roots,omitempty"`
		// Set the Rego version for the whole project or workspace. Individual roots may override this.
		RegoVersion *int `json:"rego-version,omitempty" yaml:"rego-version,omitempty"`
	}
	Category map[string]Rule
	// Defaults is used to store information about global and category
	// defaults for rules.
	Defaults struct {
		Categories map[string]Default
		Global     Default
	}
	// Default represents global or category settings for rules,
	// currently only the level is supported.
	Default struct {
		Level string `json:"level" yaml:"level"`
	}
	Features struct {
		Remote *RemoteFeatures `json:"remote,omitempty" yaml:"remote,omitempty"`
	}
	RemoteFeatures struct {
		CheckVersion bool `json:"check-version,omitempty" yaml:"check-version,omitempty"`
	}
	Ignore struct {
		Files []string `json:"files,omitempty" yaml:"files,omitempty"`
	}
	ExtraAttributes map[string]any
	Rule            struct { //nolint:recvcheck
		Ignore *Ignore         `json:"ignore,omitempty" yaml:"ignore,omitempty"`
		Extra  ExtraAttributes `json:",inline"          yaml:",inline"` //nolint:revive
		Level  string          `json:"level"`
	}
	Capabilities struct { //nolint:recvcheck
		Builtins       map[string]*Builtin `json:"builtins"        yaml:"builtins"`
		FutureKeywords []string            `json:"future_keywords" yaml:"future_keywords"`
		Features       []string            `json:"features"        yaml:"features"`
	}
	Decl struct {
		Result string   `json:"result" yaml:"result"`
		Args   []string `json:"args"   yaml:"args"`
	}
	Builtin struct {
		Decl Decl `json:"decl" yaml:"decl"`
	}
)

const (
	capabilitiesEngineOPA    = "opa"
	capabilitiesEngineEOPA   = "eopa"
	keyIgnore                = "ignore"
	keyLevel                 = "level"
	regalDirName             = ".regal"
	configFileName           = "config.yaml"
	standaloneConfigFileName = ".regal.yaml"
)

var (
	Embedded = sync.OnceValue(func() *Config {
		file := rbundle.Open("regal/config/provided/data.yaml")
		defer file.Close()

		return util.Must(FromReader(file))
	})
	OpaCapsForThisVersionTerm = sync.OnceValue(func() *ast.Term {
		return ast.NewTerm(rast.StructToValue(OpaCapsForThisVersionInterned()))
	})
	OpaCapsForThisVersionInterned = sync.OnceValue(CapabilitiesForThisVersion)
	opabuiltinCache               = concurrent.MapOf(make(map[string]*Builtin))
)

func (d Defaults) IsZero() bool {
	return d.Global.Level == "" && len(d.Categories) == 0
}

func (d Defaults) Level(category string) string {
	if cat, ok := d.Categories[category]; ok {
		return cat.Level
	}

	return d.Global.Level
}

func (r Rules) Copy() map[string]Category {
	return util.WithMappedValues(r, Category.Copy)
}

func (c Category) Copy() Category {
	return util.WithMappedValues(c, Rule.Copy)
}

func (c Category) Equal(other Category) bool {
	return maps.EqualFunc(c, other, Rule.Equal)
}

func (r Rule) Equal(other Rule) bool {
	return r.Level == other.Level &&
		((r.Ignore == nil && other.Ignore == nil) ||
			(r.Ignore != nil && other.Ignore != nil && slices.Equal(r.Ignore.Files, other.Ignore.Files))) &&
		((r.Extra == nil && other.Extra == nil) ||
			(r.Extra != nil && other.Extra != nil && reflect.DeepEqual(r.Extra, other.Extra)))
}

func (r Rule) Copy() Rule {
	c := Rule{Level: r.Level}
	if r.Ignore != nil {
		c.Ignore = &Ignore{Files: slices.Clone(r.Ignore.Files)}
	}

	if r.Extra != nil {
		c.Extra = maps.Clone(r.Extra)
	}

	return c
}

func HasConfigSuffix(path string) bool {
	return util.HasAnySuffix(path, standaloneConfigFileName, ".regal/config.yaml", ".regal\\config.yaml")
}

func FromPath(path string) (*Config, error) {
	return util.Wrap(rio.WithOpen(path, FromReader))("failed to open config file")
}

func FromReader[R io.Reader](r R) (*Config, error) {
	cfg := &Config{}
	err := yaml.NewDecoder(r).Decode(cfg)

	return cfg, err
}

// Find attempts to find either the .regal directory or .regal.yaml
// config file, and returns the appropriate file or an error.
func Find(path string) (*os.File, error) {
	regalDir, regalDirError := FindRegalDirectory(path)
	regalConfigFile, regalConfigFileError := FindRegalConfigFile(path)

	var regalDirParent, regalConfigFileParent string
	if regalDirError == nil && regalConfigFileError == nil {
		regalDirParent = filepath.Dir(regalDir.Name())
		regalConfigFileParent = filepath.Dir(regalConfigFile.Name())

		if regalDirParent == regalConfigFileParent {
			return nil, errors.New("conflicting config files: both .regal directory and .regal.yaml found")
		}
	}

	if regalDirError != nil && regalConfigFileError != nil {
		return nil, errors.New("could not find Regal config")
	}

	// if the config file parent is not "", then it was found, and if it's
	// longer then it's more specific to the search path in question, and so we
	// return that here in such cases.
	if len(regalConfigFileParent) > len(regalDirParent) {
		return regalConfigFile, nil
	}

	// if there is a .regal directory, a config file is expected to be found inside.
	if regalDirError == nil {
		expectedConfigFilePath := filepath.Join(regalDir.Name(), configFileName)

		if _, err := os.Stat(expectedConfigFilePath); err != nil && os.IsNotExist(err) {
			return nil, errors.New("config file was not found in .regal directory")
		} else if err != nil {
			return nil, fmt.Errorf("failed to stat .regal config file: %w", err)
		}

		return os.Open(expectedConfigFilePath)
	}

	// regalConfigFileError is nil at this point, so we can return the file
	return regalConfigFile, nil
}

// FindRegalDirectoryPath searches for a .regal directory upwards from the provided path.
func FindRegalDirectoryPath(path string) (string, error) {
	return findUpwards(path, regalDirName, true)
}

// FindRegalDirectory searches for a .regal directory upwards from the provided path.
func FindRegalDirectory(path string) (*os.File, error) {
	return findFileUpwards(path, regalDirName, true)
}

// FindRegalConfigFile searches for a .regal.yaml config file upwards from the provided path.
func FindRegalConfigFile(path string) (*os.File, error) {
	return findFileUpwards(path, standaloneConfigFileName, false)
}

// FindBundleRootDirectories finds all bundle root directories from the provided path,
// which **must** be an absolute path. Bundle root directories may be found either by:
//
// - Configuration (`project.roots`)
// - By the presence of a .manifest file anywhere under the path
// - By the presence of a .regal directory anywhere under or above the path ...
//
// All returned paths are absolute paths. If the provided path itself
// is determined to be a bundle root directory it will be included in the result.
func FindBundleRootDirectories(path string) ([]string, error) {
	var foundBundleRoots []string

	// This will traverse the tree **upwards** searching for a .regal directory
	regalDir, err := FindRegalDirectory(path)
	if err == nil {
		roots, err := rootsFromRegalConfigDirOrFile(regalDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get roots from .regal directory: %w", err)
		}

		foundBundleRoots = append(foundBundleRoots, roots...)
	}

	// This will traverse the tree **upwards** searching for a .regal.yaml file
	regalConfigFile, err := FindRegalConfigFile(path)
	if err == nil {
		roots, err := rootsFromRegalConfigDirOrFile(regalConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get roots from .regal.yaml: %w", err)
		}

		foundBundleRoots = append(foundBundleRoots, roots...)
	}

	f := func(path string) error {
		if filepath.Dir(path) == regalDirName {
			// Opening files as part of walking is generally not a good idea...
			// but I think we can assume the number of .regal directories in a project
			// is limited to a reasonable number.
			roots, err := rio.WithOpen(path, rootsFromRegalConfigDirOrFile)
			if err != nil {
				return fmt.Errorf("failed to get roots from .regal directory: %w", err)
			}

			foundBundleRoots = append(foundBundleRoots, roots...)
		}

		// check for .manifest directories as part of the same walk
		if filepath.Base(path) == ".manifest" {
			foundBundleRoots = append(foundBundleRoots, filepath.Dir(path))
		}

		return nil
	}

	// This will traverse the tree **downwards** searching for .regal directories
	if err := files.NewWalker(path).WithSkipFunc(filter.DefaultSkipDirectories).Walk(f); err != nil {
		return nil, fmt.Errorf("failed to walk path: %w", err)
	}

	return slices.Compact(outil.Sorted(foundBundleRoots)), nil
}

// AllRegoVersions returns a map of all Rego versions found in the provided config and .manifest files,
// keyed by the path of the directory for which the version applies. Config file has higher precedence
// than .manifest files.
//
// // TODO(anders): This is weird. Config may have higher precedence but why do we not even cnsider
// // the manifest files at all if no config is provided?
func AllRegoVersions(root string, conf *Config) (map[string]ast.RegoVersion, error) {
	versionsMap := make(map[string]ast.RegoVersion)
	if conf == nil {
		return versionsMap, nil
	}

	manifestLocations, err := rio.FindManifestLocations(root)
	if err != nil {
		return nil, fmt.Errorf("failed to find manifest locations: %w", err)
	}

	for _, dir := range manifestLocations {
		b, err := os.ReadFile(filepath.Join(root, dir, ".manifest"))
		if err != nil {
			return nil, fmt.Errorf("failed to read manifest file: %w", err)
		}

		manifest, err := encoding.JSONUnmarshalTo[bundle.Manifest](b)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest file: %w", err)
		}

		if manifest.RegoVersion != nil {
			versionsMap[dir] = regoVersionFromConfigValue(manifest.RegoVersion)
		}
	}

	if conf.Project != nil {
		if conf.Project.RegoVersion != nil {
			// empty string is used to denote the project root
			// and all other paths are relative to that
			versionsMap[""] = regoVersionFromConfigValue(conf.Project.RegoVersion)
		}

		if conf.Project.Roots != nil {
			for _, root := range *conf.Project.Roots {
				if root.RegoVersion != nil {
					versionsMap[root.Path] = regoVersionFromConfigValue(root.RegoVersion)
				}
			}
		}
	}

	return versionsMap, nil
}

// CapabilitiesForThisVersion returns the capabilities for the current OPA version Regal depends on.
func CapabilitiesForThisVersion() *Capabilities {
	return fromOPACapabilities(rio.OPACapabilities())
}

func (f *Features) IsZero() bool {
	return f == nil || f.Remote == nil
}

func GetPotentialRoots(paths ...string) ([]string, error) {
	absDirPaths := make([]string, len(paths))
	dirMap := util.NewSet[string]()

	for i, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
		}

		absDirPaths[i] = rio.Dir(abs)

		brds, err := FindBundleRootDirectories(absDirPaths[i])
		if err != nil {
			return nil, fmt.Errorf("failed to find bundle root directories in %s: %w", absDirPaths[i], err)
		}

		dirMap.Add(brds...)
	}

	if dirMap.Size() == 0 {
		return absDirPaths, nil
	}

	return dirMap.Items(), nil
}

// findUpwards searches for a file or directory matching the given name,
// starting from the provided path and moving upwards.
func findUpwards(path, name string, expectDir bool) (string, error) {
	dir := rio.Dir(path)
	volume := filepath.VolumeName(dir) // "C:" on windows, "" on other platforms

	for {
		var searchPath string
		if volume == "" {
			searchPath = filepath.Join(string(os.PathSeparator), dir, name)
		} else {
			searchPath = filepath.Join(dir, name)
		}

		if fileInfo, err := os.Stat(searchPath); err == nil && fileInfo.IsDir() == expectDir {
			return searchPath, nil
		}

		//nolint:gocritic
		if searchPath == volume+string(os.PathSeparator)+name {
			return "", errors.New("can't traverse past root directory")
		}

		// Move up one level in the directory tree
		parts := strings.Split(dir, string(os.PathSeparator))
		if len(parts) < 2 {
			return "", errors.New("stopping as dir is root directory")
		}

		parts = parts[:len(parts)-1]
		if parts[0] == volume {
			parts[0] = volume + string(os.PathSeparator)
		}

		dir = filepath.Join(parts...)
	}
}

func findFileUpwards(path, name string, expectDir bool) (*os.File, error) {
	dir, err := findUpwards(path, name, expectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find .regal directory: %w", err)
	}

	return util.Wrap(os.Open(dir))("failed to open .regal directory")
}

// TODO: this doesn't properly handle .regal.yaml??
func rootsFromRegalConfigDirOrFile(file *os.File) ([]string, error) {
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	parent, filename := filepath.Split(file.Name())
	if (fileInfo.IsDir() && filename != ".regal") ||
		(!fileInfo.IsDir() && filename != ".regal.yaml") {
		return nil, fmt.Errorf("expected a directory named '.regal' or a file named '.regal.yaml', got '%s'", filename)
	}

	parent = filepath.Clean(parent)

	configFilePath := file.Name()
	if fileInfo.IsDir() {
		configFilePath = filepath.Join(configFilePath, "config.yaml")
	}

	foundBundleRoots := []string{parent}

	if conf, err := FromPath(configFilePath); err == nil && conf.Project != nil && conf.Project.Roots != nil {
		for _, root := range *conf.Project.Roots {
			foundBundleRoots = append(foundBundleRoots, filepath.Join(parent, root.Path))
		}
	}

	// Include the "rules" directory when loading from a .regal dir
	if fileInfo.IsDir() {
		if customDir := filepath.Join(file.Name(), "rules"); rio.IsDir(customDir) {
			foundBundleRoots = append(foundBundleRoots, customDir)
		}
	}

	// Include a search for manifest files
	manifestRoots, err := rio.FindManifestLocations(parent)
	if err != nil {
		return nil, fmt.Errorf("failed while looking for manifest locations: %w", err)
	}

	return append(foundBundleRoots, outil.Map(manifestRoots, util.FilepathJoiner(parent))...), nil
}

func fromOPABuiltin(builtin *ast.Builtin) *Builtin {
	if b, ok := opabuiltinCache.Get(builtin.Name); ok {
		return b
	}

	rb := &Builtin{Decl: Decl{Args: outil.Map(builtin.Decl.FuncArgs().Args, types.Type.String)}}
	if builtin.Decl != nil && builtin.Decl.Result() != nil {
		// internal.print has no result
		rb.Decl.Result = builtin.Decl.Result().String()
	}

	opabuiltinCache.Set(builtin.Name, rb)

	return rb
}

func fromOPACapabilities(capabilities *ast.Capabilities) *Capabilities {
	result := Capabilities{Builtins: make(map[string]*Builtin, len(capabilities.Builtins))}
	for _, builtin := range capabilities.Builtins {
		result.Builtins[builtin.Name] = fromOPABuiltin(builtin)
	}

	result.FutureKeywords = capabilities.FutureKeywords
	result.Features = capabilities.Features

	return &result
}

// Translate integer value 0 or 1 as found in config or .manifest to ast.RegoVersion.
func regoVersionFromConfigValue(version *int) ast.RegoVersion {
	if version != nil {
		switch *version {
		case 0:
			return ast.RegoV0
		case 1:
			return ast.RegoV1
		}
	}

	return ast.RegoUndefined
}
