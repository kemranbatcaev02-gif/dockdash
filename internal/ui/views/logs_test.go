package views

import (
	"fmt"
	"strings"
	"testing"

	"dockdash/internal/model"
)

func TestLogsSearchAndNavigation(t *testing.T) {
	v := NewLogsView()
	v.SetContainer(model.ContainerInfo{ID: "c1", Name: "api-service", Image: "golang:1.24"})

	sampleLines := []string{
		"10:00:00 [INFO] Server started on port 8080",
		"10:00:01 [DEBUG] Initializing database pool",
		"10:00:02 [ERROR] Failed to connect to redis: connection refused",
		"10:00:03 [INFO] Retrying connection...",
		"10:00:04 [ERROR] Redis timeout occurred",
	}
	v.AppendLines(sampleLines)

	// Search for "ERROR"
	v.ApplySearch("ERROR")
	if len(v.MatchLines) != 2 {
		t.Fatalf("Expected 2 matches for 'ERROR', got %d", len(v.MatchLines))
	}
	if v.MatchLines[0] != 2 || v.MatchLines[1] != 4 {
		t.Fatalf("Expected matches on lines 2 and 4, got %v", v.MatchLines)
	}

	// Verify NextMatch and PrevMatch
	v.NextMatch()
	if v.CurrentMatch != 1 {
		t.Fatalf("Expected CurrentMatch 1 after NextMatch, got %d", v.CurrentMatch)
	}
	v.NextMatch()
	if v.CurrentMatch != 0 {
		t.Fatalf("Expected wrap-around to 0, got %d", v.CurrentMatch)
	}
	v.PrevMatch()
	if v.CurrentMatch != 1 {
		t.Fatalf("Expected PrevMatch to wrap to 1, got %d", v.CurrentMatch)
	}

	// Search regex pattern
	v.ApplySearch("port\\s+\\d+")
	if len(v.MatchLines) != 1 || v.MatchLines[0] != 0 {
		t.Fatalf("Expected regex match on line 0, got %v", v.MatchLines)
	}

	// Clear search
	v.ClearSearch()
	if v.SearchQuery != "" || len(v.MatchLines) != 0 {
		t.Fatalf("Expected search to be cleared, got query '%s', matches %d", v.SearchQuery, len(v.MatchLines))
	}
}

func TestLogsErrorIsolation(t *testing.T) {
	v := NewLogsView()
	v.SetContainer(model.ContainerInfo{ID: "c2", Name: "worker", Image: "python:3.12"})

	sampleLines := []string{
		"12:00:00 [INFO] Worker spawned",
		"12:00:01 [DEBUG] Checking message queue",
		"12:00:02 [WARN] High memory consumption detected",
		"12:00:03 [INFO] Processing batch #1",
		"12:00:04 [FATAL] Unhandled panic in pipeline",
	}
	v.AppendLines(sampleLines)

	// Enable error isolation
	v.ToggleErrorIsolation()
	if !v.ErrorIsolation {
		t.Fatalf("Expected ErrorIsolation to be true")
	}

	content := v.Viewport.View()
	if strings.Contains(content, "Worker spawned") {
		t.Fatalf("Non-error lines should be filtered out in error isolation mode")
	}
	if !strings.Contains(content, "High memory consumption detected") {
		t.Fatalf("Warning lines should be retained in error isolation mode")
	}
	if !strings.Contains(content, "Unhandled panic in pipeline") {
		t.Fatalf("Fatal lines should be retained in error isolation mode")
	}

	// Disable error isolation
	v.ToggleErrorIsolation()
	contentAfter := v.Viewport.View()
	if !strings.Contains(contentAfter, "Worker spawned") {
		t.Fatalf("Normal lines should return after disabling error isolation")
	}
}

func TestLogsSinceFilterCycle(t *testing.T) {
	v := NewLogsView()

	if v.SinceLabel() != "All" {
		t.Fatalf("Expected initial since to be All, got %s", v.SinceLabel())
	}

	s1 := v.CycleSince()
	if s1 != "15m" || v.SinceLabel() != "15m" {
		t.Fatalf("Expected 15m, got %s (%s)", s1, v.SinceLabel())
	}

	s2 := v.CycleSince()
	if s2 != "1h" || v.SinceLabel() != "1h" {
		t.Fatalf("Expected 1h, got %s (%s)", s2, v.SinceLabel())
	}

	s3 := v.CycleSince()
	if s3 != "24h" || v.SinceLabel() != "24h" {
		t.Fatalf("Expected 24h, got %s (%s)", s3, v.SinceLabel())
	}

	s4 := v.CycleSince()
	if s4 != "" || v.SinceLabel() != "All" {
		t.Fatalf("Expected All, got %s (%s)", s4, v.SinceLabel())
	}
}

func TestLogsBufferMaxLines(t *testing.T) {
	v := NewLogsView()

	var bulkLines []string
	for i := 0; i < 6000; i++ {
		bulkLines = append(bulkLines, fmt.Sprintf("Line %d", i))
	}
	v.AppendLines(bulkLines)

	if len(v.RawLines) != 5000 {
		t.Fatalf("Expected buffer to be capped at 5000 lines, got %d", len(v.RawLines))
	}
	if v.RawLines[0] != "Line 1000" {
		t.Fatalf("Expected oldest line to be Line 1000, got %s", v.RawLines[0])
	}
}
