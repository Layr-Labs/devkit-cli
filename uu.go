package main

import (
	"log"

	"sigs.k8s.io/yaml"
)

type OperatorSpec struct {
	ECDSAKey string `json:"ecdsa_key"`
}

type ChainContextConfig struct {
	Name      string         `yaml:"name"`
	ChainID   int            `yaml:"chain_id"`
	RPCURL    string         `yaml:"rpc_url"`
	Operators []OperatorSpec `yaml:"operators"`
}

func main() {
	yamlContent := []byte(`
version: 0.0.1
context:
  name: "devnet"
  chain_id: 31337
  rpc_url: "http://localhost:8545"
  operators:
    - ecdsa_key: "0xAAA"
    - ecdsa_key: "0xBBB"
`)

	var wrapper struct {
		Version string             `yaml:"version"`
		Context ChainContextConfig `yaml:"context"`
	}

	if err := yaml.Unmarshal(yamlContent, &wrapper); err != nil {
		log.Fatalf("Unmarshal error: %v", err)
	}

	log.Printf("Loaded %d operator(s)", len(wrapper.Context.Operators))
	for i, op := range wrapper.Context.Operators {
		log.Printf("Operator[%d]: %s", i, op.ECDSAKey)
	}
}
