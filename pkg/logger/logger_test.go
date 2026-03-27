package logger

import (
	"testing"
)

func TestLoggerInit(t *testing.T) {
	Init()
	l := ErrorLog()
	if l == nil {
		t.Error("Logger is nil")
	}
	l.Info("Test log output")
}
