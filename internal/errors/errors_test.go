package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestCadreError_Error(t *testing.T) {
	err := New(CodeConfigInvalid, "missing agent name")
	expected := "[CONFIG_INVALID] missing agent name"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestCadreError_Wrap(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	err := Wrap(CodeProviderError, "API call failed", inner)

	if err.Error() != "[PROVIDER_ERROR] API call failed: connection refused" {
		t.Errorf("unexpected error string: %s", err.Error())
	}

	// Unwrap should return inner
	if !errors.Is(err, inner) {
		t.Error("errors.Is should find inner error")
	}
}

func TestCadreError_WithSuggestion(t *testing.T) {
	err := New(CodeAPIKeyMissing, "ANTHROPIC_API_KEY not set").
		WithSuggestion("Set the ANTHROPIC_API_KEY environment variable or add api_key to cadre.yaml")

	if err.Suggestion != "Set the ANTHROPIC_API_KEY environment variable or add api_key to cadre.yaml" {
		t.Errorf("unexpected suggestion: %s", err.Suggestion)
	}
}

func TestCadreError_ErrorsAs(t *testing.T) {
	err := Wrap(CodeTimeout, "task timed out", fmt.Errorf("deadline exceeded"))

	var cadreErr *CadreError
	if !errors.As(err, &cadreErr) {
		t.Fatal("errors.As should work")
	}
	if cadreErr.Code != CodeTimeout {
		t.Errorf("expected code %q, got %q", CodeTimeout, cadreErr.Code)
	}
}

func TestAsCode(t *testing.T) {
	err := New(CodeMaxIterations, "agent hit iteration limit")
	if AsCode(err) != CodeMaxIterations {
		t.Errorf("expected code %q, got %q", CodeMaxIterations, AsCode(err))
	}

	// Non-CadreError
	plain := fmt.Errorf("plain error")
	if AsCode(plain) != "" {
		t.Error("expected empty code for non-CadreError")
	}
}

func TestSuggestion(t *testing.T) {
	err := New(CodeToolNotFound, "tool not found").WithSuggestion("check tool name")
	if Suggestion(err) != "check tool name" {
		t.Errorf("expected 'check tool name', got %q", Suggestion(err))
	}

	// Non-CadreError
	if Suggestion(fmt.Errorf("plain")) != "" {
		t.Error("expected empty suggestion for non-CadreError")
	}
}

func TestCadreError_WrappedAs(t *testing.T) {
	inner := New(CodeProviderError, "API error")
	wrapped := fmt.Errorf("runtime failed: %w", inner)

	var cadreErr *CadreError
	if !errors.As(wrapped, &cadreErr) {
		t.Fatal("errors.As should unwrap through fmt.Errorf")
	}
	if cadreErr.Code != CodeProviderError {
		t.Errorf("expected code %q, got %q", CodeProviderError, cadreErr.Code)
	}
}
