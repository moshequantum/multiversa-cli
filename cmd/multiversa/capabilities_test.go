package main

import "testing"

func TestCapabilitiesDeclareProjectOSWriteContract(t *testing.T) {
	caps := currentCapabilities()
	if len(caps.Protocols) == 0 || len(caps.ProfileSchemas) == 0 {
		t.Fatal("capabilities must declare protocol and profile schema support")
	}
	want := map[string]bool{
		"project-os":          false,
		"routing.lab-group":   false,
		"vault.stdin-secrets": false,
	}
	for _, feature := range caps.Features {
		if _, ok := want[feature]; ok {
			want[feature] = true
		}
	}
	for feature, found := range want {
		if !found {
			t.Errorf("missing feature %q", feature)
		}
	}
}

func TestCapabilitiesDeclareAuditAndLocalExecutionCommands(t *testing.T) {
	caps := currentCapabilities()
	want := map[string]bool{
		"doctor": false, "status": false, "alerts": false,
		"tenant.exec": false, "tenant.secrets": false,
	}
	for _, command := range caps.Commands {
		if _, ok := want[command]; ok {
			want[command] = true
		}
	}
	for command, found := range want {
		if !found {
			t.Errorf("missing command %q", command)
		}
	}
}
