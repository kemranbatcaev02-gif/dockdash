package views

import (
	"strings"
	"testing"
	"time"

	"dockdash/internal/model"
)

func TestDiskViewRenderingAndLayers(t *testing.T) {
	v := NewDiskView()
	v.Width = 120
	v.Height = 25

	now := time.Now()
	info := model.DiskUsageInfo{
		ImagesTotalSize:       1024 * 1024 * 1024,
		ImagesReclaimable:     256 * 1024 * 1024,
		ImagesCount:           2,
		ContainersTotalSize:   100 * 1024 * 1024,
		ContainersReclaimable: 50 * 1024 * 1024,
		ContainersCount:       3,
		VolumesTotalSize:      500 * 1024 * 1024,
		VolumesReclaimable:    200 * 1024 * 1024,
		VolumesCount:          2,
		BuildCacheTotalSize:   2048 * 1024 * 1024,
		BuildCacheReclaimable: 2048 * 1024 * 1024,
		TotalSize:             3672 * 1024 * 1024,
		TotalReclaimable:      2554 * 1024 * 1024,
		ImagesList: []model.ImageItem{
			{ID: "sha256:1111", ShortID: "1111", Repository: "app-server", Tag: "v1.0", Size: 300 * 1024 * 1024, Created: now},
			{ID: "sha256:2222", ShortID: "2222", Repository: "db-node", Tag: "latest", Size: 724 * 1024 * 1024, Created: now},
		},
	}

	v.SetDiskUsage(info)
	out := v.Render(false)

	if !strings.Contains(out, "DOCKER DISK & IMAGES AUDITOR") {
		t.Fatalf("Expected header in disk view")
	}
	if !strings.Contains(out, "app-server") || !strings.Contains(out, "db-node") {
		t.Fatalf("Expected image repositories in table")
	}

	// Test navigation
	v.MoveDown()
	selected := v.GetSelectedImage()
	if selected == nil || selected.Repository != "db-node" {
		t.Fatalf("Expected second image to be selected after MoveDown, got %+v", selected)
	}

	// Test layer view
	layers := []model.ImageLayerInfo{
		{ID: "layer-1", CreatedBy: "FROM alpine", Size: 5 * 1024 * 1024, CreatedAt: now},
		{ID: "layer-2", CreatedBy: "RUN build-app", Size: 120 * 1024 * 1024, CreatedAt: now, IsBloated: true},
	}
	v.SetLayers("db-node:latest", layers)

	layersOut := v.Render(false)
	if !strings.Contains(layersOut, "АНАЛИЗ СЛОЕВ ОБРАЗА") {
		t.Fatalf("Expected layer analysis section in rendered view")
	}
	if !strings.Contains(layersOut, "RUN build-app") {
		t.Fatalf("Expected command in layers table")
	}

	// Close layers
	v.CloseLayers()
	if v.ShowLayers {
		t.Fatalf("Expected layers to be closed")
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024 * 50, "50.0 MB"},
		{1024 * 1024 * 1024 * 4, "4.0 GB"},
	}

	for _, c := range cases {
		got := model.FormatBytes(c.input)
		if got != c.expected {
			t.Errorf("FormatBytes(%d) = %s, expected %s", c.input, got, c.expected)
		}
	}
}
