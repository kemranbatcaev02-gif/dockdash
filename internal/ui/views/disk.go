package views

import (
	"fmt"
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Экран аудита дискового пространства и слоев образов (dive-style)
type DiskView struct {
	Viewport        viewport.Model
	DiskUsage       model.DiskUsageInfo
	ImageCursor     int
	SelectedImageID string
	Layers          []model.ImageLayerInfo
	ShowLayers      bool
	Width           int
	Height          int
}

func NewDiskView() DiskView {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Foreground(styles.ColorText)
	return DiskView{
		Viewport: vp,
	}
}

func (v *DiskView) SetDiskUsage(info model.DiskUsageInfo) {
	v.DiskUsage = info
	if v.ImageCursor >= len(info.ImagesList) && len(info.ImagesList) > 0 {
		v.ImageCursor = len(info.ImagesList) - 1
	}
	v.RebuildContent()
}

func (v *DiskView) SetLayers(imageID string, layers []model.ImageLayerInfo) {
	v.SelectedImageID = imageID
	v.Layers = layers
	v.ShowLayers = true
	v.RebuildContent()
}

func (v *DiskView) CloseLayers() {
	v.ShowLayers = false
	v.SelectedImageID = ""
	v.Layers = nil
	v.RebuildContent()
}

func (v *DiskView) MoveUp() {
	if v.ImageCursor > 0 {
		v.ImageCursor--
		v.RebuildContent()
	}
}

func (v *DiskView) MoveDown() {
	if v.ImageCursor < len(v.DiskUsage.ImagesList)-1 {
		v.ImageCursor++
		v.RebuildContent()
	}
}

func (v *DiskView) GetSelectedImage() *model.ImageItem {
	if len(v.DiskUsage.ImagesList) == 0 || v.ImageCursor < 0 || v.ImageCursor >= len(v.DiskUsage.ImagesList) {
		return nil
	}
	return &v.DiskUsage.ImagesList[v.ImageCursor]
}

func (v *DiskView) Resize(width, height int) {
	v.Width = width
	v.Height = height
	vpHeight := height - 4
	if vpHeight < 5 {
		vpHeight = 5
	}
	vpWidth := width - 6
	if vpWidth < 20 {
		vpWidth = 20
	}
	v.Viewport.Width = vpWidth
	v.Viewport.Height = vpHeight
	v.RebuildContent()
}

// Формирует текстовое содержимое для Viewport
func (v *DiskView) RebuildContent() {
	var sb strings.Builder

	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary)
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorBorder).
		Padding(0, 1)

	// 1. Сводные карточки диска
	imgCard := cardStyle.Render(fmt.Sprintf(
		"%s\nВсего: %s\nОчищаемо: %s (%d шт)",
		lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary).Render("📦 Образы (Images)"),
		model.FormatBytes(v.DiskUsage.ImagesTotalSize),
		model.FormatBytes(v.DiskUsage.ImagesReclaimable),
		v.DiskUsage.ImagesCount,
	))

	cntCard := cardStyle.Render(fmt.Sprintf(
		"%s\nВсего: %s\nОчищаемо: %s (%d шт)",
		lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSuccess).Render("🚢 Контейнеры"),
		model.FormatBytes(v.DiskUsage.ContainersTotalSize),
		model.FormatBytes(v.DiskUsage.ContainersReclaimable),
		v.DiskUsage.ContainersCount,
	))

	volCard := cardStyle.Render(fmt.Sprintf(
		"%s\nВсего: %s\nОчищаемо: %s (%d шт)",
		lipgloss.NewStyle().Bold(true).Foreground(styles.ColorWarning).Render("💾 Тома (Volumes)"),
		model.FormatBytes(v.DiskUsage.VolumesTotalSize),
		model.FormatBytes(v.DiskUsage.VolumesReclaimable),
		v.DiskUsage.VolumesCount,
	))

	cacheCard := cardStyle.Render(fmt.Sprintf(
		"%s\nВсего: %s\nОчищаемо: %s",
		lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDanger).Render("⚡ Build Cache"),
		model.FormatBytes(v.DiskUsage.BuildCacheTotalSize),
		model.FormatBytes(v.DiskUsage.BuildCacheReclaimable),
	))

	summaryCards := lipgloss.JoinHorizontal(lipgloss.Top, imgCard, " ", cntCard, " ", volCard, " ", cacheCard)
	sb.WriteString(summaryCards + "\n\n")

	// 2. Если открыт анализ слоев (dive-style)
	if v.ShowLayers {
		sb.WriteString(sectionStyle.Render(fmt.Sprintf("● АНАЛИЗ СЛОЕВ ОБРАЗА: %s (Всего слоев: %d) — [Esc: закрыть]", v.SelectedImageID, len(v.Layers))) + "\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(strings.Repeat("─", v.Viewport.Width)) + "\n")

		header := fmt.Sprintf("  %-12s %-12s %s", "LAYER ID", "SIZE", "COMMAND / INSTRUCTION")
		sb.WriteString(styles.TableHeaderStyle.Render(header) + "\n")

		for _, l := range v.Layers {
			sizeStr := model.FormatBytes(l.Size)
			sizeStyle := lipgloss.NewStyle().Foreground(styles.ColorText)
			badge := "  "
			if l.IsBloated {
				sizeStyle = lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true)
				badge = "⚠️ "
			}

			line := fmt.Sprintf("%s%-12s %-12s %s", badge, l.ID, sizeStyle.Render(sizeStr), l.CreatedBy)
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	// 3. Таблица локальных образов
	sb.WriteString(sectionStyle.Render("● ОБРАЗЫ DOCKER (Нажмите [Enter] для анализа слоев dive-style)") + "\n")
	imgHeader := fmt.Sprintf("  %-24s %-14s %-12s %-14s %s", "REPOSITORY", "TAG", "SIZE", "CREATED", "ID")
	sb.WriteString(styles.TableHeaderStyle.Render(imgHeader) + "\n")

	for i, img := range v.DiskUsage.ImagesList {
		isSelected := (i == v.ImageCursor && !v.ShowLayers)
		pointer := "  "
		if isSelected {
			pointer = "❯ "
		}

		repo := padOrTrunc(img.Repository, 24)
		tag := padOrTrunc(img.Tag, 14)
		size := padOrTrunc(model.FormatBytes(img.Size), 12)
		created := padOrTrunc(img.Created.Format("2006-01-02 15:04"), 14)
		shortID := img.ShortID

		line := fmt.Sprintf("%s%-24s %-14s %-12s %-14s %s", pointer, repo, tag, size, created, shortID)

		if isSelected {
			sb.WriteString(lipgloss.NewStyle().
				Background(styles.ColorHighlight).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}

	v.Viewport.SetContent(sb.String())
}

func (v *DiskView) Render(active bool) string {
	availWidth := v.Width - 6
	if availWidth < 40 {
		availWidth = 40
	}

	totalReclaimBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#11111B")).
		Background(styles.ColorSuccess).
		Padding(0, 1).
		Render(fmt.Sprintf("RECLAIMABLE: %s", model.FormatBytes(v.DiskUsage.TotalReclaimable)))

	headerLeft := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.SubTitleStyle.Render("⚡ DOCKER DISK & IMAGES AUDITOR"),
	)

	headerRight := lipgloss.JoinHorizontal(
		lipgloss.Right,
		totalReclaimBadge,
	)

	gap := availWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if gap < 1 {
		gap = 1
	}
	headerSpacing := strings.Repeat(" ", gap)

	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		headerLeft,
		headerSpacing,
		headerRight,
	)

	panelStyle := styles.PanelStyle
	if active {
		panelStyle = styles.ActivePanelStyle
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", availWidth)),
		v.Viewport.View(),
	)

	return panelStyle.Width(availWidth).Height(v.Height).Render(content)
}
