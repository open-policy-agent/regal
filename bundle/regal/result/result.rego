# METADATA
# description: |
#   utility functions related to return a result from linter rules
#   policy authors are encouraged to use these over manually building
#   the expected objects, as using these functions should continure to
#   work across upgrades — i.e. if the result format changes
package regal.result

import data.regal.config
import data.regal.util

# METADATA
# description: |
#  The result.aggregate function works similarly to `result.fail`, but instead of producing
#  a violation returns an entry to be aggregated for later evaluation. This is useful in
#  aggregate rules (and only in aggregate rules) as it provides a uniform format for
#  aggregate data entries. Example return value:
#
#  {
#      "aggregate_source": {
#          "package_path": ["a", "b", "c"],
#      },
#      "aggregate_data": {
#          "foo": "bar",
#          "baz": [1, 2, 3],
#      },
#  }
#
#  Note that the first argument, which was the metadata chain from the package, is
#  no longer used, but kept for compatibility reasons.
## regal ignore:argument-always-wildcard
aggregate(_, aggregate_data) := {
	"aggregate_source": {"package_path": [part.value |
		some i, part in input.package.path
		i > 0
	]},
	"aggregate_data": aggregate_data,
}

# METADATA
# description: |
#   helper function to call when building the "return value" for the `report` in any linter rule —
#   recommendation being that both built-in rules and custom rules use this in favor of building the
#   result by hand
# scope: document

# METADATA
# description: provided rules, i.e. regal.rules.category.title
fail(metadata, details) := violation if {
	is_array(metadata) # from rego.metadata.chain()

	some link in metadata
	link.annotations.scope == "package"

	some category, title
	["regal", "rules", category, title] = link.path

	annotation := object.union(link.annotations, {
		"custom": {"category": category},
		"title": title,
		"related_resources": _related_resources(link.annotations, category, title),
	})

	violation := _fail_annotated(annotation, details)
}

# METADATA
# description: custom rules, i.e. custom.regal.rules.category.title
fail(metadata, details) := violation if {
	is_array(metadata) # from rego.metadata.chain()

	some link in metadata
	link.annotations.scope == "package"

	some category, title
	["custom", "regal", "rules", category, title] = link.path

	annotation := object.union(link.annotations, {
		"custom": {"category": category},
		"title": title,
	})

	violation := _fail_annotated_custom(annotation, details)
}

# METADATA
# description: fallback case
fail(metadata, details) := _fail_annotated(metadata, details)

# METADATA
# description: |
#   creates a notice object, i.e. one used to inform users of things like a rule getting
#   ignored because the set capabilities does not include a dependency (like a built-in function)
#   needed by the rule
notice(metadata) := result if {
	rule_meta := metadata[0]

	some category, title
	["regal", "rules", category, title, "notices"] = rule_meta.path

	result := {
		"category": category,
		"description": rule_meta.annotations.description,
		"level": "notice",
		"title": title,
		"severity": rule_meta.annotations.custom.severity,
	}
}

# regal ignore:narrow-argument
_related_resources(annotations, _, _) := annotations.related_resources

_related_resources(annotations, category, title) := rr if {
	not annotations.related_resources
	rr := [{
		"description": "documentation",
		"ref": $"{config.docs.base_url}/{category}/{title}",
	}]
}

_fail_annotated(metadata, details) := violation if {
	is_object(metadata)
	with_location := object.union(metadata, details)
	category := with_location.custom.category
	with_category := object.union(with_location, {
		"category": category,
		"level": config.level_for_rule(category, metadata.title),
	})

	without_custom_and_scope := object.remove(with_category, ["custom", "scope", "schemas"])
	related_resources := _resource_urls(without_custom_and_scope.related_resources, category)

	violation := json.patch(
		without_custom_and_scope,
		[{"op": "replace", "path": "/related_resources", "value": related_resources}],
	)
}

_fail_annotated_custom(metadata, details) := violation if {
	is_object(metadata)
	with_location := object.union(metadata, details)
	category := with_location.custom.category
	with_category := object.union(with_location, {
		"category": category,
		"level": config.level_for_rule(category, metadata.title),
	})

	violation := object.remove(with_category, ["custom", "scope", "schemas"])
}

_resource_urls(related_resources, category) := [r |
	some item in related_resources
	r := object.union(item, {"ref": config.docs.resolve_url(item.ref, category)})
]

# Note that the `text` attribute always returns the entire line and *not*
# based on the location range. This is intentional, as the context is often
# needed when this is printed out in the console. LSP diagnostics however use
# the range and will highlight based on that rather than `text`.
_with_text(loc_obj) := loc if {
	loc := {"location": object.union(loc_obj, {
		"file": input.regal.file.name,
		"text": input.regal.file.lines[loc_obj.row - 1],
	})}
} else := {"location": loc_obj}

# METADATA
# description: |
#   returns a "normalized" location object from the location value found in the AST.
#   new code should most often use one of the ranged_ location functions instead, as
#   that will also include an `"end"` location attribute
# scope: document
location(x) := _with_text(util.to_location_object(x.location))
location(x) := _with_text(util.to_location_object(x[0].location)) if is_array(x)
location(x) := _with_text(util.to_location_object(x)) if is_string(x)

# METADATA
# description: creates a location where x is the start, and y is the end (calculated from `text`)
ranged_location_between(x, y) := object.union(
	location(x),
	{"location": {"end": location(y).location.end}},
)

# METADATA
# description: creates a location where the first term location is the start, and the last term location is the end
ranged_from_ref(ref) := ranged_location_between(ref[0], regal.last(ref))

# METADATA
# description: |
#   creates a ranged location where the start location is the left hand side of an infix
#   expression, like `"foo" == "bar"`, and the end location is the end of the infix operator
infix_expr_location(terms) := location(regex.replace(
	$"{terms[1].location}{terms[0].location}",
	`^(\d+:\d+:).+:(\d+:\d+)$`,
	`$1$2`,
))
