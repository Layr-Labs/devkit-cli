package config

import _ "embed"

//go:embed keystores/operator1.bls.keystore.json
var operator1Keystore string

//go:embed keystores/operator2.bls.keystore.json
var operator2Keystore string

//go:embed keystores/operator3.bls.keystore.json
var operator3Keystore string

//go:embed keystores/operator4.bls.keystore.json
var operator4Keystore string

//go:embed keystores/operator5.bls.keystore.json
var operator5Keystore string

//go:embed keystores/operator1.ecdsa.keystore.json
var operator1ECDSAKeystore string

//go:embed keystores/operator2.ecdsa.keystore.json
var operator2ECDSAKeystore string

//go:embed keystores/operator3.ecdsa.keystore.json
var operator3ECDSAKeystore string

//go:embed keystores/operator4.ecdsa.keystore.json
var operator4ECDSAKeystore string

//go:embed keystores/operator5.ecdsa.keystore.json
var operator5ECDSAKeystore string

// Map of context name → content
var KeystoreEmbeds = map[string]string{
	"operator1.bls.keystore.json": operator1Keystore,
	"operator2.bls.keystore.json": operator2Keystore,
	"operator3.bls.keystore.json": operator3Keystore,
	"operator4.bls.keystore.json": operator4Keystore,
	"operator5.bls.keystore.json": operator5Keystore,
	"operator1.ecdsa.keystore.json": operator1ECDSAKeystore,
	"operator2.ecdsa.keystore.json": operator2ECDSAKeystore,
	"operator3.ecdsa.keystore.json": operator3ECDSAKeystore,
	"operator4.ecdsa.keystore.json": operator4ECDSAKeystore,
	"operator5.ecdsa.keystore.json": operator5ECDSAKeystore,
}
