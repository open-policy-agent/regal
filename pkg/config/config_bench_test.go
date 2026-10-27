package config_test

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/config"
)

// 1238809 ns/op	 3853793 B/op	   13599 allocs/op // JSON roundtrip
// _847580 ns/op	 3638023 B/op	    3948 allocs/op // *yaml.Node encoding Rules directly
func BenchmarkMarshalYAMLDefaultConfig(b *testing.B) {
	defaultConf := must.Return(os.ReadFile("../../bundle/regal/config/provided/data.yaml"))(b)

	var decoded config.Config
	if err := yaml.Unmarshal(defaultConf, &decoded); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_ = must.Return(yaml.Marshal(decoded))(b)
	}
}

// 224998 ns/op	  227877 B/op	    4109 allocs/op
// 147102 ns/op	  148041 B/op	    2216 allocs/op
func BenchmarkUnmarshalYAMLDefaultConfig(b *testing.B) {
	defaultConf := must.Return(os.ReadFile("../../bundle/regal/config/provided/data.yaml"))(b)

	for b.Loop() {
		var decoded config.Config
		if err := yaml.Unmarshal(defaultConf, &decoded); err != nil {
			b.Fatal(err)
		}
	}
}

// 35534 ns/op	   53264 B/op	    1145 allocs/op
// 16976 ns/op	   24713 B/op	     234 allocs/op
func BenchmarkUnmarshalVariedConfig(b *testing.B) {
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
project:
  rego-version: 0
  roots:
    - foo/bar
    - baz
    - path: bar/baz
    - path: v1
      rego-version: 1`)

	for b.Loop() {
		_ = testutil.MustUnmarshalYAML[config.Config](b, bs)
	}
}

func BenchmarkWithDefaultsFromBundle(b *testing.B) {
	b.Run("MergedWithDefaults", func(b *testing.B) {
		var userConfig *config.Config
		for b.Loop() {
			_ = userConfig.WithRegalDefaults()
		}
	})

	b.Run("MergedWithDefaults empty user config", func(b *testing.B) {
		userConfig := &config.Config{}
		for b.Loop() {
			_ = userConfig.WithRegalDefaults()
		}
	})

	b.Run("MergedWithDefaults with global defaults", func(b *testing.B) {
		userConfig := &config.Config{Defaults: config.Defaults{Global: config.Default{Level: "ignore"}}}
		for b.Loop() {
			_ = userConfig.WithRegalDefaults()
		}
	})
}
