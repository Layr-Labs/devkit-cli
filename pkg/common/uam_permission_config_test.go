package common

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func TestLoadPermissionsConfig_FromCopiedTempFile(t *testing.T) {
	// Setup temp file
	tempDir := t.TempDir()
	tempTomlPath := filepath.Join(tempDir, PermissionsTomlPath)

	srcPath := filepath.Join("..", "..", "config", "default.uam-permissions.toml")
	src, err := os.Open(srcPath)
	assert.NoError(t, err)
	defer func() {
		err := src.Close()
		assert.NoError(t, err)
	}()

	dest, err := os.Create(tempTomlPath)
	assert.NoError(t, err)
	defer func() {
		err := dest.Close()
		assert.NoError(t, err)
	}()

	_, err = io.Copy(dest, src)
	assert.NoError(t, err)
	err = dest.Sync()
	assert.NoError(t, err)

	// Load config
	cfg, err := LoadPermissionConfigFromPath(tempTomlPath)
	assert.NoError(t, err)

	// Keys
	assert.Equal(t, "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", cfg.Keys["account1"])
	assert.Equal(t, "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", cfg.Keys["pendingadmin"])

	// Set Appointee
	assert.Equal(t, len(cfg.Permissions.SetAppointee), 2)
	assert.Equal(t, "account1", cfg.Permissions.SetAppointee[0].Account)
	assert.Equal(t, "0xappointeeaddress", cfg.Permissions.SetAppointee[0].Appointee)
	assert.Equal(t, "0xtarget...", cfg.Permissions.SetAppointee[0].Target)
	assert.Equal(t, "0x12345678", cfg.Permissions.SetAppointee[0].Selector)

	assert.Equal(t, "0xdeadbeef", cfg.Permissions.SetAppointee[1].Selector)

	// ➕ Add Pending Admin
	assert.Equal(t, len(cfg.Permissions.AddPendingAdmin), 1)
	assert.Equal(t, "account1", cfg.Permissions.AddPendingAdmin[0].Account)
	assert.Equal(t, "0xpendingadminaddress", cfg.Permissions.AddPendingAdmin[0].PendingAdmin)

	// Accept Admin
	assert.Equal(t, len(cfg.Permissions.AcceptAdmin), 1)
	assert.Equal(t, "pendingadmin", cfg.Permissions.AcceptAdmin[0].Account)

	// Remove Admin
	assert.Equal(t, len(cfg.Permissions.RemoveAdmin), 1)
	assert.Equal(t, "account1", cfg.Permissions.RemoveAdmin[0].Account)
	assert.Equal(t, "0xpendingadminaddress", cfg.Permissions.RemoveAdmin[0].Admin)

	// Remove Appointee
	assert.Equal(t, len(cfg.Permissions.RemoveAppointee), 1)
	assert.Equal(t, "account1", cfg.Permissions.RemoveAppointee[0].Account)
	assert.Equal(t, "0xappointeeaddress", cfg.Permissions.RemoveAppointee[0].Appointee)
	assert.Equal(t, "0xtarget...", cfg.Permissions.RemoveAppointee[0].Target)
	assert.Equal(t, "0xdeadbeef", cfg.Permissions.RemoveAppointee[0].Selector)
}

func LoadPermissionConfigFromPath(path string) (*PermissionConfig, error) {
	var config PermissionConfig
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
