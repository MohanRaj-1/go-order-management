package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	var output bytes.Buffer

	logger := New(&output, slog.LevelInfo)

	logger.Info(
		"order created",
		"order_id", "ORD-101",
		"operation", "create_order",
	)

	var record map[string]any

	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v", err)
	}

	if record["msg"] != "order created" {
		t.Errorf("expected message %q, got %v", "order created", record["msg"])
	}

	if record["level"] != "INFO" {
		t.Errorf("expected level INFO, got %v", record["level"])
	}

	if record["order_id"] != "ORD-101" {
		t.Errorf("expected order ID ORD-101, got %v", record["order_id"])
	}

	if record["operation"] != "create_order" {
		t.Errorf("expected operation create_order, got %v", record["operation"])
	}
}

func TestLoggerFiltersMessagesBelowMinimumLevel(t *testing.T) {
	var output bytes.Buffer

	logger := New(&output, slog.LevelInfo)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warning message")
	logger.Error("error message")

	logOutput := output.String()

	if strings.Contains(logOutput, "debug message") {
		t.Error("debug message should have been filtered out")
	}

	for _, message := range []string{
		"info message",
		"warning message",
		"error message",
	} {
		if !strings.Contains(logOutput, message) {
			t.Errorf("expected log output to contain %q", message)
		}
	}
}
