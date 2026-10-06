package dashboardui

import (
	"runtime/debug"
	"testing"
)

func TestOwnModulePathMatchesConstant(t *testing.T) {
	if got := ownModulePath(); got != modulePath {
		t.Errorf("ownModulePath() = %q; want %q (package at module root)", got, modulePath)
	}
}

func TestResolveModuleVersion(t *testing.T) {
	tests := []struct {
		name   string
		info   *debug.BuildInfo
		module string
		want   string
	}{
		{
			name:   "nil build info yields empty",
			info:   nil,
			module: modulePath,
			want:   "",
		},
		{
			name: "main module reports its own version",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "(devel)"},
			},
			module: modulePath,
			want:   "(devel)",
		},
		{
			name: "dependency version when linked into a consumer binary",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/consumer", Version: "(devel)"},
				Deps: []*debug.Module{{Path: modulePath, Version: "v4.13.1"}},
			},
			module: modulePath,
			want:   "v4.13.1",
		},
		{
			name: "versioned replace wins over stale dependency version",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/consumer", Version: "(devel)"},
				Deps: []*debug.Module{
					{
						Path:    modulePath,
						Version: "v4.13.1",
						Replace: &debug.Module{Path: modulePath, Version: "v4.99.0"},
					},
				},
			},
			module: modulePath,
			want:   "v4.99.0",
		},
		{
			name: "local directory replace yields empty",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/consumer", Version: "(devel)"},
				Deps: []*debug.Module{
					{Path: modulePath, Version: "v4.13.1", Replace: &debug.Module{Path: "/src/dashboardui"}},
				},
			},
			module: modulePath,
			want:   "",
		},
		{
			name: "module absent from deps yields empty",
			info: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/consumer", Version: "(devel)"},
				Deps: []*debug.Module{{Path: "example.com/other", Version: "v1.0.0"}},
			},
			module: modulePath,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveModuleVersion(tt.info, tt.module); got != tt.want {
				t.Errorf("resolveModuleVersion() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestResolveVCS(t *testing.T) {
	settings := []debug.BuildSetting{
		{Key: "-tags", Value: "production"},
		{Key: "vcs.revision", Value: "1392410d3a5e"},
		{Key: "vcs.time", Value: "2026-10-07T00:00:00Z"},
		{Key: "vcs.modified", Value: "true"},
		{Key: "GOARCH", Value: "amd64"},
	}

	stamp := resolveVCS(settings)

	if stamp.Revision != "1392410d3a5e" {
		t.Errorf("Revision = %q; want 1392410d3a5e", stamp.Revision)
	}

	if stamp.Time != "2026-10-07T00:00:00Z" {
		t.Errorf("Time = %q; want 2026-10-07T00:00:00Z", stamp.Time)
	}

	if stamp.Modified != "true" {
		t.Errorf("Modified = %q; want true", stamp.Modified)
	}

	empty := resolveVCS(nil)
	if empty != (vcsBuildInfo{}) {
		t.Errorf("resolveVCS(nil) = %+v; want zero value", empty)
	}
}
