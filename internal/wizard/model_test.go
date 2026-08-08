package wizard

import (
	"testing"

	"github.com/moshequantum/multiversa-cli/internal/wizard/steps"
)

func TestNamingRunsAfterConsentAndBeforeStack(t *testing.T) {
	m := New()
	if len(m.steps) < 4 {
		t.Fatalf("wizard has only %d steps", len(m.steps))
	}
	if _, ok := m.steps[1].(*steps.Consent); !ok {
		t.Fatalf("step 1 = %T, want Consent", m.steps[1])
	}
	if _, ok := m.steps[2].(*steps.Naming); !ok {
		t.Fatalf("step 2 = %T, want Naming", m.steps[2])
	}
	if _, ok := m.steps[3].(*steps.Stack); !ok {
		t.Fatalf("step 3 = %T, want Stack", m.steps[3])
	}
}

func TestWizardPropagatesProjectOSNameToReviewAndInstall(t *testing.T) {
	m := New()
	var naming *steps.Naming
	var review *steps.Review
	var install *steps.Install
	for _, step := range m.steps {
		switch value := step.(type) {
		case *steps.Naming:
			naming = value
		case *steps.Review:
			review = value
		case *steps.Install:
			install = value
		}
	}
	if naming == nil || review == nil || install == nil {
		t.Fatal("wizard is missing naming, review, or install")
	}
	naming.SetName("DemoOS")

	m.propagate()

	if review.ProjectOSName != "DemoOS" {
		t.Fatalf("review project OS = %q", review.ProjectOSName)
	}
	if install.ProjectOSName != "DemoOS" {
		t.Fatalf("install project OS = %q", install.ProjectOSName)
	}
}

func TestDryRunConfiguresNamingAndInstall(t *testing.T) {
	m := newWithOptions(Options{DryRun: true})
	for _, step := range m.steps {
		switch value := step.(type) {
		case *steps.Naming:
			if !value.DryRun {
				t.Fatal("naming step did not receive dry-run mode")
			}
		case *steps.Install:
			if !value.DryRun {
				t.Fatal("install step did not receive dry-run mode")
			}
		}
	}
}
