package updater

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		vA   string
		vB   string
		want int
	}{
		{"1.1.1", "1.1.0", 1},
		{"1.2.0", "1.1.0", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.1.0", "1.1.0", 0},
		{"v1.1.0", "1.1.0", 0},
		{"1.1.0", "1.1.1", -1},
		{"1.0.9", "1.1.0", -1},
		{"1.1.0-beta", "1.1.0", 0},
	}

	for _, tt := range tests {
		got := CompareVersions(tt.vA, tt.vB)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d; want %d", tt.vA, tt.vB, got, tt.want)
		}
	}
}
