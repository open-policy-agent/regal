package refs

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/util"

	rast "github.com/open-policy-agent/regal/internal/ast"
	"github.com/open-policy-agent/regal/internal/lsp/types"
)

// DefinedInModule returns a map of refs and details about them to be used in completions that
// were found in the given module.
func DefinedInModule(module *ast.Module, builtins map[string]*ast.Builtin) map[string]types.Ref {
	modKey := module.Package.Path.String()

	// first, create a reference for the package using the metadata, if present
	packagePrettyName := strings.TrimPrefix(module.Package.Path.String(), "data.")
	packageDescription := defaultDescription(packagePrettyName)

	packageAnnotation, ok := findAnnotationForPackage(module)
	if ok {
		packageDescription = documentAnnotatedRef(packageAnnotation)
	}

	items := map[string]types.Ref{
		modKey: {Label: modKey, Kind: types.Package, Detail: "Package", Description: packageDescription},
	}

	// Create groups of rules and functions sharing the same name
	ruleGroups := make(map[string][]*ast.Rule, len(module.Rules))
	for _, rule := range module.Rules {
		if name := rule.Head.Ref().String(); !strings.HasPrefix(name, "test_") {
			ruleGroups[name] = append(ruleGroups[name], rule)
		}
	}

	for g, rs := range ruleGroups {
		if len(rs) == 0 {
			continue // this should not happen, but we depend on rules being present below
		}

		isConstant := true

		for _, r := range rs {
			if !rast.IsConstant(r) {
				isConstant = false

				break
			}
		}

		kind := types.Rule

		switch {
		case isConstant:
			kind = types.ConstantRule
		case rs[0].Head.Args != nil:
			kind = types.Function
		}

		ruleDescription := defaultDescription(g)
		if ruleAnnotation, ok := findAnnotationForRuleGroup(rs); ok {
			ruleDescription = documentAnnotatedRef(ruleAnnotation)
		}

		ruleKey := fmt.Sprintf("%s.%s", modKey, g)

		items[ruleKey] = types.Ref{
			Kind:        kind,
			Label:       ruleKey,
			Detail:      rast.GetRuleDetail(rs[0], builtins),
			Description: ruleDescription,
		}
	}

	return items
}

func defaultDescription(name string) string {
	return fmt.Sprintf(`# %s

See [METADATA Documentation](https://www.openpolicyagent.org/docs/policy-language/#metadata)
to add more detail.`, name)
}

func findAnnotationForPackage(m *ast.Module) (*ast.Annotations, bool) {
	var subPackageIndexes []int

	for i, a := range m.Annotations {
		if a.Scope == "package" {
			return a, true
		}

		if a.Scope == "subpackages" {
			subPackageIndexes = append(subPackageIndexes, i)
		}
	}

	if len(subPackageIndexes) > 0 {
		// subpackages are also permitted so they can be shown for the top level
		// package in completions. However, package annotations take precedence.
		return m.Annotations[subPackageIndexes[0]], true
	}

	return nil, false
}

// findAnnotationForRuleGroup looks for an annotation on any of the rules in the group,
// if one is found, the first one is returned.
func findAnnotationForRuleGroup(rs []*ast.Rule) (*ast.Annotations, bool) {
	for _, r := range rs {
		for _, a := range r.Annotations {
			if a.Scope == "rule" {
				return a, true
			}
		}
	}

	return nil, false
}

func documentAnnotatedRef(selectedAnnotation *ast.Annotations) string {
	var sb strings.Builder

	if selectedAnnotation.Title != "" {
		sb.WriteString("# ")
		sb.WriteString(selectedAnnotation.Title)
		sb.WriteString("\n\n")
	}

	if selectedAnnotation.Description != "" {
		sb.WriteString("**Description**:\n\n")
		sb.WriteString(selectedAnnotation.Description)
		sb.WriteString("\n\n")
	}

	if len(selectedAnnotation.Authors) > 0 {
		sb.WriteString("**Authors**:\n\n")

		for _, author := range selectedAnnotation.Authors {
			sb.WriteString("* ")

			if author.Name != "" {
				sb.WriteString(author.Name)
			}

			if author.Email != "" {
				sb.WriteString(" ")
				sb.WriteString("<")
				sb.WriteString(author.Email)
				sb.WriteString(">")
			}

			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(selectedAnnotation.Organizations) > 0 {
		sb.WriteString("**Organizations**:\n\n")

		for _, org := range selectedAnnotation.Organizations {
			sb.WriteString("* ")
			sb.WriteString(org)
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(selectedAnnotation.RelatedResources) > 0 {
		sb.WriteString("**Related Resources**:\n\n")

		for _, resource := range selectedAnnotation.RelatedResources {
			sb.WriteString("* [")

			if resource.Description != "" {
				sb.WriteString(resource.Description)
			} else {
				sb.WriteString(strings.Replace(resource.Ref.String(), "http://", "", 1))
			}

			sb.WriteString("](")
			sb.WriteString(resource.Ref.String())
			sb.WriteString(")\n")
		}

		sb.WriteString("\n")
	}

	if len(selectedAnnotation.Custom) > 0 {
		sb.WriteString("**Custom**:\n\n```yaml\n")

		bs, err := yaml.Marshal(selectedAnnotation.Custom)
		if err != nil {
			sb.WriteString("Error generating custom section")
		} else {
			sb.WriteString(util.ByteSliceToString(bytes.TrimSpace(bs)))
		}

		sb.WriteString("\n```\n")
	}

	return sb.String()
}
