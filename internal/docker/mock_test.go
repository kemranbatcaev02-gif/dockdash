package docker

import (
	"context"
	"testing"
	"time"

	"dockdash/internal/model"
)

func TestMockClientLifecycle(t *testing.T) {
	ctx := context.Background()
	mock := NewMockClient()

	// 1. Daemon Info
	daemonInfo, err := mock.DaemonInfo(ctx)
	if err != nil {
		t.Fatalf("DaemonInfo failed: %v", err)
	}
	if daemonInfo.TotalContainers == 0 {
		t.Errorf("expected containers in daemon info, got 0")
	}

	// 2. List
	containers, err := mock.ListContainers(ctx)
	if err != nil {
		t.Fatalf("ListContainers failed: %v", err)
	}
	if len(containers) == 0 {
		t.Fatalf("expected at least 1 mock container, got 0")
	}

	target := containers[0]

	// 3. Inspect
	details, err := mock.InspectContainer(ctx, target.ID)
	if err != nil {
		t.Fatalf("InspectContainer failed: %v", err)
	}
	if details.Name != target.Name {
		t.Errorf("InspectContainer name mismatch: got %s, want %s", details.Name, target.Name)
	}
	if details.IPAddress == "" {
		t.Errorf("expected IP address in inspect details, got empty")
	}

	// 4. Stop
	if err := mock.StopContainer(ctx, target.ID); err != nil {
		t.Fatalf("StopContainer failed: %v", err)
	}

	containersAfterStop, _ := mock.ListContainers(ctx)
	for _, c := range containersAfterStop {
		if c.ID == target.ID {
			if c.Status != model.StatusExited {
				t.Errorf("expected container to be exited, got %s", c.Status)
			}
			break
		}
	}

	// 5. Start
	if err := mock.StartContainer(ctx, target.ID); err != nil {
		t.Fatalf("StartContainer failed: %v", err)
	}

	containersAfterStart, _ := mock.ListContainers(ctx)
	for _, c := range containersAfterStart {
		if c.ID == target.ID {
			if c.Status != model.StatusRunning {
				t.Errorf("expected container to be running, got %s", c.Status)
			}
			break
		}
	}

	// 6. Restart
	if err := mock.RestartContainer(ctx, target.ID); err != nil {
		t.Fatalf("RestartContainer failed: %v", err)
	}

	// 6b. Kill (SIGKILL)
	if err := mock.KillContainer(ctx, target.ID); err != nil {
		t.Fatalf("KillContainer failed: %v", err)
	}

	// 7. Stats
	stats, err := mock.GetStats(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.ContainerID != target.ID {
		t.Errorf("expected stats for %s, got %s", target.ID, stats.ContainerID)
	}

	// 8. Logs Stream
	streamCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	reader, err := mock.GetLogsStream(streamCtx, target.ID, 10)
	if err != nil {
		t.Fatalf("GetLogsStream failed: %v", err)
	}
	defer reader.Close()

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("failed to read initial logs from mock stream: %v", err)
	}

	// 9. Remove
	if err := mock.RemoveContainer(ctx, target.ID, true); err != nil {
		t.Fatalf("RemoveContainer failed: %v", err)
	}

	containersAfterRemove, _ := mock.ListContainers(ctx)
	for _, c := range containersAfterRemove {
		if c.ID == target.ID {
			t.Errorf("container %s was not deleted", target.ID)
		}
	}
}
