package views

import (
	"testing"

	"dockdash/internal/model"
)

func TestFilterContainers(t *testing.T) {
	fb := NewFilterBar()

	containers := []model.ContainerInfo{
		{ID: "111", ShortID: "111", Name: "postgres-db", Image: "postgres:16"},
		{ID: "222", ShortID: "222", Name: "redis-cache", Image: "redis:7"},
		{ID: "333", ShortID: "333", Name: "api-gateway", Image: "nginx:alpine"},
	}

	// 1. Empty filter returns all
	if got := fb.FilterContainers(containers); len(got) != 3 {
		t.Errorf("expected 3 containers with empty filter, got %d", len(got))
	}

	// 2. Filter by name
	fb.Input.SetValue("redis")
	filtered := fb.FilterContainers(containers)
	if len(filtered) != 1 || filtered[0].Name != "redis-cache" {
		t.Errorf("expected only redis-cache, got %v", filtered)
	}

	// 3. Filter by image
	fb.Input.SetValue("alpine")
	filtered = fb.FilterContainers(containers)
	if len(filtered) != 1 || filtered[0].Name != "api-gateway" {
		t.Errorf("expected only api-gateway, got %v", filtered)
	}

	// 4. Non-matching filter
	fb.Input.SetValue("mongodb")
	filtered = fb.FilterContainers(containers)
	if len(filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(filtered))
	}
}
