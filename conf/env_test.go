package conf

import (
	"os"
	"testing"
)

func Test_NewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg == nil {
		t.Error("NewConfig() failed")
	}

	if os.Getenv("LOG_LEVEL") == "" && cfg.LogLevel != "debug" {
		t.Error("Default LOG_LEVEL should be debug")
	}

	if os.Getenv("LOG_LEVEL") != "" && cfg.LogLevel != os.Getenv("LOG_LEVEL") {
		t.Error("LOG_LEVEL should be the same as environment variable")
	}
}
