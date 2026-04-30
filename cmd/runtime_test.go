//nolint:testpackage // Tests cover unexported daemon compatibility helpers.
package cmd

import (
	"testing"

	"review/internal/buildinfo"
	"review/internal/client"
)

func TestDaemonCompatibleRequiresMatchingBuildID(t *testing.T) {
	t.Parallel()

	current := buildinfo.Identity{Version: "0.1.0", BuildID: "current"}

	tests := []struct {
		name string
		info client.HealthInfo
		want bool
	}{
		{
			name: "same build",
			info: client.HealthInfo{OK: true, Version: "0.1.0", BuildID: "current"},
			want: true,
		},
		{
			name: "different build",
			info: client.HealthInfo{OK: true, Version: "0.1.0", BuildID: "old"},
			want: false,
		},
		{
			name: "legacy daemon missing build id",
			info: client.HealthInfo{OK: true, Version: "0.1.0"},
			want: false,
		},
		{
			name: "different version",
			info: client.HealthInfo{OK: true, Version: "0.2.0", BuildID: "current"},
			want: false,
		},
		{
			name: "unhealthy",
			info: client.HealthInfo{OK: false, Version: "0.1.0", BuildID: "current"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := daemonCompatible(tt.info, current); got != tt.want {
				t.Fatalf("daemonCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDaemonCompatibleFallsBackToVersionWhenCurrentBuildIDUnavailable(t *testing.T) {
	t.Parallel()

	current := buildinfo.Identity{Version: "0.1.0"}
	info := client.HealthInfo{OK: true, Version: "0.1.0"}

	if !daemonCompatible(info, current) {
		t.Fatal("daemonCompatible() = false, want true")
	}
}
