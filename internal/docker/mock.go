package docker

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"dockdash/internal/model"
)

// Демо-клиент с тестовыми контейнерами (работает без запущенного Docker)
type MockClient struct {
	mu         sync.Mutex
	containers []model.ContainerInfo
}

// Создание тестового набора микросервисов
func NewMockClient() *MockClient {
	now := time.Now()
	return &MockClient{
		containers: []model.ContainerInfo{
			{
				ID:          "c1a2b3d4e5f6789012345678",
				ShortID:     "c1a2b3d4e5f6",
				Name:        "api-gateway",
				Image:       "nginx:alpine",
				Status:      model.StatusRunning,
				StateText:   "Up 3 hours (healthy)",
				Ports:       "80->80/tcp, 443->443/tcp",
				Created:     now.Add(-3 * time.Hour),
				CPUPercent:     1.4,
				MemoryUsage:    38 * 1024 * 1024,
				MemoryLimit:    512 * 1024 * 1024,
				ComposeProject: "shop-stack",
				ComposeService: "gateway",
			},
			{
				ID:             "f7e8d9c0b1a2345678901234",
				ShortID:        "f7e8d9c0b1a2",
				Name:           "auth-service",
				Image:          "golang:1.24-alpine",
				Status:         model.StatusRunning,
				StateText:      "Up 2 hours",
				Ports:          "8080->8080/tcp",
				Created:        now.Add(-2 * time.Hour),
				CPUPercent:     3.8,
				MemoryUsage:    64 * 1024 * 1024,
				MemoryLimit:    1024 * 1024 * 1024,
				ComposeProject: "shop-stack",
				ComposeService: "auth",
			},
			{
				ID:             "99887766554433221100aabb",
				ShortID:        "998877665544",
				Name:           "postgres-db",
				Image:          "postgres:16-alpine",
				Status:         model.StatusRunning,
				StateText:      "Up 5 hours (healthy)",
				Ports:          "5432->5432/tcp",
				Created:        now.Add(-5 * time.Hour),
				CPUPercent:     6.2,
				MemoryUsage:    215 * 1024 * 1024,
				MemoryLimit:    2048 * 1024 * 1024,
				ComposeProject: "data-infra",
				ComposeService: "postgres",
			},
			{
				ID:             "11223344556677889900ccdd",
				ShortID:        "112233445566",
				Name:           "redis-cache",
				Image:          "redis:7-alpine",
				Status:         model.StatusRunning,
				StateText:      "Up 5 hours",
				Ports:          "6379->6379/tcp",
				Created:        now.Add(-5 * time.Hour),
				CPUPercent:     0.8,
				MemoryUsage:    48 * 1024 * 1024,
				MemoryLimit:    512 * 1024 * 1024,
				ComposeProject: "data-infra",
				ComposeService: "redis",
			},
			{
				ID:             "aabbccddeeff001122334455",
				ShortID:        "aabbccddeeff",
				Name:           "worker-queue",
				Image:          "rabbitmq:3-management",
				Status:         model.StatusExited,
				StateText:      "Exited (0) 45 mins ago",
				Ports:          "5672->5672/tcp, 15672->15672/tcp",
				Created:        now.Add(-6 * time.Hour),
				CPUPercent:     0.0,
				MemoryUsage:    0,
				MemoryLimit:    1024 * 1024 * 1024,
				ComposeProject: "data-infra",
				ComposeService: "queue",
			},
			{
				ID:                  "778899aabbccddeeff001122",
				ShortID:             "778899aabbcc",
				Name:                "payment-webhook",
				Image:               "node:20-alpine",
				Status:              model.StatusExited,
				StateText:           "Exited (137) 2 hours ago",
				Ports:               "3000->3000/tcp",
				Created:             now.Add(-7 * time.Hour),
				CPUPercent:          0.0,
				MemoryUsage:         0,
				MemoryLimit:         512 * 1024 * 1024,
				OOMKilled:           true,
				ExitCode:            137,
				ComposeProject:      "",
				ComposeService:      "",
			},
			{
				ID:                  "3344556677889900aabbccdd",
				ShortID:             "334455667788",
				Name:                "analytics-worker",
				Image:               "python:3.12-slim",
				Status:              model.StatusRunning,
				StateText:           "Up 1 hour",
				Ports:               "—",
				Created:             now.Add(-1 * time.Hour),
				CPUPercent:          84.2,
				CPUThrottledPercent: 32.5,
				MemoryUsage:         485 * 1024 * 1024,
				MemoryLimit:         512 * 1024 * 1024,
				OOMRisk:             true,
				ComposeProject:      "shop-stack",
				ComposeService:      "analytics",
			},
		},
	}
}

func (m *MockClient) Ping(ctx context.Context) error {
	return nil
}

// Тестовая информация о сервере
func (m *MockClient) DaemonInfo(ctx context.Context) (model.DaemonInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	running := 0
	stopped := 0
	for _, c := range m.containers {
		if c.Status == model.StatusRunning {
			running++
		} else {
			stopped++
		}
	}

	return model.DaemonInfo{
		ServerVersion:   "27.5.1 (Simulated)",
		OperatingSystem: "Linux 6.6.0 (Demo Engine)",
		TotalContainers: len(m.containers),
		RunningCount:    running,
		StoppedCount:    stopped,
		ImagesCount:     8,
		CPUs:            8,
		TotalMemory:     16 * 1024 * 1024 * 1024,
	}, nil
}

// Подробная информация для экрана инспекции [i]
func (m *MockClient) InspectContainer(ctx context.Context, id string) (model.ContainerDetails, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.containers {
		if c.ID == id || c.ShortID == id {
			return model.ContainerDetails{
				ID:            c.ID,
				Name:          c.Name,
				Image:         c.Image,
				Status:        string(c.Status),
				IPAddress:     "172.20.0.12",
				Gateway:       "172.20.0.1",
				MACAddress:    "02:42:ac:14:00:0c",
				Command:       "docker-entrypoint.sh run",
				CreatedAt:           c.Created.Format(time.RFC3339),
				StartedAt:           c.Created.Add(2 * time.Second).Format(time.RFC3339),
				FinishedAt:          "",
				ExitCode:            c.ExitCode,
				OOMKilled:           c.OOMKilled,
				CPUThrottledPercent: c.CPUThrottledPercent,
				OOMRisk:             c.OOMRisk,
				RestartPolicy:       "always",
				Mounts: []string{
					"/var/run/docker.sock -> /var/run/docker.sock (bind)",
					"/var/lib/data -> /app/data (volume)",
				},
				Env: []string{
					"NODE_ENV=production",
					"PORT=8080",
					"LOG_LEVEL=info",
					"DATABASE_URL=postgres://user:pass@postgres-db:5432/appdb",
				},
				Labels: map[string]string{
					"maintainer":                 "dockdash-team",
					"com.docker.compose.project": "dockdash-cluster",
					"com.docker.compose.service": c.Name,
				},
			}, nil
		}
	}

	return model.ContainerDetails{}, fmt.Errorf("контейнер не найден: %s", id)
}

func (m *MockClient) ListContainers(ctx context.Context) ([]model.ContainerInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]model.ContainerInfo, len(m.containers))
	copy(out, m.containers)
	return out, nil
}

func (m *MockClient) StartContainer(ctx context.Context, id string) error {
	time.Sleep(250 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.containers {
		if m.containers[i].ID == id || m.containers[i].ShortID == id {
			m.containers[i].Status = model.StatusRunning
			m.containers[i].StateText = "Up just now"
			m.containers[i].CPUPercent = 2.1
			m.containers[i].MemoryUsage = 50 * 1024 * 1024
			return nil
		}
	}
	return fmt.Errorf("контейнер не найден: %s", id)
}

func (m *MockClient) StopContainer(ctx context.Context, id string) error {
	time.Sleep(250 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.containers {
		if m.containers[i].ID == id || m.containers[i].ShortID == id {
			m.containers[i].Status = model.StatusExited
			m.containers[i].StateText = "Exited (0) just now"
			m.containers[i].CPUPercent = 0
			m.containers[i].MemoryUsage = 0
			return nil
		}
	}
	return fmt.Errorf("контейнер не найден: %s", id)
}

func (m *MockClient) RestartContainer(ctx context.Context, id string) error {
	time.Sleep(350 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.containers {
		if m.containers[i].ID == id || m.containers[i].ShortID == id {
			m.containers[i].Status = model.StatusRunning
			m.containers[i].StateText = "Up just now (restarted)"
			return nil
		}
	}
	return fmt.Errorf("контейнер не найден: %s", id)
}

func (m *MockClient) KillContainer(ctx context.Context, id string) error {
	time.Sleep(100 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.containers {
		if m.containers[i].ID == id || m.containers[i].ShortID == id {
			m.containers[i].Status = model.StatusExited
			m.containers[i].StateText = "Exited (137) just now (killed)"
			m.containers[i].CPUPercent = 0
			m.containers[i].MemoryUsage = 0
			return nil
		}
	}
	return fmt.Errorf("контейнер не найден: %s", id)
}

func (m *MockClient) RemoveContainer(ctx context.Context, id string, force bool) error {
	time.Sleep(200 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.containers {
		if m.containers[i].ID == id || m.containers[i].ShortID == id {
			m.containers = append(m.containers[:i], m.containers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("контейнер не найден: %s", id)
}

// Имитация живого потока логов в реальном времени
func (m *MockClient) GetLogsStream(ctx context.Context, id string, tailLines int, since string) (io.ReadCloser, error) {
	m.mu.Lock()
	name := id
	for _, c := range m.containers {
		if c.ID == id || c.ShortID == id {
			name = c.Name
			break
		}
	}
	m.mu.Unlock()

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		initLogs := []string{
			fmt.Sprintf("[init] Starting container %s...", name),
			"[init] Loading configuration from /etc/config.json",
			"[init] Environment: PRODUCTION | LogLevel: INFO",
			"[ready] Service initialized and listening for traffic",
			"[http] GET /healthz 200 OK - 2ms",
			"[http] GET /api/v1/status 200 OK - 4ms",
		}

		if since != "" {
			initLogs = append(initLogs, fmt.Sprintf("[dockdash] Logs filtered with --since=%s", since))
		}

		for _, l := range initLogs {
			_, _ = fmt.Fprintf(pw, "%s %s\n", time.Now().Format("15:04:05"), l)
		}

		ticker := time.NewTicker(1200 * time.Millisecond)
		defer ticker.Stop()

		samples := []string{
			"[worker] Heartbeat check OK, active workers: 4",
			"[cache] Hit ratio: 94.2% (1284 keys checked)",
			"[db] Query executed in 1.4ms: SELECT * FROM sessions WHERE active = true",
			"[http] POST /events 201 Created - 12ms",
			"[metrics] Flushed 42 data points to telemetry collector",
			"[http] GET /api/v1/data 200 OK - 7ms",
			"[WARN] Redis latency spike detected: 48ms",
			"[ERROR] Database connection pool warning: 90% in use",
			"[http] POST /api/checkout 500 Internal Server Error - 210ms",
			"[worker] Processing queued task #48194",
		}

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				idx := rand.Intn(len(samples))
				_, err := fmt.Fprintf(pw, "%s %s\n", t.Format("15:04:05"), samples[idx])
				if err != nil {
					return
				}
			}
		}
	}()

	return pr, nil
}

// Имитация колебания нагрузки CPU и памяти
func (m *MockClient) GetStats(ctx context.Context, id string) (model.StatsUpdate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.containers {
		if c.ID == id || c.ShortID == id {
			if c.Status != model.StatusRunning {
				return model.StatsUpdate{
					ContainerID: id,
					CPUPercent:  0,
					MemoryUsage: 0,
					MemoryLimit: c.MemoryLimit,
				}, nil
			}

			// Небольшое случайное колебание процентов
			jitterCPU := (rand.Float64() - 0.5) * 1.5
			cpu := c.CPUPercent + jitterCPU
			if cpu < 0.2 {
				cpu = 0.5
			}

			jitterMem := int64((rand.Float64() - 0.5) * 4 * 1024 * 1024)
			mem := int64(c.MemoryUsage) + jitterMem
			if mem < 10*1024*1024 {
				mem = 10 * 1024 * 1024
			}

			return model.StatsUpdate{
				ContainerID:         id,
				CPUPercent:          cpu,
				MemoryUsage:         uint64(mem),
				MemoryLimit:         c.MemoryLimit,
				CPUThrottledPercent: c.CPUThrottledPercent,
				OOMRisk:             c.OOMRisk,
			}, nil
		}
	}

	return model.StatsUpdate{}, fmt.Errorf("контейнер не найден")
}

// Возвращает интерактивную команду-симулятор для демо-режима
func (m *MockClient) GetExecCommand(ctx context.Context, id string) (*exec.Cmd, error) {
	name := id
	m.mu.Lock()
	for _, c := range m.containers {
		if c.ID == id {
			name = c.Name
			break
		}
	}
	m.mu.Unlock()

	if runtime.GOOS == "windows" {
		banner := fmt.Sprintf("echo [DockDash Demo Shell] Подключено к контейнеру '%s' & echo Это симуляция шелла в демо-режиме. Введите exit для возврата & cmd /k", name)
		return exec.Command("cmd", "/c", banner), nil
	}
	banner := fmt.Sprintf("echo '[DockDash Demo Shell] Подключено к контейнеру %s'; echo 'Это симуляция шелла в демо-режиме. Введите exit для возврата'; sh", name)
	return exec.Command("sh", "-c", banner), nil
}

func (m *MockClient) IsDemo() bool {
	return true
}
