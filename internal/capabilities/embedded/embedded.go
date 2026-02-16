// This file is copied and modified from:
//
// https://github.com/open-policy-agent/opa/blob/main/ast/capabilities.go
//
// It is made available under the Apache 2 license, which you can view here:
//
// https://github.com/open-policy-agent/opa/blob/main/LICENSE
//
// The original license disclaimer is included below:
//
// Copyright 2021 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.
//
// This file and the included helper methods allow Enterprise OPA's
// capabilities files to be consumed as a Go package. This mirrors the way Open
// Policy Agent does thing.

// Package embedded handles embedding and access JSON files directly included in
// Regal from it's source repository
package embedded

import (
	"bytes"
	"embed"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
)

//go:embed */*.json
var FS embed.FS

// LoadCapabilitiesVersion loads a JSON serialized capabilities structure from the specific version.
func LoadCapabilitiesVersion(engine, version string) (*ast.Capabilities, error) {
	cvs, err := LoadCapabilitiesVersions(engine)
	if err != nil {
		return nil, err
	}

	for _, cv := range cvs {
		if cv != version {
			continue
		}

		cont, err := FS.ReadFile(engine + "/" + cv + ".json")
		if err != nil {
			return nil, fmt.Errorf("failed to open capabilities version '%s' for engine '%s': %w", cv, engine, err)
		}

		caps, err := ast.LoadCapabilitiesJSON(bytes.NewReader(cont))
		if err != nil {
			return nil, fmt.Errorf("failed to load capabilities version '%s' for engine '%s': %w", cv, engine, err)
		}

		return caps, nil
	}

	return nil, fmt.Errorf("(Regal embedded %s capabilities library) no capabilities version found %v", engine, version)
}

// LoadCapabilitiesVersions loads all capabilities versions.
func LoadCapabilitiesVersions(engine string) ([]string, error) {
	ents, err := FS.ReadDir(engine)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to embedded capabilities directory for engine '%s' (does the engine exist?): %w",
			engine, err,
		)
	}

	capabilitiesVersions := make([]string, 0, len(ents))
	for _, ent := range ents {
		capabilitiesVersions = append(capabilitiesVersions, strings.Replace(ent.Name(), ".json", "", 1))
	}

	return capabilitiesVersions, nil
}
