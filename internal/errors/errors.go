package errors

import (
	"errors"
	"fmt"
)

// Error codes for programmatic handling.
const (
	CodeConfigInvalid    = "CONFIG_INVALID"
	CodeAgentNotFound    = "AGENT_NOT_FOUND"
	CodeProviderError    = "PROVIDER_ERROR"
	CodeTimeout          = "TIMEOUT"
	CodeMaxIterations    = "MAX_ITERATIONS"
	CodeAPIKeyMissing    = "API_KEY_MISSING"
	CodeCyclicDependency = "CYCLIC_DEPENDENCY"
	CodeToolNotFound     = "TOOL_NOT_FOUND"
)

// CadreError is a structured error with a code and actionable suggestion.
type CadreError struct {
	Code       string // machine-readable code (e.g. CONFIG_INVALID)
	Message    string // human-readable description
	Suggestion string // actionable fix
	Err        error  // wrapped underlying error
}

// Error implements the error interface.
func (e *CadreError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Code, e.Message)
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// Unwrap supports errors.Is / errors.As.
func (e *CadreError) Unwrap() error {
	return e.Err
}

// New creates a CadreError with the given code and message.
func New(code, message string) *CadreError {
	return &CadreError{Code: code, Message: message}
}

// Wrap creates a CadreError wrapping an existing error.
func Wrap(code, message string, err error) *CadreError {
	return &CadreError{Code: code, Message: message, Err: err}
}

// WithSuggestion returns a copy with the suggestion set.
func (e *CadreError) WithSuggestion(suggestion string) *CadreError {
	e.Suggestion = suggestion
	return e
}

// Is checks whether target matches this error's code.
func (e *CadreError) Is(target error) bool {
	var ce *CadreError
	if errors.As(target, &ce) {
		return e.Code == ce.Code
	}
	return false
}

// AsCode extracts the CadreError code from an error, or "" if not a CadreError.
func AsCode(err error) string {
	var ce *CadreError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return ""
}

// Suggestion extracts the suggestion from an error, or "" if not a CadreError.
func Suggestion(err error) string {
	var ce *CadreError
	if errors.As(err, &ce) {
		return ce.Suggestion
	}
	return ""
}
