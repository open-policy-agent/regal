package modify

import (
	"strings"
	"testing"
)

func TestSetKey(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		yamlContent string
		path        []string
		value       any
		expected    string
	}{
		"set missing rule level to ignore": {
			yamlContent: `rules:
  style:
    existing-rule:
      level: error`,
			path:  []string{"rules", "style", "new-rule", "level"},
			value: "ignore",
			expected: `rules:
  style:
    existing-rule:
      level: error
    new-rule:
      level: ignore`,
		},
		"replace existing rule level": {
			yamlContent: `rules:
  style:
    prefer-snake-case:
      level: error`,
			path:  []string{"rules", "style", "prefer-snake-case", "level"},
			value: "ignore",
			expected: `rules:
  style:
    prefer-snake-case:
      level: ignore`,
		},
		"preserves comments": {
			yamlContent: `# Configuration file
rules:
  style:
    prefer-snake-case:
      level: error  # Important rule
      # Additional configuration
      ignore:
        files:
          - "*_test.rego"`,
			path:  []string{"rules", "style", "line-length", "level"},
			value: "ignore",
			expected: `# Configuration file
rules:
  style:
    prefer-snake-case:
      level: error # Important rule
      # Additional configuration
      ignore:
        files:
          - "*_test.rego"
    line-length:
      level: ignore`,
		},
		"empty rules structure": {
			yamlContent: `rules: {}`,
			path:        []string{"rules", "bugs", "use-assignment-operator", "level"},
			value:       "ignore",
			expected: `rules:
  bugs:
    use-assignment-operator:
      level: ignore`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := SetKey(tc.yamlContent, tc.path, tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if exp, got := strings.TrimSpace(tc.expected), strings.TrimSpace(result); exp != got {
				t.Errorf("result doesn't match expected.\nExpected:\n%s\n\nGot:\n%s", exp, got)
			}
		})
	}
}

func TestSetKeyErrors(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		yamlContent   string
		path          []string
		value         any
		expectedError string
	}{
		"empty path": {
			yamlContent:   "rules: {}",
			path:          []string{},
			value:         "value",
			expectedError: "path cannot be empty",
		},
		"unsupported type": {
			yamlContent:   "rules: {}",
			path:          []string{"rules", "test"},
			value:         123,
			expectedError: "unsupported type",
		},
		"invalid YAML": {
			yamlContent:   "invalid: yaml: [[[",
			path:          []string{"test"},
			value:         "value",
			expectedError: "failed to parse YAML",
		},
		"cannot navigate into scalar value": {
			yamlContent: `rules:
  style: "this-is-a-string"`,
			path:          []string{"rules", "style", "new-rule", "level"},
			value:         "ignore",
			expectedError: `cannot navigate into key "style": expected mapping but found yaml.Kind`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := SetKey(tc.yamlContent, tc.path, tc.value)
			if err == nil {
				t.Errorf("expected error for %s", name)
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error containing %q, got: %v", tc.expectedError, err)
			}
		})
	}
}
