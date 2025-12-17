package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/bundle"

	"github.com/open-policy-agent/regal/internal/capabilities"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/io/files"
	"github.com/open-policy-agent/regal/internal/io/files/filter"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
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

type (
	// The receiver types is a bit of a mess, but we can probably not change
	// these without breaking some integration. So this should be a 1.0 change.

	//nolint:recvcheck
	Config struct {
		// Defaults state is loaded from configuration under rules and so is not (un)marshalled
		// in the same way.
		Defaults        Defaults            `json:"-"                          yaml:"-"`
		Rules           map[string]Category `json:"rules"                      yaml:"rules"`
		Capabilities    *Capabilities       `json:"capabilities,omitempty"     yaml:"capabilities,omitempty"`
		Features        *Features           `json:"features,omitempty"         yaml:"features,omitempty"`
		Project         *Project            `json:"project,omitempty"          yaml:"project,omitempty"`
		CapabilitiesURL string              `json:"capabilities_url,omitempty" yaml:"capabilities_url,omitempty"`
		Ignore          Ignore              `json:"ignore"                     yaml:"ignore"`
	}

	Root struct {
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

	//nolint:recvcheck
	Rule struct {
		Ignore *Ignore `json:"ignore,omitempty" yaml:"ignore,omitempty"`
		Extra  ExtraAttributes
		Level  string
	}

	Capabilities struct {
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

func (d *Default) mapToConfig(result any) error {
	resultMap, ok := result.(map[string]any)
	if !ok {
		return errors.New("result was not a map")
	}

	level, ok := resultMap[keyLevel].(string)
	if ok {
		d.Level = level
	}

	return nil
}

func FromPath(path string) (Config, error) {
	return util.Wrap(rio.WithOpen(path, FromFile))("failed to open config file")
}

func FromFile(file *os.File) (Config, error) {
	conf := Config{}
	err := yaml.NewDecoder(file).Decode(&conf)

	return conf, err
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

		_, err := os.Stat(expectedConfigFilePath)
		if err != nil && os.IsNotExist(err) {
			return nil, errors.New("config file was not found in .regal directory")
		} else if err != nil {
			return nil, fmt.Errorf("failed to stat .regal config file: %w", err)
		}

		return os.Open(expectedConfigFilePath)
	}

	// regalConfigFileError is nil at this point, so we can return the file
	return regalConfigFile, nil
}

// findUpwards searches for a file or directory matching the given name,
// starting from the provided path and moving upwards.
func findUpwards(path, name string, expectDir bool) (string, error) {
	dir := path
	if !rio.IsDir(path) {
		dir = filepath.Dir(path)
	}

	// returns e.g. "C:" on windows, "" on other platforms
	volume := filepath.VolumeName(dir)

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

	// This will traverse the tree **downwards** searching for .regal directories
	if err := files.NewWalker(path).WithSkipFunc(filter.DefaultSkipDirectories).Walk(func(path string) error {
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
	}); err != nil {
		return nil, fmt.Errorf("failed to walk path: %w", err)
	}

	return slices.Compact(util.Sorted(foundBundleRoots)), nil
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

	return append(foundBundleRoots, util.Map(manifestRoots, util.FilepathJoiner(parent))...), nil
}

// AllRegoVersions returns a map of all Rego versions found in the provided config and .manifest files,
// keyed by the path of the directory for which the version applies. Config file has higher precedence
// than .manifest files.
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

// Translate integer value 0 or 1 as found in config or .manifest to ast.RegoVersion.
func regoVersionFromConfigValue(version *int) ast.RegoVersion {
	if version == nil {
		return ast.RegoUndefined
	}

	switch *version {
	case 0:
		return ast.RegoV0
	case 1:
		return ast.RegoV1
	default:
		return ast.RegoUndefined
	}
}

func (c Config) MarshalYAML() (any, error) {
	unstructuredConfig, err := encoding.JSONRoundTripTo[map[string]any](c)
	if err != nil {
		return nil, fmt.Errorf("failed to created unstructured config: %w", err)
	}

	// place the global defaults at the top level under rules
	if c.Defaults.Global.Level != "" {
		r, ok := unstructuredConfig["rules"].(map[string]any)
		if !ok {
			return nil, errors.New("rules in config were not a map")
		}

		r["default"] = c.Defaults.Global
	}

	// place the category defaults under the respective category
	for categoryName, categoryDefault := range c.Defaults.Categories {
		rawRuleMap, ok := unstructuredConfig["rules"].(map[string]any)
		if !ok {
			return nil, errors.New("rules in config were not a map")
		}

		rawCategoryMap, ok := rawRuleMap[categoryName].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("category %s was not a map", categoryName)
		}

		rawCategoryMap["default"] = categoryDefault
	}

	if len(c.Ignore.Files) == 0 {
		delete(unstructuredConfig, keyIgnore)
	}

	if c.CapabilitiesURL == "" || c.CapabilitiesURL == capabilities.DefaultURL {
		delete(unstructuredConfig, "capabilities_url")
	}

	return unstructuredConfig, nil
}

// unmarshallingIntermediary is used to contain config data in a format that is used during unmarshalling.
// The internally loaded config data layout differs from the user-defined YAML.
type marshallingIntermediary struct {
	// rules are unmarshalled as any since the defaulting needs to be extracted from here
	// and configured elsewhere in the struct.
	Rules        map[string]any `yaml:"rules"`
	Project      *Project       `yaml:"project"`
	Capabilities struct {
		From struct {
			Engine  string `yaml:"engine"`
			Version any    `yaml:"version"`
			File    string `yaml:"file"`
			URL     string `yaml:"url"`
		} `yaml:"from"`
		Plus struct {
			Builtins []*ast.Builtin `yaml:"builtins"`
		} `yaml:"plus"`
		Minus struct {
			Builtins []struct {
				Name string `yaml:"name"`
			} `yaml:"builtins"`
		} `yaml:"minus"`
	} `yaml:"capabilities"`
	Ignore   Ignore `yaml:"ignore"`
	Features struct {
		RemoteFeatures struct {
			CheckVersion bool `yaml:"check-version"`
		} `yaml:"remote"`
	} `yaml:"features"`
}

func (p *Project) UnmarshalYAML(value *yaml.Node) error {
	var result map[string]any

	if err := value.Decode(&result); err != nil {
		return fmt.Errorf("unmarshalling project failed %w", err)
	}

	if result["roots"] != nil {
		// A project root can either be provided as a string or an object. If it's a string, we'll
		// convert it to an object with the path as the only value.
		if roots, ok := result["roots"].([]any); ok {
			for key, val := range roots {
				if root, ok := val.(string); ok {
					roots[key] = Root{Path: root}
				}
			}
		}
	}

	return encoding.JSONRoundTrip(result, p)
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	var result marshallingIntermediary

	if err := value.Decode(&result); err != nil {
		return fmt.Errorf("unmarshalling config failed %w", err)
	}

	// this call will walk the rule config and load defaults into the config
	if err := extractDefaults(c, &result); err != nil {
		return fmt.Errorf("extracting defaults failed: %w", err)
	}

	if err := extractRules(c, &result); err != nil {
		return fmt.Errorf("extracting rules failed: %w", err)
	}

	c.Ignore = result.Ignore

	capabilitiesFile := result.Capabilities.From.File
	capabilitiesEngine := result.Capabilities.From.Engine
	capabilitiesEngineVersion := result.Capabilities.From.Version
	capabilitiesURL := result.Capabilities.From.URL

	// Capabilities can be specified by an engine+version combo, a local
	// file path, or a URL. These cannot be mixed and matched.
	if capabilitiesURL != "" && capabilitiesFile != "" {
		return errors.New("capabilities from.url and from.file are mutually exclusive")
	}

	if capabilitiesURL != "" && capabilitiesEngine != "" {
		return errors.New("capabilities from.url and from.engine are mutually exclusive")
	}

	if capabilitiesURL != "" && capabilitiesEngineVersion != "" {
		return errors.New("capabilities from.url and from.version are mutually exclusive")
	}

	if capabilitiesFile != "" && capabilitiesEngine != "" {
		return errors.New("capabilities from.file and from.engine are mutually exclusive")
	}

	if capabilitiesEngine != "" && capabilitiesEngineVersion == "" {
		// Although regal:///capabilities/{engine} is valid and refers
		// to the latest version for that engine, we'll keep the
		// existing (pre-capabilities.Lookup()) behavior in place and
		// disallow that when using the engine key.
		return errors.New("please set the version for the engine from which to load capabilities from")
	}

	if capabilitiesEngine != "" {
		version, ok := capabilitiesEngineVersion.(string)
		if !ok {
			return errors.New("capabilities: from.version must be a string")
		}

		if capabilitiesEngine == capabilitiesEngineOPA && !strings.HasPrefix(version, "v") {
			return errors.New("capabilities: from.version must be a valid OPA version (with a 'v' prefix)")
		}

		capabilitiesURL = "regal:///capabilities/" + capabilitiesEngine + "/" + version
	}

	if capabilitiesFile != "" {
		absfp, err := filepath.Abs(capabilitiesFile)
		if err != nil {
			return fmt.Errorf(
				"unable to load capabilities from '%s', failed to determine absolute path: %w",
				capabilitiesFile,
				err,
			)
		}

		// prepending a / is done here to ensure that windows drive letter paths
		// are parsed as paths and not host:ports in URLs.
		if !strings.HasPrefix(absfp, "/") {
			absfp = "/" + absfp
		}

		capabilitiesURL = "file://" + absfp
	}

	if capabilitiesEngine == "" && capabilitiesFile == "" && capabilitiesURL == "" {
		capabilitiesURL = capabilities.DefaultURL
	}

	opaCaps, err := capabilities.Lookup(context.Background(), capabilitiesURL)
	if err != nil {
		return fmt.Errorf("failed to load capabilities: %w", err)
	}

	// This is used in the LSP to load the OPA capabilities, since the
	// capabilities version in the user-facing config does not contain all
	// of the information that the LSP needs.
	c.CapabilitiesURL = capabilitiesURL
	c.Capabilities = fromOPACapabilities(opaCaps)
	c.Project = result.Project

	// remove any builtins referenced in the minus config
	for _, minusBuiltin := range result.Capabilities.Minus.Builtins {
		delete(c.Capabilities.Builtins, minusBuiltin.Name)
	}

	// add any builtins referenced in the plus config
	for _, plusBuiltin := range result.Capabilities.Plus.Builtins {
		c.Capabilities.Builtins[plusBuiltin.Name] = fromOPABuiltin(*plusBuiltin)
	}

	// feature defaults
	if result.Features.RemoteFeatures.CheckVersion {
		c.Features = &Features{Remote: &RemoteFeatures{CheckVersion: true}}
	}

	return nil
}

// extractRules is a helper to load rules from the raw config data.
func extractRules(config *Config, result *marshallingIntermediary) error {
	// in order to support wildcard 'default' configs, we
	// have some hooks in this unmarshalling process to load these.
	categoryMap := make(map[string]Category)

	for key, val := range result.Rules {
		if key == "default" {
			continue
		}

		rawRuleMap, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("rules for category %s were not a map", key)
		}

		ruleMap := make(map[string]Rule)

		for ruleName, ruleData := range rawRuleMap {
			if ruleName == "default" {
				continue
			}

			var r Rule

			if err := r.mapToConfig(ruleData); err != nil {
				return fmt.Errorf("unmarshalling rule failed: %w", err)
			}

			ruleMap[ruleName] = r
		}

		categoryMap[key] = ruleMap
	}

	config.Rules = categoryMap

	return nil
}

// extractDefaults is a helper to load both global and category defaults from the raw config data.
func extractDefaults(c *Config, result *marshallingIntermediary) error {
	c.Defaults.Categories = make(map[string]Default)

	if rawGlobalDefault, ok := result.Rules["default"]; ok {
		if err := c.Defaults.Global.mapToConfig(rawGlobalDefault); err != nil {
			return fmt.Errorf("unmarshalling global defaults failed: %w", err)
		}
	}

	for key, val := range result.Rules {
		rawRuleMap, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("rules for category %s were not a map", key)
		}

		rawCategoryDefault, ok := rawRuleMap["default"]
		if ok {
			var categoryDefault Default

			if err := categoryDefault.mapToConfig(rawCategoryDefault); err != nil {
				return fmt.Errorf("unmarshalling category defaults failed: %w", err)
			}

			c.Defaults.Categories[key] = categoryDefault
		}
	}

	return nil
}

// CapabilitiesForThisVersion returns the capabilities for the current OPA version Regal depends on.
func CapabilitiesForThisVersion() *Capabilities {
	return fromOPACapabilities(rio.OPACapabilities())
}

func fromOPABuiltin(builtin ast.Builtin) *Builtin {
	funcArgs := builtin.Decl.FuncArgs().Args
	args := make([]string, len(funcArgs))

	for i, arg := range funcArgs {
		args[i] = arg.String()
	}

	rb := &Builtin{Decl: Decl{Args: args}}

	if builtin.Decl != nil && builtin.Decl.Result() != nil {
		// internal.print has no result
		rb.Decl.Result = builtin.Decl.Result().String()
	}

	return rb
}

func fromOPACapabilities(capabilities *ast.Capabilities) *Capabilities {
	result := Capabilities{Builtins: make(map[string]*Builtin, len(capabilities.Builtins))}
	for _, builtin := range capabilities.Builtins {
		result.Builtins[builtin.Name] = fromOPABuiltin(*builtin)
	}

	result.FutureKeywords = capabilities.FutureKeywords
	result.Features = capabilities.Features

	return &result
}

func ToMap(config Config) map[string]any {
	confMap := make(map[string]any)

	encoding.MustJSONRoundTrip(config, &confMap)

	// Not sure why `omitempty` doesn't do the trick here, but having `ignore: {}` in the config for each
	// rule is annoying noice when printed from Rego.
	for categoryName, category := range config.Rules {
		for ruleName, rule := range category {
			if rule.Ignore == nil {
				// casts should be safe here as the structure is copied from the config struct
				//nolint:forcetypeassert
				delete(confMap["rules"].(map[string]any)[categoryName].(map[string]any)[ruleName].(map[string]any), "ignore")
			}
		}
	}

	return confMap
}

func (rule *Rule) MarshalJSON() ([]byte, error) {
	result, err := rule.MarshalYAML()
	if err != nil {
		return nil, fmt.Errorf("marshalling rule failed %w", err)
	}

	return encoding.JSON().Marshal(&result)
}

func (rule *Rule) UnmarshalJSON(data []byte) error {
	result, err := encoding.JSONUnmarshalTo[map[string]any](data)
	if err != nil {
		return fmt.Errorf("unmarshalling rule failed %w", err)
	}

	return rule.mapToConfig(result)
}

func (rule *Rule) MarshalYAML() (any, error) {
	result := map[string]any{keyLevel: rule.Level}

	if rule.Ignore != nil && len(rule.Ignore.Files) != 0 {
		result[keyIgnore] = rule.Ignore
	}

	for key, val := range rule.Extra {
		if key != keyIgnore && key != keyLevel {
			result[key] = val
		}
	}

	return result, nil
}

func (rule *Rule) UnmarshalYAML(value *yaml.Node) error {
	var result map[string]any
	if err := value.Decode(&result); err != nil {
		return fmt.Errorf("unmarshalling rule failed %w", err)
	}

	return rule.mapToConfig(result)
}

// Note that this function will mutate the result map. This isn't a problem right now
// as we only use this after unmarshalling, but if we use this for other purposes later
// we need to make a copy of the map first.
func (rule *Rule) mapToConfig(result any) error {
	ruleMap, ok := result.(map[string]any)
	if !ok {
		return errors.New("result was not a map")
	}

	level, ok := ruleMap[keyLevel].(string)
	if ok {
		rule.Level = level
	}

	if ignore, ok := ruleMap[keyIgnore]; ok {
		dst, err := encoding.JSONRoundTripTo[Ignore](ignore)
		if err != nil {
			return fmt.Errorf("unmarshalling rule ignore failed: %w", err)
		}

		rule.Ignore = &dst
	}

	rule.Extra = ruleMap

	delete(rule.Extra, keyLevel)
	delete(rule.Extra, keyIgnore)

	return nil
}

func (f *Features) IsZero() bool {
	return f == nil || (f.Remote == nil)
}

func GetPotentialRoots(paths ...string) ([]string, error) {
	var err error

	absDirPaths := make([]string, len(paths))

	for i, path := range paths {
		abs := path

		if !filepath.IsAbs(abs) {
			if abs, err = filepath.Abs(path); err != nil {
				return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
			}
		}

		if rio.IsDir(abs) {
			absDirPaths[i] = abs
		} else {
			absDirPaths[i] = filepath.Dir(abs)
		}
	}

	dirMap := util.NewSet[string]()

	for _, dir := range absDirPaths {
		brds, err := FindBundleRootDirectories(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to find bundle root directories in %s: %w", dir, err)
		}

		dirMap.Add(brds...)
	}

	if dirMap.Size() == 0 {
		return absDirPaths, nil
	}

	return dirMap.Items(), nil
}
