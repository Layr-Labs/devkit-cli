package config

import _ "embed"

//go:embed default.eigen.toml
var DefaultEigenToml string

//go:embed default.uam-permissions.toml
var DefaultUAMPermissionsToml string

//go:embed templates.yml
var TemplatesYml string
