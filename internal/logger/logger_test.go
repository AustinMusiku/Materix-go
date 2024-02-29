package logger

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{"Debug", LevelDebug, `"DEBUG"`},
		{"Info", LevelInfo, `"INFO"`},
		{"Warn", LevelWarn, `"WARN"`},
		{"Error", LevelError, `"ERROR"`},
		{"Fatal", LevelFatal, `"FATAL"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.level)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("expected %s, but got %s", tt.expected, string(data))
			}
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{"Debug", LevelDebug, "DEBUG"},
		{"Info", LevelInfo, "INFO"},
		{"Warn", LevelWarn, "WARN"},
		{"Error", LevelError, "ERROR"},
		{"Fatal", LevelFatal, "FATAL"},
		{"Unknown", Level(100), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("expected %s, but got %s", tt.expected, tt.level.String())
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		logger := New(nil, LevelDebug)
		if logger == nil {
			t.Fatal("expected logger to be not nil")
		}

		if logger.minLevel != LevelDebug {
			t.Errorf("expected minLevel to be %d, but got %d", LevelDebug, logger.minLevel)
		}

		if logger.out != nil {
			t.Error("expected out to be nil")
		}

		if len(logger.buf) != 0 {
			t.Errorf("expected buf to be empty, but got %d", len(logger.buf))
		}

		if cap(logger.buf) != 0 {
			t.Errorf("expected buf to have capacity of 0, but got %d", cap(logger.buf))
		}
	})
}

func TestLogger_write(t *testing.T) {
	t.Parallel()

	now := time.Now().Format(time.RFC3339)

	tests := []struct {
		time     string
		minLevel Level
		level    Level
		message  string
		fields   map[string]string
		expected map[string]any
	}{
		{
			now,
			LevelDebug,
			LevelDebug,
			"message",
			map[string]string{"msg": ""},
			map[string]any{
				"time":    now,
				"level":   "DEBUG",
				"message": "message",
				"fields":  map[string]string{"msg": ""},
			},
		},
		{
			now,
			LevelInfo,
			LevelDebug,
			"message",
			map[string]string{"msg": ""},
			map[string]any{},
		},
		{
			now,
			LevelInfo,
			LevelInfo,
			"message",
			map[string]string{"msg": ""},
			map[string]any{
				"time":    now,
				"level":   "INFO",
				"message": "message",
				"fields":  map[string]string{"msg": ""},
			},
		},
		{
			now,
			LevelInfo,
			LevelWarn,
			"message",
			map[string]string{"msg": ""},
			map[string]any{
				"time":    now,
				"level":   "WARN",
				"message": "message",
				"fields":  map[string]string{"msg": ""},
			},
		},
		{
			now,
			LevelInfo,
			LevelError,
			errors.New("error").Error(),
			nil,
			map[string]any{
				"time":    now,
				"level":   "ERROR",
				"message": "error",
			},
		},
		{
			now,
			LevelInfo,
			LevelFatal,
			errors.New("error").Error(),
			nil,
			map[string]any{
				"time":    now,
				"level":   "FATAL",
				"message": "error",
			},
		},
	}

	for _, tt := range tests {
		title := tt.minLevel.String() + "/" + tt.level.String()
		t.Run(title, func(t *testing.T) {
			logger := New(os.Stdout, tt.minLevel)
			logger.write(tt.level, tt.message, tt.fields)

			var result map[string]any

			err := json.Unmarshal(logger.buf, &result)
			if err == nil && tt.level < tt.minLevel {
				t.Errorf("expected error %s, but got %v", "unexpected end of JSON input", err)
			} else if err != nil && tt.level >= tt.minLevel {
				t.Errorf("unexpected error: %v", err)
			}

			// Check if expected.time is equal to result.time
			if result["time"] != tt.expected["time"] {
				t.Errorf("expected time to be %s, but got %s", tt.expected["time"], result["time"])
			}

			// Check if expected.level is equal to result.level
			if result["level"] != tt.expected["level"] {
				t.Errorf("expected level to be %s, but got %s", tt.expected["level"].(string), result["level"].(string))
			}

			// Check if expected.message is equal to result.message
			if result["message"] != tt.expected["message"] {
				t.Errorf("expected message to be %s, but got %s", tt.expected["message"], result["message"])
			}

			// Check if expected.fields is equal to result.fields
			if fields, ok := result["fields"].(map[string]string); ok {
				for expectedFieldKey, expectedFieldValue := range tt.expected["fields"].(map[string]string) {
					if fields[expectedFieldKey] != expectedFieldValue {
						t.Errorf("expected fields[%s] to be %s, but got %s", expectedFieldKey, expectedFieldValue, fields[expectedFieldKey])
					}
				}
			}

			// Check expected.trace
			if trace, ok := result["trace"].(string); ok {
				if trace == "" {
					t.Error("expected trace to be not empty")
				}
			}
		})
	}
}

func TestLogger(t *testing.T) {
	t.Parallel()

	t.Run("Debug", func(t *testing.T) {
		logger := New(os.Stdout, LevelDebug)
		logger.Debug("message", nil)

		var result map[string]any
		err := json.Unmarshal(logger.buf, &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result["level"] != "DEBUG" {
			t.Errorf("expected level to be DEBUG, but got %s", result["level"])
		}

		if result["message"] != "message" {
			t.Errorf("expected message to be message, but got %s", result["message"])
		}

		if result["fields"] != nil {
			t.Errorf("expected fields to be nil, but got %v", result["fields"])
		}

		if result["trace"] != nil {
			t.Errorf("expected trace to be empty, but got %s", result["trace"])
		}
	})

	t.Run("Info", func(t *testing.T) {
		logger := New(os.Stdout, LevelDebug)
		logger.Info("message", nil)

		var result map[string]any
		err := json.Unmarshal(logger.buf, &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result["level"] != "INFO" {
			t.Errorf("expected level to be INFO, but got %s", result["level"])
		}

		if result["message"] != "message" {
			t.Errorf("expected message to be message, but got %s", result["message"])
		}

		if result["fields"] != nil {
			t.Errorf("expected fields to be nil, but got %v", result["fields"])
		}

		if result["trace"] != nil {
			t.Errorf("expected trace to be empty, but got %s", result["trace"])
		}
	})

	t.Run("Warn", func(t *testing.T) {
		logger := New(os.Stdout, LevelDebug)
		logger.Warn("message", nil)

		var result map[string]any
		err := json.Unmarshal(logger.buf, &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result["level"] != "WARN" {
			t.Errorf("expected level to be WARN, but got %s", result["level"])
		}

		if result["message"] != "message" {
			t.Errorf("expected message to be message, but got %s", result["message"])
		}

		if result["fields"] != nil {
			t.Errorf("expected fields to be nil, but got %v", result["fields"])
		}

		if result["trace"] != nil {
			t.Errorf("expected trace to be empty, but got %s", result["trace"])
		}
	})

	t.Run("Error", func(t *testing.T) {
		logger := New(os.Stdout, LevelDebug)
		logger.Error(errors.New("error"), nil)

		var result map[string]any
		err := json.Unmarshal(logger.buf, &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result["level"] != "ERROR" {
			t.Errorf("expected level to be ERROR, but got %s", result["level"])
		}

		if result["message"] != "error" {
			t.Errorf("expected message to be error, but got %s", result["message"])
		}

		if result["fields"] != nil {
			t.Errorf("expected fields to be nil, but got %v", result["fields"])
		}

		if result["trace"] == "" {
			t.Error("expected trace to be not empty")
		}
	})

	t.Run("Fatal", func(t *testing.T) {
		logger := New(os.Stdout, LevelDebug)
		Exit = func(code int) {}

		logger.Fatal(errors.New("error"), nil)

		var result map[string]any
		err := json.Unmarshal(logger.buf, &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result["level"] != "FATAL" {
			t.Errorf("expected level to be FATAL, but got %s", result["level"])
		}

		if result["message"] != "error" {
			t.Errorf("expected message to be error, but got %s", result["message"])
		}

		if result["fields"] != nil {
			t.Errorf("expected fields to be nil, but got %v", result["fields"])
		}

		if result["trace"] == "" {
			t.Error("expected trace to be not empty")
		}
	})

	t.Run("Write", func(t *testing.T) {
		logger := New(os.Stdout, LevelDebug)
		logger.Write([]byte("message"))

		var result map[string]any
		err := json.Unmarshal(logger.buf, &result)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if result["message"] != "message" {
			t.Errorf("expected buf to be message, but got %s", string(logger.buf))
		}

		if result["level"] != "ERROR" {
			t.Errorf("expected level to be ERROR, but got %s", result["level"])
		}

		if result["fields"] != nil {
			t.Errorf("expected fields to be nil, but got %v", result["fields"])
		}

		if result["trace"] == "" {
			t.Error("expected trace to be not empty")
		}
	})
}

func TestSetOutput(t *testing.T) {
	t.Parallel()

	t.Run("SetOutput", func(t *testing.T) {
		logger := New(nil, LevelDebug)
		logger.SetOutput(os.Stdout)

		if logger.out != os.Stdout {
			t.Errorf("expected out to be os.Stdout, but got %v", logger.out)
		}
	})
}

func TestSetMinLevel(t *testing.T) {
	t.Parallel()

	t.Run("SetMinLevel", func(t *testing.T) {
		logger := New(nil, LevelDebug)
		logger.SetMinLevel(LevelInfo)

		if logger.minLevel != LevelInfo {
			t.Errorf("expected minLevel to be %d, but got %d", LevelInfo, logger.minLevel)
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("Close", func(t *testing.T) {
		logger := New(os.Stdout, LevelDebug)
		logger.Debug(fmt.Sprintf("Logger.out is %v", logger.out), nil)
		logger.Close()

		if logger.out != nil {
			t.Error("expected out to be nil")
		}

		if len(logger.buf) != 0 {
			t.Errorf("expected buf to be empty, but got %d", len(logger.buf))
		}

		if cap(logger.buf) != 0 {
			t.Errorf("expected buf to have capacity of 0, but got %d", cap(logger.buf))
		}
	})
}
