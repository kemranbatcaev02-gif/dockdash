package model

import (
	"fmt"
	"time"
)

// Информация о слое Docker-образа (для анализа раздутия dive-style)
type ImageLayerInfo struct {
	ID        string    `json:"id"`
	CreatedBy string    `json:"created_by"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	IsBloated bool      `json:"is_bloated"` // Слой больше 50MB
}

// Форматирование размера в байтах в человекочитаемый вид (KB, MB, GB)
func FormatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Элемент списка образов
type ImageItem struct {
	ID         string           `json:"id"`
	ShortID    string           `json:"short_id"`
	Repository string           `json:"repository"`
	Tag        string           `json:"tag"`
	Size       int64            `json:"size"`
	Created    time.Time        `json:"created"`
	Layers     []ImageLayerInfo `json:"layers,omitempty"`
}

// Полная статистика использования диска Docker (docker system df)
type DiskUsageInfo struct {
	ImagesTotalSize       int64       `json:"images_total_size"`
	ImagesReclaimable     int64       `json:"images_reclaimable"`
	ImagesCount           int         `json:"images_count"`
	ContainersTotalSize   int64       `json:"containers_total_size"`
	ContainersReclaimable int64       `json:"containers_reclaimable"`
	ContainersCount       int         `json:"containers_count"`
	VolumesTotalSize      int64       `json:"volumes_total_size"`
	VolumesReclaimable    int64       `json:"volumes_reclaimable"`
	VolumesCount          int         `json:"volumes_count"`
	BuildCacheTotalSize   int64       `json:"build_cache_total_size"`
	BuildCacheReclaimable int64       `json:"build_cache_reclaimable"`
	TotalReclaimable      int64       `json:"total_reclaimable"`
	TotalSize             int64       `json:"total_size"`
	ImagesList            []ImageItem `json:"images_list"`
}

// Цели очистки диска (Prune)
type PruneTarget string

const (
	PruneImages     PruneTarget = "images"
	PruneContainers PruneTarget = "containers"
	PruneVolumes    PruneTarget = "volumes"
	PruneBuildCache PruneTarget = "buildcache"
	PruneAll        PruneTarget = "all"
)

// Результат выполнения операции очистки
type PruneResult struct {
	Target       PruneTarget
	SpaceCleaned int64
	Err          error
}
