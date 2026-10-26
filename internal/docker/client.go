package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"dockdash/internal/model"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Клиент для подключения к живому Docker
type RealClient struct {
	cli *client.Client
}

// Создание клиента с автоопределением версии API
func NewRealClient() (*RealClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента docker: %w", err)
	}
	return &RealClient{cli: cli}, nil
}

// Проверка доступности Docker
func (r *RealClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := r.cli.Ping(ctx)
	return err
}

// Статистика Docker-демона (версия, ОС, ядра, память)
func (r *RealClient) DaemonInfo(ctx context.Context) (model.DaemonInfo, error) {
	info, err := r.cli.Info(ctx)
	if err != nil {
		return model.DaemonInfo{}, err
	}

	return model.DaemonInfo{
		ServerVersion:   info.ServerVersion,
		OperatingSystem: info.OperatingSystem,
		TotalContainers: info.Containers,
		RunningCount:    info.ContainersRunning,
		PausedCount:     info.ContainersPaused,
		StoppedCount:    info.ContainersStopped,
		ImagesCount:     info.Images,
		CPUs:            info.NCPU,
		TotalMemory:     uint64(info.MemTotal),
	}, nil
}

// Получение подробной информации о контейнере (IP, тома, ENV)
func (r *RealClient) InspectContainer(ctx context.Context, id string) (model.ContainerDetails, error) {
	jsonResp, err := r.cli.ContainerInspect(ctx, id)
	if err != nil {
		return model.ContainerDetails{}, err
	}

	ip := ""
	gw := ""
	mac := ""
	if jsonResp.NetworkSettings != nil {
		ip = jsonResp.NetworkSettings.IPAddress
		gw = jsonResp.NetworkSettings.Gateway
		mac = jsonResp.NetworkSettings.MacAddress
		if ip == "" {
			for _, net := range jsonResp.NetworkSettings.Networks {
				if net.IPAddress != "" {
					ip = net.IPAddress
					gw = net.Gateway
					mac = net.MacAddress
					break
				}
			}
		}
	}

	cmdStr := ""
	var envList []string
	var labelsMap map[string]string
	if jsonResp.Config != nil {
		cmdStr = strings.Join(jsonResp.Config.Cmd, " ")
		envList = jsonResp.Config.Env
		labelsMap = jsonResp.Config.Labels
	}

	var mountsList []string
	for _, m := range jsonResp.Mounts {
		mountsList = append(mountsList, fmt.Sprintf("%s -> %s (%s)", m.Source, m.Destination, m.Type))
	}

	restartPolicy := "no"
	if jsonResp.HostConfig != nil {
		restartPolicy = string(jsonResp.HostConfig.RestartPolicy.Name)
	}

	name := strings.TrimPrefix(jsonResp.Name, "/")
	status := ""
	created := jsonResp.Created
	started := ""
	finished := ""
	exitCode := 0

	if jsonResp.State != nil {
		status = jsonResp.State.Status
		started = jsonResp.State.StartedAt
		finished = jsonResp.State.FinishedAt
		exitCode = jsonResp.State.ExitCode
	}

	return model.ContainerDetails{
		ID:            jsonResp.ID,
		Name:          name,
		Image:         jsonResp.Config.Image,
		Status:        status,
		IPAddress:     ip,
		Gateway:       gw,
		MACAddress:    mac,
		Command:       cmdStr,
		CreatedAt:     created,
		StartedAt:     started,
		FinishedAt:    finished,
		ExitCode:      exitCode,
		RestartPolicy: restartPolicy,
		Mounts:        mountsList,
		Env:           envList,
		Labels:        labelsMap,
	}, nil
}

// Список всех контейнеров
func (r *RealClient) ListContainers(ctx context.Context) ([]model.ContainerInfo, error) {
	containers, err := r.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка контейнеров: %w", err)
	}

	result := make([]model.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		var portStrs []string
		for _, p := range c.Ports {
			if p.PublicPort != 0 {
				portStrs = append(portStrs, fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type))
			} else if p.PrivatePort != 0 {
				portStrs = append(portStrs, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			}
		}

		status := model.StatusExited
		switch strings.ToLower(c.State) {
		case "running":
			status = model.StatusRunning
		case "paused":
			status = model.StatusPaused
		case "restarting":
			status = model.StatusRestarting
		case "dead":
			status = model.StatusDead
		}

		result = append(result, model.ContainerInfo{
			ID:        c.ID,
			ShortID:   shortID,
			Name:      name,
			Image:     c.Image,
			Status:    status,
			StateText: c.Status,
			Ports:     model.FormatPorts(portStrs),
			Created:   time.Unix(c.Created, 0),
		})
	}

	return result, nil
}

func (r *RealClient) StartContainer(ctx context.Context, id string) error {
	return r.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (r *RealClient) StopContainer(ctx context.Context, id string) error {
	timeout := 10
	return r.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
}

func (r *RealClient) RestartContainer(ctx context.Context, id string) error {
	timeout := 10
	return r.cli.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout})
}

func (r *RealClient) KillContainer(ctx context.Context, id string) error {
	return r.cli.ContainerKill(ctx, id, "SIGKILL")
}

func (r *RealClient) RemoveContainer(ctx context.Context, id string, force bool) error {
	return r.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

// Потоковое чтение логов контейнера (демультиплексирование stdout и stderr)
func (r *RealClient) GetLogsStream(ctx context.Context, id string, tailLines int) (io.ReadCloser, error) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       fmt.Sprintf("%d", tailLines),
		Timestamps: false,
	}

	rawLogs, err := r.cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	go func() {
		defer rawLogs.Close()
		defer pw.Close()
		_, _ = stdcopy.StdCopy(pw, pw, rawLogs)
	}()

	return pr, nil
}

// Замер нагрузки CPU и памяти
func (r *RealClient) GetStats(ctx context.Context, id string) (model.StatsUpdate, error) {
	statsResp, err := r.cli.ContainerStats(ctx, id, false)
	if err != nil {
		return model.StatsUpdate{}, err
	}
	defer statsResp.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		return model.StatsUpdate{}, err
	}

	// Расчет процента нагрузки CPU
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}

	cpuPercent := 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	// Расчет занятой памяти
	memUsage := stats.MemoryStats.Usage
	if cache, ok := stats.MemoryStats.Stats["cache"]; ok && memUsage > cache {
		memUsage -= cache
	}

	return model.StatsUpdate{
		ContainerID: id,
		CPUPercent:  cpuPercent,
		MemoryUsage: memUsage,
		MemoryLimit: stats.MemoryStats.Limit,
	}, nil
}

// Возвращает команду для интерактивного шелла в контейнере
func (r *RealClient) GetExecCommand(ctx context.Context, id string) (*exec.Cmd, error) {
	cmd := exec.Command("docker", "exec", "-it", id, "sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi")
	return cmd, nil
}

func (r *RealClient) IsDemo() bool {
	return false
}
