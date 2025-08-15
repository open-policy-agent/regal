package cmd

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/embeds"
	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/pkg/config"
)

type newRuleCommandParams struct {
	type_    string // 'type' is a keyword
	category string
	name     string
	output   string
}

type TemplateValues struct {
	Category     string
	NameOriginal string
	Name         string
	NameTest     string
}

var (
	categoryRegex = regexp.MustCompile(`^[a-z]+$`)
	nameRegex     = regexp.MustCompile(`^[a-z_]+[a-z0-9_\-]*$`)
)

//nolint:lll
func init() {
	newCommand := &cobra.Command{
		Hidden: true,
		Use:    "new <template>",
		Long: `Create a new resource according to the chosen template (currently only 'rule' available).

The new command is a development utility for scaffolding new resources for use by Regal.

An example of such a resource would be new linter rules, which could be created either for inclusion in Regal core, or custom rules for your organization or team.`,
	}

	params := newRuleCommandParams{}

	newRuleCommand := &cobra.Command{
		Use:   "rule [-t type] [-c category] [-n name]",
		Short: "Create new rule from template",
		Long: `Create a new linter rule, for inclusion in Regal or a custom rule for your organization or team.

Example:

regal new rule --type custom --category naming --name camel-case`,

		PreRunE: func(*cobra.Command, []string) error {
			log.SetOutput(os.Stdout)

			if params.type_ != "custom" && params.type_ != "builtin" {
				return fmt.Errorf("type must be 'custom' or 'builtin', got %v", params.type_)
			}

			if params.category == "" {
				return errors.New("category is required for rule")
			}

			if !categoryRegex.MatchString(params.category) {
				return errors.New("category must be a single word, using lowercase letters only")
			}

			if params.name == "" {
				return errors.New("name is required for rule")
			}

			if !nameRegex.MatchString(params.name) {
				return errors.New("name must consist only of lowercase letters, numbers, underscores and dashes")
			}

			params.output = cmp.Or(params.output, io.Getwd())

			return nil
		},

		Run: func(*cobra.Command, []string) {
			if err := scaffoldRule(params); err != nil {
				log.SetOutput(os.Stderr)
				log.Println(err)
				os.Exit(1)
			}
		},
	}

	newRuleCommand.Flags().StringVarP(&params.type_, "type", "t", "custom", "type of rule (custom or builtin)")
	newRuleCommand.Flags().StringVarP(&params.category, "category", "c", "", "category for rule")
	newRuleCommand.Flags().StringVarP(&params.name, "name", "n", "", "name of rule")
	newRuleCommand.Flags().StringVarP(&params.output, "output", "o", "", "output directory")

	newCommand.AddCommand(newRuleCommand)
	RootCommand.AddCommand(newCommand)
}

func scaffoldRule(params newRuleCommandParams) error {
	switch params.type_ {
	case "custom":
		return scaffoldCustomRule(params)
	case "builtin":
		steps := []func(newRuleCommandParams) error{scaffoldBuiltinRule, createBuiltinDocs, addToDataYAML}
		for _, f := range steps {
			if err := f(params); err != nil {
				return err
			}
		}

		return nil
	}

	return fmt.Errorf("unsupported type %v", params.type_)
}

func addToDataYAML(params newRuleCommandParams) error {
	dataFilePath := filepath.Join("bundle", "regal", "config", "provided", "data.yaml")

	yamlContent, err := os.ReadFile(dataFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("data.yaml file not found. " +
				"Please run this command from the top-level directory of the Regal repository")
		}

		return err
	}

	var existingConfig config.Config
	if err := yaml.Unmarshal(yamlContent, &existingConfig); err != nil {
		return err
	}

	existingConfig.Capabilities = nil

	// Check if the category already exists in rules
	if existingConfig.Rules[params.category] == nil {
		existingConfig.Rules[params.category] = make(map[string]config.Rule)
	}

	// Assign a new rule/level value to the Category map
	existingConfig.Rules[params.category][params.name] = config.Rule{Level: "error"}

	// Sort the map keys alphabetically (categories)
	for _, cat := range util.KeysSorted(existingConfig.Rules) {
		// Sort rule names within each category alphabetically
		sortedRuleNames := util.KeysSorted(existingConfig.Rules[cat])

		sortedCategory := make(config.Category, len(sortedRuleNames))
		for _, ruleName := range sortedRuleNames {
			sortedCategory[ruleName] = existingConfig.Rules[cat][ruleName]
		}

		existingConfig.Rules[cat] = sortedCategory
	}

	var b bytes.Buffer

	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)

	if err = yamlEncoder.Encode(existingConfig); err != nil {
		return err
	}

	// Write the YAML content to file. If the output flag is set, this is not
	// necessarily the same file as the one we read from.
	out := filepath.Join(params.output, "bundle", "regal", "config", "provided", "data.yaml")

	return io.WithCreateRecursive(out, func(file *os.File) error {
		if _, err := file.Write(b.Bytes()); err == nil {
			log.Printf("Wrote configuration update to %s\n", out)
		}

		return err
	})
}

func scaffoldCustomRule(params newRuleCommandParams) error {
	return renderTemplates(params, filepath.Join(
		params.output, ".regal", "rules", "custom", "regal", "rules", params.category, params.name,
	))
}

func scaffoldBuiltinRule(params newRuleCommandParams) error {
	return renderTemplates(params, filepath.Join(
		params.output, "bundle", "regal", "rules", params.category, params.name,
	))
}

func renderTemplates(params newRuleCommandParams, dir string) error {
	templates := []string{
		fmt.Sprintf("templates/%[1]s/%[1]s.rego.tpl", params.type_),
		fmt.Sprintf("templates/%[1]s/%[1]s_test.rego.tpl", params.type_),
	}
	for _, name := range templates {
		tpl := filepath.Join(dir, templateFilename(params.name, name))

		err := io.WithCreateRecursive(tpl, func(file *os.File) error {
			return template.Must(template.ParseFS(embeds.EmbedTemplatesFS, name)).Execute(file, templateValues(params))
		})
		if err != nil {
			return fmt.Errorf("failed to render template %s: %w", name, err)
		}
	}

	log.Printf("Created %s rule %q in %s\n", params.type_, params.name, dir)

	return nil
}

func templateFilename(name, template string) string {
	if strings.Contains(template, "_test.rego") {
		return strings.ToLower(strings.ReplaceAll(name, "-", "_")) + "_test.rego"
	}

	return strings.ToLower(strings.ReplaceAll(name, "-", "_")) + ".rego"
}

func createBuiltinDocs(params newRuleCommandParams) error {
	out := filepath.Join(params.output, "docs", "rules", params.category, strings.ToLower(params.name)+".md")
	if err := os.MkdirAll(filepath.Dir(out), 0o770); err != nil {
		return err
	}

	tpl := template.New("builtin.md.tpl").Funcs(template.FuncMap{"ToUpper": strings.ToUpper})
	res := template.Must(tpl.ParseFS(embeds.EmbedTemplatesFS, "templates/builtin/builtin.md.tpl"))

	err := io.WithCreateRecursive(out, func(docFile *os.File) error {
		return res.Execute(docFile, templateValues(params))
	})
	if err != nil {
		return fmt.Errorf("failed to render builtin rule doc template: %w", err)
	}

	log.Printf("Created doc template for builtin rule %q at %s\n", params.name, out)

	return nil
}

func templateValues(params newRuleCommandParams) (tvs TemplateValues) {
	tvs.Category = params.category
	tvs.NameOriginal = params.name

	if strings.Contains(params.name, "-") || strings.Contains(params.name, "_") {
		dashedNameValue := strings.ReplaceAll(params.name, "_", "-")
		tvs.Name = `["` + dashedNameValue + `"]`
		tvs.NameTest = `["` + dashedNameValue + `_test"]`
	} else {
		tvs.Name = "." + params.name
		tvs.NameTest = "." + params.name + "_test"
	}

	return tvs
}
