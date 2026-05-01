package config

import _ "embed"

//go:embed constraints.g.yml
var RawConstraints []byte
