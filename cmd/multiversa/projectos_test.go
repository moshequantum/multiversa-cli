package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/manifest"
	"github.com/moshequantum/multiversa-cli/internal/profile"
)

func saveProjectOSProfile(t *testing.T, name string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	p := profile.Default()
	p.Level = profile.Enthusiast
	p.ProjectOSName = name
	if err := p.Save(); err != nil {
		t.Fatalf("save profile: %v", err)
	}
}

func TestConfiguredProjectOSNameReadsProfile(t *testing.T) {
	saveProjectOSProfile(t, "MiniUniversoOS")
	if got := configuredProjectOSName(); got != "MiniUniversoOS" {
		t.Fatalf("configured project OS = %q", got)
	}
}

func TestApplyProjectOSNameEnrichesDefaultManifest(t *testing.T) {
	saveProjectOSProfile(t, "MiniUniversoOS")
	m := manifest.Default()

	applyProjectOSName(m)

	if m.Tenant.OSName != "MiniUniversoOS" || m.Tenant.Kind != "project-os" {
		t.Fatalf("manifest project OS not enriched: %+v", m.Tenant)
	}
}

func TestApplyProjectOSNamePrefersManifestIdentity(t *testing.T) {
	saveProjectOSProfile(t, "ProfileOS")
	m := manifest.Default()
	m.Tenant.Name = "ManifestOS"

	applyProjectOSName(m)

	if m.Tenant.OSName != "ManifestOS" {
		t.Fatalf("manifest identity overwritten by profile: %+v", m.Tenant)
	}
}

func TestStatusRendersProjectOSName(t *testing.T) {
	var out bytes.Buffer
	renderStatus(&out, statusJSON{ProjectOSName: "MiniUniversoOS"})
	if !strings.Contains(out.String(), "project os  MiniUniversoOS") {
		t.Fatalf("status does not reflect project OS name:\n%s", out.String())
	}
}
