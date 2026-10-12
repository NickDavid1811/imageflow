// Package openapi embeds the public HTTP contract served by ImageFlow.
package openapi

import _ "embed"

// Document contains the OpenAPI 3.1 contract in YAML format.
//
//go:embed openapi.yaml
var Document []byte
