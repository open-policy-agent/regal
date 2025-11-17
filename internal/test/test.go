package test

import (
	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/parse"
	"github.com/open-policy-agent/regal/pkg/rules"
)

func InputPolicy(filename, policy string) *rules.Input {
	content := map[string]string{filename: policy}
	modules := map[string]*ast.Module{filename: parse.MustParseModule(policy)}
	input := rules.NewInput(content, modules)

	return &input
}
