package tui

import (
	"strings"
	"testing"
)

func TestHeader_TitleAndSubtitleAlwaysPresent(t *testing.T) {
	out := Header("multiversa lab", "consultive setup", 0, 0)
	if !strings.Contains(out, "multiversa lab") {
		t.Fatalf("title missing from header: %q", out)
	}
	if !strings.Contains(out, "consultive setup") {
		t.Fatalf("subtitle missing from header: %q", out)
	}
}

func TestHeader_OmitsCrumbWhenTotalIsZero(t *testing.T) {
	out := Header("title", "sub", 0, 0)
	if strings.Contains(out, "step ") {
		t.Errorf("step crumb should be omitted when total=0: %q", out)
	}
}

func TestHeader_IncludesCrumbWhenStepsProvided(t *testing.T) {
	out := Header("title", "sub", 2, 5)
	if !strings.Contains(out, "step 2 of 5") {
		t.Errorf("step crumb missing: %q", out)
	}
}

func TestHeader_NoHyperlinks(t *testing.T) {
	// OSC 8 hyperlink sequence is "\x1b]8;". Header output must never
	// emit it — terminals copy hyperlinks awkwardly into scrollback.
	out := Header("title", "sub", 1, 3)
	if strings.Contains(out, "\x1b]8;") {
		t.Errorf("header emitted ANSI hyperlink: %q", out)
	}
}

func TestStepCrumb_Format(t *testing.T) {
	got := stepCrumb(3, 7)
	if got != "step 3 of 7" {
		t.Errorf("stepCrumb(3,7) = %q, want %q", got, "step 3 of 7")
	}
}
