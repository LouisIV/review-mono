package buildinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sync"
)

const Version = "0.1.0"

var (
	identityOnce sync.Once
	identity     Identity
)

type Identity struct {
	Version string `json:"version"`
	BuildID string `json:"build_id,omitempty"`
}

func Current() Identity {
	identityOnce.Do(func() {
		identity = Identity{Version: Version, BuildID: executableHash()}
	})

	return identity
}

func executableHash() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}

	//nolint:gosec // The path comes from os.Executable and identifies the current process binary.
	data, err := os.ReadFile(exe)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}
