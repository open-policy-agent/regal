package regal.ast

# METADATA
# description: |
#   parses provided policy with all future keywords imported. Primarily for testing.
#   deprecated: use ast.policy instead
with_rego_v1(policy) := regal.parse_module("policy.rego", $"package policy\n\nimport rego.v1\n\n{policy}")

# METADATA
# description: parses provided policy with v0 syntax and no imports. Primarily for testing.
with_rego_v0(policy) := regal.parse_module("policy_v0.rego", $"package policy\n\n{policy}")

# METADATA
# description: parse provided snippet with a generic package declaration added
policy(snippet) := regal.parse_module("policy.rego", $"package policy\n\n{snippet}")
