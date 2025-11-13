# METADATA
# description: utility module to help determine if an update of Regal should be recommended
package update

default check["needs_update"] := false

# METADATA
# description: true if current version is behind latest version
# scope: document
check["needs_update"] if {
	current_version := trim(input.current_version, "v")

	semver.is_valid(current_version)
	semver.compare(current_version, trim(check.latest_version, "v")) == -1
}

# METADATA
# description: the latest version, as determined by the release server
# scope: document
check["latest_version"] := input.latest_version if input.latest_version != ""

check["latest_version"] := response.body.tag_name if {
	input.latest_version == ""

	response := http.send({"url": _release_server_url, "method": "GET"})
}

# METADATA
# description: the CTA message, if an update is available
# scope: document
check["cta"] := sprintf(
	"A new version of Regal is available (%s). You are running %s.\nSee %s for the latest release.\n",
	[check.latest_version, input.current_version, concat("", [_cta_url_prefix, check.latest_version])],
) if {
	check.needs_update
}

_release_server_url := concat("", [_release_server_host, _release_server_path])

default _release_server_host := "https://api.github.com"

_release_server_host := _ensure_http(trim_suffix(input.release_server_host, "/")) if input.release_server_host != ""

default _release_server_path := "/repos/open-policy-agent/regal/releases/latest"

_release_server_path := input.release_server_path if input.release_server_path != ""

default _cta_url_prefix := "https://github.com/open-policy-agent/regal/releases/tag/"

_cta_url_prefix := input.cta_url_prefix if input.cta_url_prefix != ""

_ensure_http(url) := url if startswith(url, "http")
_ensure_http(url) := concat("", ["https://", url]) if not startswith(url, "http")
