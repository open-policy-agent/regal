package config

import (
	"fmt"

	"dario.cat/mergo"

	"github.com/open-policy-agent/opa/v1/bundle"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

// WithDefaultsFromBundle returns a complete configuration based on the default
// (embedded) configuration and (optionally) a user-provided configuration. Panics if the
// embedded configuration can't be read, as that is a hard dependency.
func WithDefaultsFromBundle(regalBundle *bundle.Bundle, userConfig *Config) (Config, error) {
	bundled := util.Must(util.SearchMap(regalBundle.Data, "regal", "config", "provided"))

	bundledConf, ok := bundled.(map[string]any)
	if !ok {
		panic("expected 'rules' of object type in default configuration")
	}

	defaultConfig := util.Must(encoding.JSONRoundTripTo[Config](bundledConf))
	if userConfig == nil {
		defaultConfig.Capabilities = CapabilitiesForThisVersion()

		return defaultConfig, nil
	}

	providedRuleLevels := providedConfLevels(&defaultConfig)

	if err := mergo.Merge(&defaultConfig, userConfig, mergo.WithOverride); err != nil {
		return Config{}, fmt.Errorf("failed to merge user config: %w", err)
	}

	if defaultConfig.Capabilities == nil {
		defaultConfig.Capabilities = CapabilitiesForThisVersion()
	}

	// adopt user rule levels based on config and defaults
	// If the user configuration contains rules with the level unset, copy the level from the provided configuration
	extractUserRuleLevels(userConfig, &defaultConfig, providedRuleLevels)

	return defaultConfig, nil
}

// extractUserRuleLevels uses defaulting config and per-rule levels from user configuration to set the level for each
// rule.
func extractUserRuleLevels(userConfig *Config, mergedConf *Config, providedRuleLevels map[string]string) {
	for categoryName, rulesByCategory := range mergedConf.Rules {
		for ruleName, rule := range rulesByCategory {
			var (
				providedLevel             string
				ok, userHasConfiguredRule bool
			)

			if providedLevel, ok = providedRuleLevels[ruleName]; !ok {
				continue
			}

			// use the level from the provided configuration as the fallback
			selectedRuleLevel := providedLevel

			if _, ok := userConfig.Rules[categoryName]; ok {
				_, userHasConfiguredRule = userConfig.Rules[categoryName][ruleName]
			}

			if userHasConfiguredRule && userConfig.Rules[categoryName][ruleName].Level != "" {
				// if the user config has a level for the rule, use that
				selectedRuleLevel = userConfig.Rules[categoryName][ruleName].Level
			} else if categoryDefault, ok := mergedConf.Defaults.Categories[categoryName]; ok {
				// if the config has a default level for the category, use that
				if categoryDefault.Level != "" {
					selectedRuleLevel = categoryDefault.Level
				}
			} else if mergedConf.Defaults.Global.Level != "" {
				// if the config has a global default level, use that
				selectedRuleLevel = mergedConf.Defaults.Global.Level
			}

			rule.Level = selectedRuleLevel
			mergedConf.Rules[categoryName][ruleName] = rule
		}
	}
}

// Copy the level of each rule from the provided configuration.
func providedConfLevels(conf *Config) map[string]string {
	ruleLevels := make(map[string]string)

	for categoryName, rulesByCategory := range conf.Rules {
		for ruleName := range rulesByCategory {
			// Note that this assumes all rules have unique names,
			// which we'll likely always want for provided rules
			ruleLevels[ruleName] = conf.Rules[categoryName][ruleName].Level
		}
	}

	return ruleLevels
}
