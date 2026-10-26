package model

import (
	"fmt"
	"strings"
	"time"
)

type ContainerStatus string

const (
	StatusRunning    ContainerStatus = "running"
	StatusExited     ContainerStatus = "exited"
	StatusPaused     ContainerStatus = "paused"
	StatusRestarting ContainerStatus = "restarting"
	StatusDead       ContainerStatus = "dead"
)

// Краткая инфа о контейнере для главной таблицы
type ContainerInfo struct {
	ID          string          `json:"id"`
	ShortID     string          `json:"short_id"`
	Name        string          `json:"name"`
	Image       string          `json:"image"`
	Status      ContainerStatus `json:"status"`
	StateText   string          `json:"state_text"` // например: "Up 2 hours"
	Ports       string          `json:"ports"`
	Created     time.Time       `json:"created"`
	CPUPercent     float64         `json:"cpu_percent"`
	MemoryUsage    uint64          `json:"memory_usage"`
	MemoryLimit    uint64          `json:"memory_limit"`
	ComposeProject string          `json:"compose_project"`
	ComposeService string          `json:"compose_service"`
	IsLoading      bool            `json:"is_loading"`
	LoadingMsg     string          `json:"loading_msg"`
}

// Форматирование памяти (МБ / ГБ)
func (c ContainerInfo) FormatMemory() string {
	if c.MemoryLimit == 0 || c.MemoryUsage == 0 {
		return "—"
	}
	usageMB := float64(c.MemoryUsage) / (1024 * 1024)
	limitMB := float64(c.MemoryLimit) / (1024 * 1024)
	if limitMB > 1024 {
		return fmt.Sprintf("%.1f MB / %.2f GB", usageMB, limitMB/1024)
	}
	return fmt.Sprintf("%.1f / %.0f MB", usageMB, limitMB)
}

// Форматирование процента процессора
func (c ContainerInfo) FormatCPU() string {
	if c.Status != StatusRunning {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", c.CPUPercent)
}

// Сборка списка портов в одну строчку
func FormatPorts(ports []string) string {
	if len(ports) == 0 {
		return "—"
	}
	return strings.Join(ports, ", ")
}

// Полная инфа о контейнере для экрана инспекции [i]
type ContainerDetails struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Status        string            `json:"status"`
	IPAddress     string            `json:"ip_address"`
	Gateway       string            `json:"gateway"`
	MACAddress    string            `json:"mac_address"`
	Command       string            `json:"command"`
	CreatedAt     string            `json:"created_at"`
	StartedAt     string            `json:"started_at"`
	FinishedAt    string            `json:"finished_at"`
	ExitCode      int               `json:"exit_code"`
	RestartPolicy string            `json:"restart_policy"`
	Mounts        []string          `json:"mounts"`
	Env           []string          `json:"env"`
	Labels        map[string]string `json:"labels"`
}

// Общая статистика самого Docker
type DaemonInfo struct {
	ServerVersion   string `json:"server_version"`
	OperatingSystem string `json:"operating_system"`
	TotalContainers int    `json:"total_containers"`
	RunningCount    int    `json:"running_count"`
	PausedCount     int    `json:"paused_count"`
	StoppedCount    int    `json:"stopped_count"`
	ImagesCount     int    `json:"images_count"`
	CPUs            int    `json:"cpus"`
	TotalMemory     uint64 `json:"total_memory"`
}

type ActionType string

const (
	ActionStart   ActionType = "start"
	ActionStop    ActionType = "stop"
	ActionRestart ActionType = "restart"
	ActionRemove  ActionType = "remove"
	ActionKill    ActionType = "kill"
)

// Результат фонового действия (старт, стоп и т.д.)
type ActionResult struct {
	ContainerID   string
	ContainerName string
	Action        ActionType
	Err           error
}

// Обновление метрик CPU/RAM в реальном времени
type StatsUpdate struct {
	ContainerID string
	CPUPercent  float64
	MemoryUsage uint64
	MemoryLimit uint64
}
