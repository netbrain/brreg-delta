package main

import (
	"testing"
)

func TestEscapeYAMLString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `Simple text`,
			expected: `Simple text`,
		},
		{
			input:    `Text with "quotes" inside`,
			expected: `Text with \"quotes\" inside`,
		},
		{
			input:    `Text with \ backslash`,
			expected: `Text with \\ backslash`,
		},
		{
			input:    `Text with both "quotes" and \ backslash`,
			expected: `Text with both \"quotes\" and \\ backslash`,
		},
	}

	for _, tt := range tests {
		result := escapeYAMLString(tt.input)
		if result != tt.expected {
			t.Errorf("escapeYAMLString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
