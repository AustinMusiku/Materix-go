package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	}
	return ""
}

func (l Level) MarshalJSON() ([]byte, error) {
	return []byte(`"` + l.String() + `"`), nil
}

type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	buf      []byte
	minLevel Level
}

func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

func (l *Logger) Debug(message string, fields map[string]string) {
	l.write(LevelDebug, message, fields)
}
func (l *Logger) Info(message string, fields map[string]string) {
	l.write(LevelInfo, message, fields)
}
func (l *Logger) Warn(message string, fields map[string]string) {
	l.write(LevelWarn, message, fields)
}
func (l *Logger) Error(err error, fields map[string]string) {
	l.write(LevelError, err.Error(), fields)
}

func (l *Logger) Fatal(err error, fields map[string]string) {
	l.write(LevelInfo, err.Error(), fields)
	os.Exit(1)
}

func (l *Logger) write(level Level, message string, fields map[string]string) (int, error) {
	var err error
	if level < l.minLevel {
		return 0, nil
	}

	var info struct {
		Time    string            `json:"time"`
		Level   Level             `json:"level"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields,omitempty"`
		Trace   string            `json:"trace,omitempty"`
	}

	info.Time = time.Now().Format(time.RFC3339)
	info.Level = level
	info.Message = message
	info.Fields = fields

	if level == LevelError || level == LevelFatal {
		info.Trace = string(debug.Stack())
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf = l.buf[:0]
	l.buf, err = json.MarshalIndent(info, "", "\t") // TODO: Perform benchmarking vs json.Marshal and optimize if necessary
	if err != nil {
		l.buf = append(l.buf, fmt.Sprintf(`{"time":"%s","level":"ERROR","message":"error marshalling log message: %s"}`, time.Now().Format(time.RFC3339), err)...)
	}

	l.buf = append(l.buf, '\n')

	return l.out.Write(l.buf)
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

func (l *Logger) SetMinLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.out != os.Stdin && l.out != os.Stdout && l.out != os.Stderr {
		if closer, ok := l.out.(io.Closer); ok {
			closer.Close()
		}
	}

	l.out = nil
	l.buf = nil
}

// Implement io.Writer interface so that Logger can be used with http.Server.ErrorLog.
// Therefore, it will be used to log LevelError messages
func (l *Logger) Write(message []byte) (int, error) {
	return l.write(LevelError, string(message), nil)
}
