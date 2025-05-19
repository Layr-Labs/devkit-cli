package config

import _ "embed"

//go:embed config.yaml
var DefaultConfigYaml string

//go:embed templates.yaml
var TemplatesYaml string

//go:embed .gitignore
var GitIgnore string

//go:embed contexts/devnet.yaml
var devnetContextYaml string

//go:embed .env.example
var EnvExample string

// Map of context name → content
var ContextYamls = map[string]string{
	"devnet": devnetContextYaml,
}
