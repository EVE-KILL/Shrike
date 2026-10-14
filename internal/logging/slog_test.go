package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

func TestSlogBridgePreservesGroupsAndErrors(t *testing.T) {
	var output bytes.Buffer
	logger := Slog(zerolog.New(&output)).With("subsystem", "river").WithGroup("job").With("attempt", 2)
	logger.ErrorContext(context.Background(), "job failed", "error", errors.New("fixture error"))
	var data map[string]any
	if err := json.Unmarshal(output.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if data["level"] != "error" || data["subsystem"] != "river" || data["job.attempt"] != float64(2) || data["job.error"] != "fixture error" {
		t.Fatal(data)
	}
}
