package uri

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/open-policy-agent/regal/internal/lsp/clients"
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
	}

	if driveLetter != "" {
		path = strings.TrimPrefix(path, driveLetter)
	}

	parts := strings.Split(filepath.ToSlash(path), uriSeparator)
	for i, part := range parts {
		parts[i] = url.QueryEscape(part)
		parts[i] = strings.ReplaceAll(parts[i], "+", "%20")
	}

	if client == clients.IdentifierVSCode {
		if driveLetter != "" {
			return "file:///" + url.QueryEscape(driveLetter) + strings.Join(parts, uriSeparator)
		}

		return "file:///" + strings.Join(parts, uriSeparator)
	}

	if driveLetter != "" {
		return "file:///" + driveLetter + strings.Join(parts, uriSeparator)
	}

	return "file:///" + strings.Join(parts, uriSeparator)
}

// ToPath converts a URI to a file path from a format for a given client.
// Some clients represent URIs differently, and so this function exists to convert
// client URIs into a standard file paths.
func ToPath(client clients.Identifier, uri string) string {
	// if it looks like uri was a file URI, then there might be encoded characters in the path
	path, hadPrefix := strings.CutPrefix(uri, "file://")
	if hadPrefix {
		// if it looks like a URI, then try and decode the path
		decodedPath, err := url.QueryUnescape(path)
		if err == nil {
			path = decodedPath
		}
	}

	// handling case for windows when the drive letter is set
	if client == clients.IdentifierVSCode &&
		drivePattern.MatchString(path) {
		path = strings.TrimPrefix(path, uriSeparator)
	}

	// Convert path to use system separators
	return filepath.FromSlash(path)
}
