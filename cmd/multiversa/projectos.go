package main

import (
	"strings"

	"github.com/moshequantum/multiversa-cli/internal/manifest"
	"github.com/moshequantum/multiversa-cli/internal/profile"
)

func configuredProjectOSName() string {
	p, err := profile.Load()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(p.ProjectOSName)
}

func applyProjectOSName(m *manifest.Manifest) {
	if m == nil || m.Tenant.OSName != "" {
		return
	}
	if m.Tenant.Name != "" {
		m.Tenant.OSName = m.Tenant.Name
	} else {
		m.Tenant.OSName = configuredProjectOSName()
	}
	if m.Tenant.OSName != "" && m.Tenant.Kind == "" {
		m.Tenant.Kind = "project-os"
	}
}
