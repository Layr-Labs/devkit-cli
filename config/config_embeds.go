package config

import _ "embed"

//go:embed templates.yaml
var TemplatesYaml string

//go:embed .gitignore
var GitIgnore string

//go:embed .env.example
var EnvExample string

//go:embed .zeus
var ZeusConfig string
