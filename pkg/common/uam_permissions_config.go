package common

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

const PermissionsTomlPath = "uam-permissions.toml"

type SetAppointee struct {
	Account   string `toml:"account"`
	Appointee string `toml:"appointee"`
	Target    string `toml:"target"`
	Selector  string `toml:"selector"`
}

type AddPendingAdmin struct {
	Account      string `toml:"account"`
	PendingAdmin string `toml:"pending_admin"`
}

type AcceptAdmin struct {
	Account string `toml:"account"`
}

type RemoveAdmin struct {
	Account string `toml:"account"`
	Admin   string `toml:"admin"`
}

type RemoveAppointee struct {
	Account   string `toml:"account"`
	Appointee string `toml:"appointee"`
	Target    string `toml:"target"`
	Selector  string `toml:"selector"`
}

type PermissionBlock struct {
	SetAppointee    []SetAppointee    `toml:"set_appointee"`
	AddPendingAdmin []AddPendingAdmin `toml:"add_pending_admin"`
	AcceptAdmin     []AcceptAdmin     `toml:"accept_admin"`
	RemoveAdmin     []RemoveAdmin     `toml:"remove_admin"`
	RemoveAppointee []RemoveAppointee `toml:"remove_appointee"`
}

type PermissionConfig struct {
	Keys        map[string]string `toml:"keys"`
	Permissions PermissionBlock   `toml:"permissions"`
}

func LoadPermissionConfig() (*PermissionConfig, error) {
	var config PermissionConfig
	if _, err := toml.DecodeFile(PermissionsTomlPath, &config); err != nil {
		return nil, fmt.Errorf("%s not found. Are you running this command from your project directory?", PermissionsTomlPath)
	}
	return &config, nil
}
