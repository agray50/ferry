package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type TargetsFile map[string]Target

type Target struct {
	Arch           string `json:"arch"`
	Libc           string `json:"libc"`
	Profile        string `json:"profile"`
	BundleHash     string `json:"bundle_hash"`
	BootstrappedAt string `json:"bootstrapped_at"`
	LastUpdated    string `json:"last_updated"`
}

// ReadTargets reads ~/.ferry/targets.json.
// Returns empty TargetsFile if file does not exist.
func ReadTargets() (TargetsFile, error) {
	path := TargetsFilePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(TargetsFile), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading targets.json: %w", err)
	}
	var tf TargetsFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parsing targets.json: %w", err)
	}
	return tf, nil
}

// WriteTargets writes TargetsFile to ~/.ferry/targets.json.
func WriteTargets(tf TargetsFile) error {
	if err := EnsureFerryDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding targets.json: %w", err)
	}
	path := TargetsFilePath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing targets.json: %w", err)
	}
	return nil
}

// UpsertTarget adds or updates a single target entry.
func UpsertTarget(host string, t Target) error {
	tf, err := ReadTargets()
	if err != nil {
		return err
	}
	tf[host] = t
	return WriteTargets(tf)
}

// RemoveTarget removes a target entry. No-op if not present.
func RemoveTarget(host string) error {
	tf, err := ReadTargets()
	if err != nil {
		return err
	}
	delete(tf, host)
	return WriteTargets(tf)
}

// GetTarget returns a target by host string.
func GetTarget(host string) (Target, error) {
	tf, err := ReadTargets()
	if err != nil {
		return Target{}, err
	}
	t, ok := tf[host]
	if !ok {
		return Target{}, fmt.Errorf("target %q not found — run: ferry bootstrap %s", host, host)
	}
	return t, nil
}
