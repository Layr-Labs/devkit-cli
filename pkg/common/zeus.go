package common

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"gopkg.in/yaml.v3"
)

// L1ZeusAddressData represents the addresses returned by zeus list command
type L1ZeusAddressData struct {
	AllocationManager    string `json:"allocationManager"`
	DelegationManager    string `json:"delegationManager"`
	StrategyManager      string `json:"strategyManager"`
	CrossChainRegistry   string `json:"crossChainRegistry"`
	KeyRegistrar         string `json:"keyRegistrar"`
	ReleaseManager       string `json:"releaseManager"`
	OperatorTableUpdater string `json:"operatorTableUpdater"`
	TaskMailbox          string `json:"taskMailbox"`
	PermissionController string `json:"permissionController"`
}

type L2ZeusAddressData struct {
	OperatorTableUpdater     string `json:"operatorTableUpdater"`
	ECDSACertificateVerifier string `json:"ecdsaCertificateVerifier"`
	BN254CertificateVerifier string `json:"bn254CertificateVerifier"`
	TaskMailbox              string `json:"taskMailbox"`
}

// GetZeusAddresses runs the zeus env show commands and extracts core EigenLayer addresses.
func GetZeusAddresses(ctx context.Context, logger iface.Logger, contextName string) (*L1ZeusAddressData, *L2ZeusAddressData, error) {
	var (
		l1Raw, l2Raw []byte
		err          error
	)

	// Default zeus env to sepolia for testnet/devnet
	l1ZeusEnv := "testnet-sepolia"
	l2ZeusEnv := "testnet-base-sepolia"

	// Override with mainnet envs
	if contextName == "mainnet" {
		l1ZeusEnv = "mainnet"
		l2ZeusEnv = "base"
	}

	// Run L1
	l1Raw, err = runZeusJSON(ctx, l1ZeusEnv)
	if err != nil {
		return nil, nil, fmt.Errorf("zeus L1: %w", err)
	}

	// Run L2
	l2Raw, err = runZeusJSON(ctx, l2ZeusEnv)
	if err != nil {
		return nil, nil, fmt.Errorf("zeus L2: %w", err)
	}

	// Parse the JSON outputs
	var (
		l1ZeusData map[string]interface{}
		l2ZeusData map[string]interface{}
	)
	if err := json.Unmarshal(l1Raw, &l1ZeusData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Zeus L1 JSON: %w; json=%q", err, truncate(string(l1Raw), 400))
	}
	if err := json.Unmarshal(l2Raw, &l2ZeusData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Zeus L2 JSON: %w; json=%q", err, truncate(string(l2Raw), 400))
	}

	logger.Info("Parsing Zeus JSON output\n\n")

	// Extract addresses
	l1 := &L1ZeusAddressData{}
	l2 := &L2ZeusAddressData{}

	// L1 keys
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_AllocationManager_Proxy"].(string); ok {
		l1.AllocationManager = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_DelegationManager_Proxy"].(string); ok {
		l1.DelegationManager = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_StrategyManager_Proxy"].(string); ok {
		l1.StrategyManager = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_CrossChainRegistry_Proxy"].(string); ok {
		l1.CrossChainRegistry = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_KeyRegistrar_Proxy"].(string); ok {
		l1.KeyRegistrar = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_ReleaseManager_Proxy"].(string); ok {
		l1.ReleaseManager = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_OperatorTableUpdater_Proxy"].(string); ok {
		l1.OperatorTableUpdater = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_TaskMailbox_Proxy"].(string); ok {
		l1.TaskMailbox = v
	}
	if v, ok := l1ZeusData["ZEUS_DEPLOYED_PermissionController_Proxy"].(string); ok {
		l1.PermissionController = v
	}

	if l1.AllocationManager == "" || l1.DelegationManager == "" || l1.StrategyManager == "" ||
		l1.CrossChainRegistry == "" || l1.KeyRegistrar == "" || l1.ReleaseManager == "" || l1.OperatorTableUpdater == "" {
		logger.Warn("failed to extract required L1 addresses from zeus output")
		return nil, nil, fmt.Errorf("missing required L1 addresses")
	}

	// L2 keys
	if v, ok := l2ZeusData["ZEUS_DEPLOYED_OperatorTableUpdater_Proxy"].(string); ok {
		l2.OperatorTableUpdater = v
	}
	if v, ok := l2ZeusData["ZEUS_DEPLOYED_ECDSACertificateVerifier_Proxy"].(string); ok {
		l2.ECDSACertificateVerifier = v
	}
	if v, ok := l2ZeusData["ZEUS_DEPLOYED_BN254CertificateVerifier_Proxy"].(string); ok {
		l2.BN254CertificateVerifier = v
	}
	if v, ok := l2ZeusData["ZEUS_DEPLOYED_TaskMailbox_Proxy"].(string); ok {
		l2.TaskMailbox = v
	}

	return l1, l2, nil
}

// UpdateContextWithZeusAddresses updates the context configuration with addresses from Zeus
func UpdateContextWithZeusAddresses(context context.Context, logger iface.Logger, contextMap *yaml.Node, contextName string) error {
	logger.Title("Fetching EigenLayer core addresses for L1 and L2 from Zeus...")
	l1Addresses, l2Addresses, err := GetZeusAddresses(context, logger, contextName)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"l1": l1Addresses,
		"l2": l2Addresses,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("found addresses (marshal failed): %w", err)
	}
	logger.Info("Found addresses: %s", b)

	logger.Info("\nUpdating context with Zeus addresses...\n\n")

	// contextMap must be the mapping node for "context". Guard it.
	if contextMap == nil || contextMap.Kind != yaml.MappingNode {
		return fmt.Errorf("expected context mapping node")
	}

	// eigenlayer: {}
	eigen := ensureMapping(contextMap, "eigenlayer")
	// l1: {}
	l1Map := ensureMapping(eigen, "l1")
	// l2: {}
	l2Map := ensureMapping(eigen, "l2")

	// Prepare nodes for L1 contracts
	allocationManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "allocation_manager"}
	allocationManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.AllocationManager}
	delegationManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "delegation_manager"}
	delegationManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.DelegationManager}
	strategyManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "strategy_manager"}
	strategyManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.StrategyManager}
	crossChainRegistryKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "cross_chain_registry"}
	crossChainRegistryVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.CrossChainRegistry}
	keyRegistrarKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key_registrar"}
	keyRegistrarVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.KeyRegistrar}
	releaseManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "release_manager"}
	releaseManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.ReleaseManager}
	operatorTableUpdaterKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "operator_table_updater"}
	operatorTableUpdaterVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.OperatorTableUpdater}
	taskMailboxKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "task_mailbox"}
	taskMailboxVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.TaskMailbox}
	permissionControllerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "permission_controller"}
	permissionControllerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.PermissionController}

	// Replace existing or append new entries in l1 section
	SetMappingValue(l1Map, allocationManagerKey, allocationManagerVal)
	SetMappingValue(l1Map, delegationManagerKey, delegationManagerVal)
	SetMappingValue(l1Map, strategyManagerKey, strategyManagerVal)
	SetMappingValue(l1Map, crossChainRegistryKey, crossChainRegistryVal)
	SetMappingValue(l1Map, keyRegistrarKey, keyRegistrarVal)
	SetMappingValue(l1Map, releaseManagerKey, releaseManagerVal)
	SetMappingValue(l1Map, operatorTableUpdaterKey, operatorTableUpdaterVal)
	SetMappingValue(l1Map, taskMailboxKey, taskMailboxVal)
	SetMappingValue(l1Map, permissionControllerKey, permissionControllerVal)

	// Prepare nodes for L2 contracts
	l2OperatorTableUpdaterKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "operator_table_updater"}
	l2OperatorTableUpdaterVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.OperatorTableUpdater}
	l2ECDSACertificateVerifierKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ecdsa_certificate_verifier"}
	l2ECDSACertificateVerifierVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.ECDSACertificateVerifier}
	bn254Key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bn254_certificate_verifier"}
	bn254Val := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.BN254CertificateVerifier}
	l2TaskMailboxKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "task_mailbox"}
	l2TaskMailboxVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.TaskMailbox}

	// Replace existing or append new entries in l2 section
	SetMappingValue(l2Map, l2OperatorTableUpdaterKey, l2OperatorTableUpdaterVal)
	SetMappingValue(l2Map, l2ECDSACertificateVerifierKey, l2ECDSACertificateVerifierVal)
	SetMappingValue(l2Map, bn254Key, bn254Val)
	SetMappingValue(l2Map, l2TaskMailboxKey, l2TaskMailboxVal)

	return nil
}

func runZeusJSON(ctx context.Context, env string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "zeus", "env", "show", env, "--json")
	// Keep stdout clean. Do not merge stderr.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Reduce noise from Zeus if it supports these envs.
	// They are harmless if unknown.
	cmd.Env = append(os.Environ(),
		"NO_COLOR=1",
		"CLICOLOR=0",
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("exec failed: %w; stderr=%s", err, truncate(stderr.String(), 800))
	}

	// Clean stdout and extract the first JSON value
	clean := sanitizeCLIJSON(stdout.Bytes())
	slice, err := extractFirstTopLevelJSON(clean)
	if err != nil {
		return nil, fmt.Errorf("failed to locate JSON payload: %w; cleaned=%q", err, truncate(string(clean), 800))
	}
	return slice, nil
}

// Remove ANSI escapes and lines starting with '+' (xtrace), preserve the rest.
func sanitizeCLIJSON(b []byte) []byte {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	s := ansi.ReplaceAll(b, nil)
	var out []byte
	sc := bufio.NewScanner(bytes.NewReader(s))
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] == '+' {
			continue
		}
		out = append(out, line...)
		out = append(out, '\n')
	}
	return bytes.TrimSpace(out)
}

// Return the first complete top-level JSON object or array from text.
func extractFirstTopLevelJSON(in []byte) ([]byte, error) {
	start := bytes.IndexAny(in, "{[")
	if start < 0 {
		return nil, fmt.Errorf("no JSON start found")
	}
	open := in[start]
	close := byte('}')
	if open == '[' {
		close = ']'
	}

	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(in); i++ {
		c := in[i]
		if inStr {
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return in[start : i+1], nil
			}
		}
	}
	return nil, fmt.Errorf("unterminated JSON")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// ensureMapping finds key under parent mapping and ensures its value node is a mapping.
// It returns the value node (which will be a *yaml.Node{Kind: MappingNode}).
func ensureMapping(parent *yaml.Node, key string) *yaml.Node {
	if parent == nil || parent.Kind != yaml.MappingNode {
		panic("ensureMapping: parent must be a mapping node")
	}
	// search existing pair
	for i := 0; i+1 < len(parent.Content); i += 2 {
		k := parent.Content[i]
		v := parent.Content[i+1]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			if v.Kind != yaml.MappingNode {
				// replace with an empty mapping
				parent.Content[i+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			}
			return parent.Content[i+1]
		}
	}
	// create new key: value: {}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	parent.Content = append(parent.Content, keyNode, valNode)
	return valNode
}
