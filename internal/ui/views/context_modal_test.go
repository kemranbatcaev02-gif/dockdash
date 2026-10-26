package views

import (
	"strings"
	"testing"

	"dockdash/internal/model"
)

func TestContextModalNavigation(t *testing.T) {
	m := NewContextModal()
	contexts := []model.DockerContext{
		{Name: "default", Description: "Local daemon", Endpoint: "local", IsCurrent: true},
		{Name: "staging", Description: "Staging host", Endpoint: "ssh://staging", IsCurrent: false},
		{Name: "production", Description: "Prod cluster", Endpoint: "ssh://prod", IsCurrent: false},
	}

	m.Show(contexts)
	if !m.Active {
		t.Fatalf("Expected modal to be active")
	}
	if m.Cursor != 0 {
		t.Fatalf("Expected cursor on current context (0), got %d", m.Cursor)
	}

	m.MoveDown()
	if m.Cursor != 1 {
		t.Fatalf("Expected cursor to move down to 1, got %d", m.Cursor)
	}
	selected := m.GetSelected()
	if selected == nil || selected.Name != "staging" {
		t.Fatalf("Expected staging context selected, got %+v", selected)
	}

	m.MoveDown()
	m.MoveDown() // test bounds
	if m.Cursor != 2 {
		t.Fatalf("Expected cursor at 2, got %d", m.Cursor)
	}

	m.MoveUp()
	if m.Cursor != 1 {
		t.Fatalf("Expected cursor at 1 after MoveUp, got %d", m.Cursor)
	}

	out := m.Render(100, 30)
	if !strings.Contains(out, "MULTI-HOST") {
		t.Fatalf("Expected title in rendered modal, got:\n%s", out)
	}
	if !strings.Contains(out, "CURRENT") {
		t.Fatalf("Expected CURRENT badge in rendered modal")
	}

	m.Hide()
	if m.Active {
		t.Fatalf("Expected modal to be inactive")
	}
	if m.Render(100, 30) != "" {
		t.Fatalf("Render when inactive should return empty string")
	}
}
