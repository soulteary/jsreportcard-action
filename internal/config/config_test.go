package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidConfig(t *testing.T) {
	for _, data := range []string{`null`, `[]`, `{"mode":"full"}`, `{"config_mode":"automatic"}`, `{"unknown":true}`, `{} {}`, `{"complexity_limit":0}`, `{"include":[]}`} {
		p := filepath.Join(t.TempDir(), "config.json")
		os.WriteFile(p, []byte(data), 0644)
		if _, err := Load(p); err == nil {
			t.Errorf("accepted %s", data)
		}
	}
}
