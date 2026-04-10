package config

import (
	"os"
	"path/filepath"
)

func FerryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ferry")
}

func KeyFile() string {
	return filepath.Join(FerryDir(), "key.txt")
}

func PubKeyFile() string {
	return filepath.Join(FerryDir(), "key.pub")
}

func TargetsFilePath() string {
	return filepath.Join(FerryDir(), "targets.json")
}

func StoreDir() string {
	return filepath.Join(FerryDir(), "store")
}

func BundlesDir() string {
	return filepath.Join(FerryDir(), "bundles")
}

func LockFilePath() string {
	return "ferry.lock"
}
