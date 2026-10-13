package assets

import _ "embed"

// ModelsYAML is the embedded default models manifest.
//
//go:embed models.yaml
var ModelsYAML []byte
