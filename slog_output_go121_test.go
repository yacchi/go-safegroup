//go:build go1.21

package safegroup_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/yacchi/go-safegroup"
)

func TestWaitSlogOutput(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))

	group, _ := safegroup.WithContext(context.Background())
	group.GoLabel("job-1", func(context.Context) error {
		panic("boom")
	})

	err := group.Wait()
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	logger.Error("wait failed", "error", err, "panic", safegroup.AsPanic(err))

	logLine := strings.TrimSpace(output.String())
	t.Log(logLine)

	if !strings.Contains(logLine, `"stack":"`) {
		t.Fatalf("expected stack in slog output: %s", logLine)
	}
}
