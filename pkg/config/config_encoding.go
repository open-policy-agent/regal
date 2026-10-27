package config

import (
	"cmp"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/capabilities"
	"github.com/open-policy-agent/regal/internal/encoding/util/node"
	"github.com/open-policy-agent/regal/internal/encoding/util/read"
	"github.com/open-policy-agent/regal/internal/encoding/util/write"
)

// marshallingIntermediary is used to contain config data in a format that is used during unmarshalling.
// The internally loaded config data layout differs from the user-defined YAML.
type marshallingIntermediary struct {
	// rules are unmarshalled as any since the defaulting needs to be extracted from here
	// and configured elsewhere in the struct.
	Rules        yaml.Node `yaml:"rules"`
	Project      *Project  `yaml:"project"`
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

func (c *Config) String() string {
	yamlBytes, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("error marshaling config: %v", err)
	}

	return outil.ByteSliceToString(yamlBytes)
}

func (c Config) MarshalYAML() (any, error) {
	a := struct {
		Rules        *yaml.Node    `yaml:"rules"`
		Project      *Project      `yaml:"project,omitempty"`
		Capabilities *Capabilities `yaml:"capabilities,omitempty"`
		Ignore       Ignore        `yaml:"ignore,omitempty"`
		Features     *Features     `yaml:"features,omitempty"`
	}{
		Rules:        &yaml.Node{Kind: yaml.MappingNode},
		Project:      c.Project,
		Capabilities: c.Capabilities,
		Ignore:       c.Ignore,
		Features:     c.Features,
	}

	if c.Defaults.Global.Level != "" {
		level := c.Defaults.Global.Level
		a.Rules.Content = append(a.Rules.Content, node.String("default"), node.LevelMapping(level))
	}

	for _, catName := range outil.KeysSorted(c.Rules) {
		catRules := c.Rules[catName]
		a.Rules.Content = append(a.Rules.Content, node.String(catName))

		ruleNodes := make([]*yaml.Node, 0, len(catRules))

		var defaultLevel string

		if c.Defaults.Categories != nil {
			if catDefault, ok := c.Defaults.Categories[catName]; ok {
				defaultLevel = catDefault.Level
				ruleNodes = append(ruleNodes, node.String("default"), node.LevelMapping(defaultLevel))
			}
		}

		for _, ruleName := range outil.KeysSorted(catRules) {
			ruleConf := catRules[ruleName]

			preAlloc := 2 + len(ruleConf.Extra)*2
			if ruleConf.Ignore != nil {
				preAlloc += 2
			}

			confNodes := append(make([]*yaml.Node, 0, preAlloc), node.Level(cmp.Or(ruleConf.Level, defaultLevel))...)

			if ruleConf.Ignore != nil && len(ruleConf.Ignore.Files) > 0 {
				confNodes = append(confNodes,
					node.String("ignore"),
					node.Mapping(node.String("files"), node.ArrayOfString(ruleConf.Ignore.Files)),
				)
			}

			for _, key := range outil.KeysSorted(ruleConf.Extra) {
				if key != "level" && key != "ignore" {
					value := ruleConf.Extra[key]

					attr := &yaml.Node{}
					if err := attr.Encode(value); err != nil {
						return nil, err
					}

					confNodes = append(confNodes, node.String(key), attr)
				}
			}

			ruleNodes = append(ruleNodes, node.String(ruleName), node.Mapping(confNodes...))
		}

		a.Rules.Content = append(a.Rules.Content, node.Mapping(ruleNodes...))
	}

	return a, nil
}

func (c *Config) UnmarshalYAML(value *yaml.Node) (err error) {
	var result marshallingIntermediary

	if err := value.Decode(&result); err != nil {
		return fmt.Errorf("unmarshalling config failed %w", err)
	}

	if err := extractRulesAndDefaults(c, &result); err != nil {
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
			return fmt.Errorf("failed to determine absolute path of %s: %w", capabilitiesFile, err)
		}

		// prepending a / is done here to ensure that windows drive letter paths
		// are parsed as paths and not host:ports in URLs.
		capabilitiesURL = "file://" + outil.WithPrefix(absfp, "/")
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
		c.Capabilities.Builtins[plusBuiltin.Name] = fromOPABuiltin(plusBuiltin)
	}

	// feature defaults
	if result.Features.RemoteFeatures.CheckVersion {
		c.Features = &Features{Remote: &RemoteFeatures{CheckVersion: true}}
	}

	return nil
}

func (c Capabilities) MarshalYAML() (any, error) {
	builtins := &yaml.Node{Kind: yaml.MappingNode, Content: make([]*yaml.Node, 0, len(c.Builtins)*2)}
	for k, v := range c.Builtins {
		builtins.Content = append(builtins.Content, node.String(k), node.Mapping(node.String("decl"),
			node.Mapping(
				node.String("result"), node.String(v.Decl.Result),
				node.String("args"), node.ArrayOfString(v.Decl.Args),
			),
		))
	}

	return node.Mapping(
		node.String("builtins"), builtins,
		node.String("future_keywords"), node.ArrayOfString(c.FutureKeywords),
		node.String("features"), node.ArrayOfString(c.Features),
	), nil
}

func (r Rule) MarshalJSONTo(enc *jsontext.Encoder) error {
	enc.WriteToken(jsontext.BeginObject)
	write.StringField(enc, keyLevel, r.Level)

	if r.Ignore != nil && len(r.Ignore.Files) > 0 {
		write.String(enc, "ignore")
		enc.WriteToken(jsontext.BeginObject)
		write.ArrayOfStringField(enc, "files", r.Ignore.Files)
		enc.WriteToken(jsontext.EndObject)
	}

	for key, val := range r.Extra {
		if key != keyIgnore && key != keyLevel {
			_ = write.Field(enc, key, val)
		}
	}

	return enc.WriteToken(jsontext.EndObject)
}

func (r *Rule) UnmarshalYAML(value *yaml.Node) (err error) {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected rule as map, got %v", value.Kind)
	}

	var currKey string

	for i, node := range value.Content {
		if node.Kind == yaml.ScalarNode && i%2 == 0 {
			currKey = node.Value
		} else {
			switch currKey {
			case keyLevel:
				r.Level = node.Value
			case keyIgnore:
				if node.Kind == yaml.MappingNode {
					if r.Ignore == nil {
						r.Ignore = &Ignore{}
					}

					for j := 0; j < len(node.Content); j += 2 {
						keyNode := node.Content[j]
						valNode := node.Content[j+1]

						if keyNode.Value == "files" && valNode.Kind == yaml.SequenceNode {
							r.Ignore.Files = make([]string, len(valNode.Content))
							for k, fileNode := range valNode.Content {
								r.Ignore.Files[k] = fileNode.Value
							}
						}
					}
				}
			default:
				if r.Extra == nil {
					r.Extra = make(map[string]any)
				}

				if currKey != keyIgnore && currKey != keyLevel {
					// TODO: Only decode non-scalars
					var extra any
					if err := node.Decode(&extra); err != nil {
						return err
					}

					r.Extra[currKey] = extra
				}
			}
		}
	}

	return nil
}

func (r *Rule) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return read.Object(dec, func(dec *jsontext.Decoder, key string) (err error) {
		switch key {
		case keyLevel:
			r.Level, err = read.String(dec)
		case keyIgnore:
			if r.Ignore == nil {
				r.Ignore = &Ignore{}
			}

			err = json.UnmarshalDecode(dec, r.Ignore)
		default:
			if r.Extra == nil {
				r.Extra = make(map[string]any)
			}

			var val any

			err = json.UnmarshalDecode(dec, &val)
			r.Extra[key] = val
		}

		return err
	})
}

func (r *Root) UnmarshalYAML(value *yaml.Node) (err error) {
	// A project root can either be provided as a string or an object. If it's a
	// string, use it to set the Path field of the Root struct (and no version).
	if value.Kind == yaml.ScalarNode {
		r.Path = value.Value
	} else {
		type Alias Root

		err = value.Decode((*Alias)(r))
	}

	return err
}

func (e ExtraAttributes) MarshalJSONTo(enc *jsontext.Encoder) error {
	return write.Map(enc, e)
}

// extractRulesAndDefaults is a helper to load rules and defaults from the raw config data.
func extractRulesAndDefaults(config *Config, result *marshallingIntermediary) error {
	// in order to support wildcard 'default' configs, we
	// have some hooks in this unmarshalling process to load these.
	config.Rules = make(map[string]Category)
	config.Defaults = Defaults{Categories: make(map[string]Default, 8)}

	var currCategory, currRule string

	for _, node := range result.Rules.Content {
		if node.Kind == yaml.ScalarNode {
			currCategory = node.Value
			if node.Value != "default" {
				config.Rules[currCategory] = Category{}
			}
		} else {
			if currCategory == "default" {
				if len(node.Content) == 2 {
					config.Defaults.Global = Default{Level: node.Content[1].Value}
				}

				continue
			}

			ruleNodes := node.Content
			for _, ruleNode := range ruleNodes {
				switch {
				case ruleNode.Kind == yaml.ScalarNode:
					currRule = ruleNode.Value
				case currRule == "default":
					if len(ruleNode.Content) == 2 {
						config.Defaults.Categories[currCategory] = Default{Level: ruleNode.Content[1].Value}
					}
				default:
					rule := Rule{}
					if err := (&rule).UnmarshalYAML(ruleNode); err != nil {
						return err
					}

					config.Rules[currCategory][currRule] = rule
				}
			}
		}
	}

	return nil
}
