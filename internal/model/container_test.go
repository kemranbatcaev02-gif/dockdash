package model

import (
	"testing"
)

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name     string
		usage    uint64
		limit    uint64
		expected string
	}{
		{
			name:     "Zero limits",
			usage:    0,
			limit:    0,
			expected: "—",
		},
		{
			name:     "MB display",
			usage:    50 * 1024 * 1024,
			limit:    512 * 1024 * 1024,
			expected: "50.0 / 512 MB",
		},
		{
			name:     "GB display",
			usage:    150 * 1024 * 1024,
			limit:    2 * 1024 * 1024 * 1024,
			expected: "150.0 MB / 2.00 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ContainerInfo{MemoryUsage: tt.usage, MemoryLimit: tt.limit}
			got := c.FormatMemory()
			if got != tt.expected {
				t.Errorf("FormatMemory() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFormatCPU(t *testing.T) {
	cRunning := ContainerInfo{Status: StatusRunning, CPUPercent: 12.34}
	if got := cRunning.FormatCPU(); got != "12.3%" {
		t.Errorf("FormatCPU() = %v, want 12.3%%", got)
	}

	cExited := ContainerInfo{Status: StatusExited, CPUPercent: 0}
	if got := cExited.FormatCPU(); got != "—" {
		t.Errorf("FormatCPU() = %v, want —", got)
	}
}

func TestFormatPorts(t *testing.T) {
	if got := FormatPorts(nil); got != "—" {
		t.Errorf("FormatPorts(nil) = %v, want —", got)
	}

	ports := []string{"80->80/tcp", "443->443/tcp"}
	if got := FormatPorts(ports); got != "80->80/tcp, 443->443/tcp" {
		t.Errorf("FormatPorts() = %v, want '80->80/tcp, 443->443/tcp'", got)
	}
}
