package config

import (
	"cmp"
	"maps"
	"slices"
)

// WithRegalDefaults applies the embedded Regal defaults to the
// configuration, with any existing values preserved.
func (c *Config) WithRegalDefaults() *Config {
	return c.WithDefaults(Embedded())
}

func (c *Config) WithDefaults(defaults *Config) *Config {
	if c == nil || c == defaults {
		return defaults
	}

	return &Config{
		Rules:           c.Rules.withDefaults(defaults.Rules, c.Defaults),
		Features:        c.Features.withDefaults(defaults.Features),
		Ignore:          c.Ignore.withDefaults(defaults.Ignore),
		Capabilities:    c.Capabilities.withDefaults(defaults.Capabilities),
		CapabilitiesURL: c.CapabilitiesURL,
		Project:         c.Project,
	}
}

func (r Rules) withDefaults(defaults Rules, global Defaults) (m map[string]Category) {
	if global.IsZero() && (len(r) == 0 || maps.EqualFunc(r, defaults, Category.Equal)) {
		return defaults // no copy needed
	}

	// apply global defaults to the cloned defaults
	m = defaults.Copy()
	for name, category := range m {
		override := global.Level(name)
		if override == "" {
			continue
		}

		for ruleName, rule := range category {
			rule.Level = cmp.Or(override, rule.Level)
			category[ruleName] = rule
		}
	}

	for name, category := range r {
		if existing, ok := m[name]; ok {
			m[name] = category.withDefaults(existing, global.Level(name))
		} else {
			m[name] = category
		}
	}

	return m
}

func (c Category) withDefaults(defaults Category, defaultLevel string) (m Category) {
	m = maps.Clone(defaults)
	if defaultLevel != "" {
		for k, v := range m {
			m[k] = v.withDefaults(v, defaultLevel)
		}
	}

	for k, v := range c {
		if existingRule, ok := m[k]; ok {
			m[k] = v.withDefaults(existingRule, defaultLevel)
		} else {
			m[k] = v
		}
	}

	return m
}

func (r Rule) withDefaults(defaults Rule, defaultLevel string) (m Rule) {
	m.Level = cmp.Or(r.Level, defaultLevel, defaults.Level)
	m.Ignore = cmp.Or(r.Ignore, defaults.Ignore)

	if r.Extra == nil {
		m.Extra = defaults.Extra
	} else {
		if defaults.Extra != nil {
			m.Extra = maps.Clone(defaults.Extra)
			maps.Copy(m.Extra, r.Extra)
		} else {
			m.Extra = maps.Clone(r.Extra)
		}
	}

	return m
}

func (f *Features) withDefaults(defaults *Features) *Features {
	if f == nil {
		return defaults
	}

	if f.Remote == nil {
		return &Features{Remote: defaults.Remote}
	}

	return &Features{Remote: f.Remote}
}

func (i Ignore) withDefaults(defaults Ignore) (m Ignore) {
	if m.Files = i.Files; m.Files == nil {
		m.Files = defaults.Files
	}

	return m
}

func (c *Capabilities) withDefaults(defaults *Capabilities) *Capabilities {
	// TODO: this isn't properly implemented but also potentially not
	// needed as this will be populated when use config is parsed
	if c == nil {
		return defaults
	}

	if c.Builtins == nil {
		c.Builtins = defaults.Builtins
	}

	if c.FutureKeywords == nil {
		c.FutureKeywords = slices.Clone(defaults.FutureKeywords)
	}

	if c.Features == nil {
		c.Features = slices.Clone(defaults.Features)
	}

	return c
}
