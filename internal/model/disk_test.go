package model

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "Zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "Negative bytes",
			bytes:    -100,
			expected: "0 B",
		},
		{
			name:     "Bytes below 1KB",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "Kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "Megabytes",
			bytes:    100 * 1024 * 1024,
			expected: "100.0 MB",
		},
		{
			name:     "Gigabytes",
			bytes:    4 * 1024 * 1024 * 1024,
			expected: "4.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatBytes(%d) = %v, want %v", tt.bytes, got, tt.expected)
			}
		})
	}
}
