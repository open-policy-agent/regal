# METADATA
# description: |
#   the capabilities package unsurprisingly helps rule authors deal
#   with capabilities, which may have been provided by either Regal,
#   or customized by end-users via configuration
package regal.capabilities

import data.regal.config

default provided := {}

# METADATA
# description: |
#   The capabilities object for Regal itself. Use `config.capabilities`
#   to get the capabilities for the target environment (i.e. the policies
#   getting linted).
# scope: document
provided := data.internal.capabilities

# METADATA
# description: true if `object.keys` is available
has_object_keys if "object.keys" in config.builtin_names

# METADATA
# description: true if `strings.count` is available
has_strings_count if "strings.count" in config.builtin_names

# METADATA
# description: true if `sprintf` is available
# scope: document
has_sprintf if "sprintf" in config.builtin_names

# if if if!
# METADATA
# description: true if the `if` keyword is available
# scope: document
has_if if "if" in config.capabilities.future_keywords
has_if if has_rego_v1_feature
has_if if is_opa_v1

# METADATA
# description: true if the `contains` keyword is available
# scope: document
has_contains if "contains" in config.capabilities.future_keywords
has_contains if has_rego_v1_feature
has_contains if is_opa_v1

# METADATA
# description: true if `rego.v1` is available
has_rego_v1_feature if "rego_v1_import" in config.capabilities.features

# METADATA
# description: true if `OPA 1.0+` policy is targeted
has_string_interpolation if "template_strings" in config.capabilities.features

# METADATA
# description: true if `OPA 1.0+` policy is targeted
is_opa_v1 if "rego_v1" in config.capabilities.features
