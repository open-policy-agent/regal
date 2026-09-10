package modify

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	outil "github.com/open-policy-agent/opa/v1/util"
)

var (
	errEmptyPath      = errors.New("path cannot be empty")
	errInvalidDoc     = errors.New("invalid document structure")
	errRootNotMapping = errors.New("root is not a mapping")

	yamlBoolTrueNode  = &yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"}
	yamlBoolFalseNode = &yaml.Node{Kind: yaml.ScalarNode, Value: "false", Tag: "!!bool"}
)

// SetKey sets a key at the given path to the specified value.
// Comments are preserved, but indentation is always 2.
func SetKey(yamlContent string, path []string, value any) (string, error) {
	if len(path) == 0 {
		return "", errEmptyPath
	}

	var root yaml.Node
	if err := yaml.Unmarshal(outil.StringToByteSlice(yamlContent), &root); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return "", errInvalidDoc
	}

	document := root.Content[0]
	if document.Kind != yaml.MappingNode {
		return "", errRootNotMapping
	}

	current := document
	// navigate to the parent of the final key
	for _, key := range path[:len(path)-1] {
		found := false

		// YAML mappings store key-value pairs as alternating nodes: [key1, value1, key2, value2, ...]
		for j := 0; j < len(current.Content); j += 2 {
			if current.Content[j].Value != key {
				continue
			}

			// ensure the value node is a mapping - error if it's a scalar
			if current.Content[j+1].Kind != yaml.MappingNode {
				return "", fmt.Errorf("cannot navigate into key %q: expected mapping but found %T", key, current.Content[j+1].Kind)
			}

			current = current.Content[j+1]
			current.Style = 0 // force the use of the default style, rather than compact style
			found = true

			break
		}

		if !found { // create ancestor node
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
			valueNode := &yaml.Node{Kind: yaml.MappingNode, Style: 0} // default style rather than compact
			current.Content = append(current.Content, keyNode, valueNode)
			current = valueNode
		}
	}

	finalKey := path[len(path)-1]

	buf := new(strings.Builder)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)

	// check if the final key already exists and update it
	for i := 0; i < len(current.Content); i += 2 {
		if current.Content[i].Value != finalKey {
			continue
		}

		// update the existing value
		valueNode, err := createNodeFromValue(value)
		if err != nil {
			return "", fmt.Errorf("failed to set key: %w", err)
		}

		current.Content[i+1] = valueNode

		return encodeAndClose(buf, enc, &root)
	}

	// create the new key if it doesn't exist
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: finalKey}

	valueNode, err := createNodeFromValue(value)
	if err != nil {
		return "", fmt.Errorf("failed to set key: %w", err)
	}

	current.Content = append(current.Content, keyNode, valueNode)

	return encodeAndClose(buf, enc, &root)
}

func encodeAndClose(buf *strings.Builder, enc *yaml.Encoder, root *yaml.Node) (string, error) {
	if err := enc.Encode(root); err != nil {
		return "", fmt.Errorf("failed to encode YAML: %w", err)
	}

	err := enc.Close()

	return buf.String(), err
}

// createNodeFromValue only supports bool and strings -> YAML node.
func createNodeFromValue(value any) (*yaml.Node, error) {
	switch v := value.(type) {
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v}, nil
	case bool:
		if v {
			return yamlBoolTrueNode, nil
		}

		return yamlBoolFalseNode, nil
	default:
		return nil, fmt.Errorf("unsupported type %T, only string and bool are supported", value)
	}
}
