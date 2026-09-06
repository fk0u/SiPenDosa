package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Port == "" {
		t.Error("Port should not be empty")
	}
	if cfg.Host == "" {
		t.Error("Host should not be empty")
	}
	if cfg.DBPath == "" {
		t.Error("DBPath should not be empty")
	}
	if cfg.WASessionPath == "" {
		t.Error("WASessionPath should not be empty")
	}
}

func TestResolvePath(t *testing.T) {
	home, _ := os.UserHomeDir()

	// 1. Tilde expansion
	p1 := resolvePath("~/test/db.sqlite", "")
	expected1 := filepath.Join(home, "test/db.sqlite")
	if p1 != expected1 {
		t.Errorf("expected %s, got %s", expected1, p1)
	}

	// 2. Absolute path
	absPath := "/var/data/sipen.db"
	p2 := resolvePath(absPath, "/other/base")
	if p2 != absPath {
		t.Errorf("expected %s, got %s", absPath, p2)
	}

	// 3. Relative path with baseDir
	p3 := resolvePath("data/sipen.db", "/opt/sipen")
	expected3 := filepath.Join("/opt/sipen", "data/sipen.db")
	if p3 != expected3 {
		t.Errorf("expected %s, got %s", expected3, p3)
	}
}
