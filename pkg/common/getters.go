package common

// getFromConfigOrDefault returns the value from config if present, else the provided default.
func GetImageConfigOrDefault(configValue string) string {
	if configValue != "" {
		return configValue
	}
	return FOUNDRY_IMAGE // default
}

// GetChainArgsConfigOrDefault returns the value from config if present, else the provided default.
func GetChainArgsConfigOrDefault(configValue string) string {
	if configValue != "" {
		return configValue
	}
	return CHAIN_ARGS // default
}

// GetRpcUrlConfigOrDefault returns the value from config if present, else the provided default.
func GetRpcUrlConfigOrDefault(configValue string) string {
	if configValue != "" {
		return configValue
	}
	return RPC_URL // default
}
