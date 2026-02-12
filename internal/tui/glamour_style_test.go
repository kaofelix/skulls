package tui

import "testing"

func TestGlamourStyleFromEnv_DefaultsToTerminalANSI(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "")
	if got := glamourStyleFromEnv(); got != terminalANSIStyleName {
		t.Fatalf("style=%q, want %q", got, terminalANSIStyleName)
	}
}

func TestGlamourStyleFromEnv_AutoFallsBackToTerminalANSI(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "auto")
	if got := glamourStyleFromEnv(); got != terminalANSIStyleName {
		t.Fatalf("style=%q, want %q", got, terminalANSIStyleName)
	}
}

func TestGlamourStyleFromEnv_RespectsExplicitStyle(t *testing.T) {
	t.Setenv("GLAMOUR_STYLE", "light")
	if got := glamourStyleFromEnv(); got != "light" {
		t.Fatalf("style=%q, want %q", got, "light")
	}
}
