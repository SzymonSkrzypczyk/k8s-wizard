package app

import (
	"testing"

	"github.com/k8s-wizard/internal/kubectl"
)

// Test that when a command finishes executing, the full output that will be
// shown to the user is also stored in currentOutputContent, so saving uses the
// complete content rather than a potentially truncated viewport view.
func TestCommandExecutedStoresFullOutputForSaving(t *testing.T) {
	m := Model{}

	result := kubectl.CommandResult{
		Output: "line 1\nline 2\nline 3",
		Error:  "",
	}

	updated, _ := m.Update(commandExecutedMsg{result: result})
	model, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", updated)
	}

	expected := "Output:\n" + result.Output
	if model.currentOutputContent != expected {
		t.Fatalf("currentOutputContent mismatch.\nexpected:\n%q\ngot:\n%q", expected, model.currentOutputContent)
	}
}

