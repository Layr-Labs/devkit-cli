package template

import (
	"devkit-cli/config"
	"testing"
)

func TestGetTemplateURLs_Comprehensive(t *testing.T) {
	tests := []struct {
		name                string
		yamlContent         string
		arch, lang          string
		wantMainURL         string
		wantMainCommit      string
		wantContractsURL    string
		wantContractsCommit string
	}{
		{
			name: "with full data including contracts",
			yamlContent: `
architectures:
  task:
    languages:
      go:
        template: "https://github.com/main"
        commit: "main-commit"
    contracts:
      languages:
        solidity:
          template: "https://github.com/contracts"
          commit: "contracts-commit"
`,
			arch:                "task",
			lang:                "go",
			wantMainURL:         "https://github.com/main",
			wantMainCommit:      "main-commit",
			wantContractsURL:    "https://github.com/contracts",
			wantContractsCommit: "contracts-commit",
		},
		{
			name: "missing contracts block",
			yamlContent: `
architectures:
  task:
    languages:
      go:
        template: "https://github.com/main"
        commit: "main-commit"
`,
			arch:                "task",
			lang:                "go",
			wantMainURL:         "https://github.com/main",
			wantMainCommit:      "main-commit",
			wantContractsURL:    "",
			wantContractsCommit: "",
		},
		{
			name: "missing commit and contracts",
			yamlContent: `
architectures:
  task:
    languages:
      go:
        template: "https://github.com/main"
`,
			arch:                "task",
			lang:                "go",
			wantMainURL:         "https://github.com/main",
			wantMainCommit:      "",
			wantContractsURL:    "",
			wantContractsCommit: "",
		},
		{
			name: "nonexistent architecture",
			yamlContent: `
architectures:
  task:
    languages:
      go:
        template: "https://github.com/main"
`,
			arch:                "nonexistent",
			lang:                "go",
			wantMainURL:         "",
			wantMainCommit:      "",
			wantContractsURL:    "",
			wantContractsCommit: "",
		},
		{
			name: "nonexistent language",
			yamlContent: `
architectures:
  task:
    languages:
      go:
        template: "https://github.com/main"
        commit: "main-commit"
`,
			arch:                "task",
			lang:                "rust",
			wantMainURL:         "",
			wantMainCommit:      "",
			wantContractsURL:    "",
			wantContractsCommit: "",
		},
		{
			name: "contracts present, but missing solidity",
			yamlContent: `
architectures:
  task:
    languages:
      go:
        template: "https://github.com/main"
        commit: "main-commit"
    contracts:
      languages:
        vyper:
          template: "https://github.com/contracts-vyper"
          commit: "vyper-commit"
`,
			arch:                "task",
			lang:                "go",
			wantMainURL:         "https://github.com/main",
			wantMainCommit:      "main-commit",
			wantContractsURL:    "",
			wantContractsCommit: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Inject custom YAML content
			config.TemplatesYaml = tc.yamlContent

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			mainURL, mainCommit, contractsURL, contractsCommit :=
				GetTemplateURLs(cfg, tc.arch, tc.lang)

			if mainURL != tc.wantMainURL {
				t.Errorf("mainURL = %q, want %q", mainURL, tc.wantMainURL)
			}
			if mainCommit != tc.wantMainCommit {
				t.Errorf("mainCommit = %q, want %q", mainCommit, tc.wantMainCommit)
			}
			if contractsURL != tc.wantContractsURL {
				t.Errorf("contractsURL = %q, want %q", contractsURL, tc.wantContractsURL)
			}
			if contractsCommit != tc.wantContractsCommit {
				t.Errorf("contractsCommit = %q, want %q", contractsCommit, tc.wantContractsCommit)
			}
		})
	}
}
