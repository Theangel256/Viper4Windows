package windows

// apo_chain_backup.go
//
// Backing store for DeviceService.AttachAPO/DetachAPO's chain
// backup/restore (see device_service.go). This is the same fix
// delivered previously in legacy-patch/apo_chain_backup.go +
// apo_attach_detach.go, ported here because THIS is the
// AttachAPO/DetachAPO that's actually reachable from the new
// architecture — the old one lived on legacy/driveManager.go's
// DriverManager, a type the new App never constructs. The legacy-patch/
// files can stay where they are as a reference/for the old build, but
// they're no longer the copy that matters.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// fxValue captures one FxProperties string value as it existed right
// before ViPER4Windows first attached to an endpoint. Present == false
// means the value key did not exist at all, so restoring it means
// deleting the key rather than writing back an empty string.
type fxValue struct {
	Value   string `json:"value"`
	Present bool   `json:"present"`
}

// fxChainBackup is everything AttachAPO is capable of touching.
type fxChainBackup struct {
	PreMix   fxValue `json:"preMix"`
	PostMix  fxValue `json:"postMix"`
	Endpoint fxValue `json:"endpoint"`
	FXName   fxValue `json:"fxName"`
}

// chainBackupStore is the on-disk shape: one entry per device ID.
type chainBackupStore struct {
	Endpoints map[string]fxChainBackup `json:"endpoints"`
}

func chainBackupPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve exe path: %w", err)
	}
	return filepath.Join(filepath.Dir(exePath), "apo_chain_backup.json"), nil
}

func loadChainBackupStore() (chainBackupStore, error) {
	store := chainBackupStore{Endpoints: map[string]fxChainBackup{}}

	path, err := chainBackupPath()
	if err != nil {
		return store, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil // no backups yet — that's fine
		}
		return store, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return store, nil
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return store, fmt.Errorf("parse %s: %w", path, err)
	}
	if store.Endpoints == nil {
		store.Endpoints = map[string]fxChainBackup{}
	}
	return store, nil
}

func saveChainBackupStore(store chainBackupStore) error {
	path, err := chainBackupPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode chain backup: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// captureCurrentFxChain reads whatever is in FxProperties right now,
// before ViPER4Windows writes anything. Missing values are recorded as
// Present:false, which is what tells restoreFxValue to delete rather
// than write back an empty string.
func captureCurrentFxChain(fxPath string) (fxChainBackup, error) {
	var backup fxChainBackup

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.QUERY_VALUE)
	if err != nil {
		// Key doesn't exist yet — a completely clean endpoint. Every
		// value is correctly "not present".
		return backup, nil
	}
	defer k.Close()

	backup.PreMix = readFxValue(k, pkeyFXPreMix)
	backup.PostMix = readFxValue(k, pkeyFXPostMix)
	backup.Endpoint = readFxValue(k, pkeyFXEndpoint)
	backup.FXName = readFxValue(k, pkeyFXName)
	return backup, nil
}

func readFxValue(k registry.Key, propKey string) fxValue {
	v, _, err := k.GetStringValue(propKey)
	if err != nil {
		return fxValue{Present: false}
	}
	return fxValue{Value: v, Present: true}
}

func restoreFxValue(k registry.Key, propKey string, v fxValue) {
	if v.Present {
		_ = k.SetStringValue(propKey, v.Value)
		return
	}
	// Wasn't there originally — DeleteValue on an absent value is a
	// harmless no-op.
	_ = k.DeleteValue(propKey)
}

// legacyDeleteFxValues is the old DetachAPO body: unconditional
// delete, no restore. Used only when there's genuinely no backup to
// restore from (installs from before this fix existed).
func legacyDeleteFxValues(fxPath string) error {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.SET_VALUE)
	if err != nil {
		return nil // already gone
	}
	defer k.Close()

	_ = k.DeleteValue(pkeyFXPreMix)
	_ = k.DeleteValue(pkeyFXPostMix)
	_ = k.DeleteValue(pkeyFXEndpoint)
	return nil
}
