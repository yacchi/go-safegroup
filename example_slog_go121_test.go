//go:build go1.21

package safegroup_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yacchi/go-safegroup"
)

func ExamplePanicError_LogValue() {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))

	logger.Error("worker failed", "panic", &safegroup.PanicError{
		Label: "job-1",
		Value: "boom",
	})
	fmt.Print(output.String())

	// Output:
	// {"level":"ERROR","msg":"worker failed","panic":{"message":"panic in \"job-1\": boom","value":"boom","label":"job-1"}}
}

func Example_slogWithPanics() {
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

	if err := group.Wait(); err != nil {
		logger.Error("wait failed", "error", err, "panic", safegroup.AsPanic(err))
	}

	fmt.Println(strings.Contains(output.String(), `"stack":"`))

	// Output:
	// true
}
