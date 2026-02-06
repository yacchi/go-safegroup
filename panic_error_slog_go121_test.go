//go:build go1.21

package safegroup

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestPanicErrorSlogIncludesStack(t *testing.T) {
	panicErr := &PanicError{
		Label: "job-1",
		Value: "boom",
		Stack: captureStack(0),
	}

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	logger.Error("worker failed", "panic", panicErr)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse slog json: %v", err)
	}

	panicField, ok := entry["panic"].(map[string]any)
	if !ok {
		t.Fatalf("expected panic field object, got %T", entry["panic"])
	}
	if got, ok := panicField["label"].(string); !ok || got != "job-1" {
		t.Fatalf("unexpected label: %#v", panicField["label"])
	}
	if got, ok := panicField["message"].(string); !ok || got == "" {
		t.Fatalf("expected non-empty message, got %#v", panicField["message"])
	}
	if got, ok := panicField["stack"].(string); !ok || got == "" {
		t.Fatalf("expected non-empty stack, got %#v", panicField["stack"])
	}
}

func TestPanicErrorSlogOmitsStackWhenEmpty(t *testing.T) {
	panicErr := &PanicError{
		Label: "job-1",
		Value: "boom",
	}

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	logger.Error("worker failed", "panic", panicErr)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse slog json: %v", err)
	}

	panicField, ok := entry["panic"].(map[string]any)
	if !ok {
		t.Fatalf("expected panic field object, got %T", entry["panic"])
	}
	if _, exists := panicField["stack"]; exists {
		t.Fatalf("expected stack to be omitted, got %#v", panicField["stack"])
	}
}
