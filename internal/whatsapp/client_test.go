package whatsapp

import (
	"testing"
)

func TestFormatJID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		isErr    bool
	}{
		{
			input:    "081234567890",
			expected: "6281234567890@s.whatsapp.net",
			isErr:    false,
		},
		{
			input:    "+62 812-3456-7890",
			expected: "6281234567890@s.whatsapp.net",
			isErr:    false,
		},
		{
			input:    "628987654321",
			expected: "628987654321@s.whatsapp.net",
			isErr:    false,
		},
		{
			input:    "120363023456789012@g.us",
			expected: "120363023456789012@g.us",
			isErr:    false,
		},
		{
			input: "123",
			isErr: true,
		},
	}

	for _, tt := range tests {
		jid, err := FormatJID(tt.input)
		if tt.isErr {
			if err == nil {
				t.Errorf("FormatJID(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("FormatJID(%q) unexpected error: %v", tt.input, err)
			} else if jid.String() != tt.expected {
				t.Errorf("FormatJID(%q) = %q, expected %q", tt.input, jid.String(), tt.expected)
			}
		}
	}
}
