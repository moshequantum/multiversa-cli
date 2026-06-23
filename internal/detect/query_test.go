package detect

import "testing"

func TestRequiredMissing(t *testing.T) {
	r := Report{Tools: []Tool{
		{Name: "git", Installed: true},
		{Name: "ssh", Installed: false},
	}}
	missing := RequiredMissing(r, []string{"git", "ssh"})
	if len(missing) != 1 || missing[0] != "ssh" {
		t.Fatalf("expected only 'ssh' missing, got %v", missing)
	}
}

func TestRequiredMissingAllPresent(t *testing.T) {
	r := Report{Tools: []Tool{
		{Name: "git", Installed: true},
		{Name: "ssh", Installed: true},
	}}
	missing := RequiredMissing(r, []string{"git", "ssh"})
	if len(missing) != 0 {
		t.Fatalf("expected no missing tools, got %v", missing)
	}
}

func TestRequiredMissingNoneRequired(t *testing.T) {
	r := Report{}
	missing := RequiredMissing(r, nil)
	if len(missing) != 0 {
		t.Fatalf("expected no missing tools for empty required list, got %v", missing)
	}
}
