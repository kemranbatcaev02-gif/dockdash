package views

import (
	"strings"
	"testing"

	"dockdash/internal/model"
)

func TestNoRowWrapping(t *testing.T) {
	containers := []model.ContainerInfo{
		{ID: "1", Name: "api-gateway", Image: "nginx:alpine", Status: model.StatusRunning, Ports: "80->80/tcp, 443->443/tcp", CPUPercent: 1.2, MemoryUsage: 38 * 1024 * 1024, MemoryLimit: 512 * 1024 * 1024, StateText: "Up 3 hours (healthy)"},
		{ID: "2", Name: "postgres-db", Image: "postgres:16-alpine", Status: model.StatusRunning, Ports: "5432->5432/tcp", CPUPercent: 6.2, MemoryUsage: 215 * 1024 * 1024, MemoryLimit: 2048 * 1024 * 1024, StateText: "Up 5 hours (healthy)"},
		{ID: "3", Name: "redis-cache", Image: "redis:7-alpine", Status: model.StatusRunning, Ports: "6379->6379/tcp", CPUPercent: 0.8, MemoryUsage: 48 * 1024 * 1024, MemoryLimit: 512 * 1024 * 1024, StateText: "Up 5 hours"},
		{ID: "4", Name: "worker-queue", Image: "rabbitmq:3-management", Status: model.StatusExited, Ports: "5672->5672/tcp, 15672->15672/tcp", StateText: "Exited (0) 45 mins ago"},
	}

	for w := 50; w <= 220; w += 5 {
		v := NewContainersListView()
		v.Width = w
		v.Height = 10

		for cur := 0; cur < len(containers); cur++ {
			v.Cursor = cur
			out := v.Render(containers, true)

			lines := strings.Split(out, "\n")
			// When Height=10, with borders and header it must NEVER exceed Height+2=12 lines
			if len(lines) > 12 {
				t.Fatalf("Wrapping detected at width=%d, cursor=%d! Got %d lines, expected <= 12", w, cur, len(lines))
			}
		}
	}
}
