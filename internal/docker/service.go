package docker

import (
	"context"
	"io"
	"os/exec"

	"dockdash/internal/model"
)

// Интерфейс для работы с Docker (реальный клиент или демо-заглушка)
type Service interface {
	Ping(ctx context.Context) error
	ListContainers(ctx context.Context) ([]model.ContainerInfo, error)
	InspectContainer(ctx context.Context, id string) (model.ContainerDetails, error)
	DaemonInfo(ctx context.Context) (model.DaemonInfo, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) error
	RestartContainer(ctx context.Context, id string) error
	KillContainer(ctx context.Context, id string) error
	RemoveContainer(ctx context.Context, id string, force bool) error
	GetLogsStream(ctx context.Context, id string, tailLines int, since string) (io.ReadCloser, error)
	GetStats(ctx context.Context, id string) (model.StatsUpdate, error)
	GetExecCommand(ctx context.Context, id string) (*exec.Cmd, error)
	GetDiskUsage(ctx context.Context) (model.DiskUsageInfo, error)
	GetImageLayers(ctx context.Context, imageID string) ([]model.ImageLayerInfo, error)
	Prune(ctx context.Context, target model.PruneTarget) (model.PruneResult, error)
	IsDemo() bool
}
