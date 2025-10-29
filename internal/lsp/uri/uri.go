package uri

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/util"
)

// uris always use / as the uriSeparator, regardless of system.
const uriSeparator = "/"

var drivePattern = regexp.MustCompile(`^\/?([A-Za-z]):`)

// FromPath converts a file path to a URI for a given client.
// Since clients expect URIs to be in a specific format, this function
// will convert the path to the appropriate format for the client.
func FromPath(client clients.Identifier, path string) string {
	path = strings.TrimPrefix(path, "file://")
	path = strings.TrimPrefix(path, uriSeparator)

	var driveLetter string
	if matches := drivePattern.FindStringSubmatch(path); len(matches) > 0 {
		driveLetter = matches[1] + ":"
		path = strings.TrimPrefix(path, driveLetter)
	}

	parts := strings.Split(filepath.ToSlash(path), uriSeparator)
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(url.QueryEscape(part), "+", "%20")
	}

	if client == clients.IdentifierVSCode && driveLetter != "" {
		driveLetter = url.QueryEscape(driveLetter)
	}

	return "file:///" + driveLetter + strings.Join(parts, uriSeparator)
}

// ToPath converts a URI to a file path from a format for a given client.
// Some clients represent URIs differently, and so this function exists to convert
// client URIs into a standard file paths.
func ToPath(client clients.Identifier, uri string) string {
	// if it looks like uri was a file URI, then there might be encoded characters in the path
	path, hadPrefix := strings.CutPrefix(uri, "file://")
	if hadPrefix {
		// if it looks like a URI, then try and decode the path
		if decodedPath, err := url.QueryUnescape(path); err == nil {
			path = decodedPath
		}
	}

	// handling case for windows when the drive letter is set
	if drivePattern.MatchString(path) {
		path = strings.TrimPrefix(path, uriSeparator)
	}

	// Convert path to use system separators
	return filepath.FromSlash(path)
}

// ToRelativePath converts a URI to a file path relative to the given workspace root URI.
func ToRelativePath(client clients.Identifier, uri, workspaceRootURI string) string {
	absolutePath := ToPath(client, uri)
	workspaceRootPath := ToPath(client, workspaceRootURI)

	// Ensure workspace root path has trailing separator for consistent trimming
	if workspaceRootPath != "" {
		workspaceRootPath = util.EnsureSuffix(workspaceRootPath, string(filepath.Separator))
	}

	return strings.TrimPrefix(absolutePath, workspaceRootPath)
}

// FromRelativePath creates a URI from a relative path and workspace root URI.
func FromRelativePath(client clients.Identifier, relativePath, workspaceRootURI string) string {
	workspaceRootPath := ToPath(client, workspaceRootURI)
	absolutePath := filepath.Join(workspaceRootPath, relativePath)

	return FromPath(client, absolutePath)
}
