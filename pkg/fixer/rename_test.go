package fixer

import (
	"path/filepath"
	"testing"
)

func TestRenameCandidate(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		oldName  string
		expected string
	}{
		"rename a policy file": {
			oldName:  "policy.rego",
			expected: "policy_1.rego",
		},
		"rename a policy file with existing increment": {
			oldName:  "policy_999.rego",
			expected: "policy_1000.rego",
		},
		"rename a policy file with existing increment and a number in the filename": {
			oldName:  "policy_123_999.rego",
			expected: "policy_123_1000.rego",
		},
		"rename a policy file in a dir": {
			oldName:  "/foo/policy.rego",
			expected: "/foo/policy_1.rego",
		},
		"rename a test file": {
			oldName:  "policy_test.rego",
			expected: "policy_1_test.rego",
		},
		"rename a test file with existing increment": {
			oldName:  "policy_999_test.rego",
			expected: "policy_1000_test.rego",
		},
		"rename a test file with existing increment and a number in the filename": {
			oldName:  "/foo/policy_123_999_test.rego",
			expected: "/foo/policy_123_1000_test.rego",
		},
		"rename a test file in a dir": {
			oldName:  "/foo/policy_test.rego",
			expected: "/foo/policy_1_test.rego",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := renameCandidate(tc.oldName)
			if actual != filepath.FromSlash(tc.expected) {
				t.Errorf("expected %s, got %s", tc.expected, actual)
			}
		})
	}
}
