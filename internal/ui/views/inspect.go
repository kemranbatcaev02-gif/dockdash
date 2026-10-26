package views

import (
	"fmt"
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Экран глубокого просмотра настроек контейнера [i]
type InspectView struct {
	Viewport viewport.Model
	Details  model.ContainerDetails
	Width    int
	Height   int
}

func NewInspectView() InspectView {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Foreground(styles.ColorText)
	return InspectView{
		Viewport: vp,
	}
}

// Заполнение экрана данными выбранного контейнера
func (v *InspectView) SetDetails(d model.ContainerDetails) {
	v.Details = d

	labelKey := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary).Width(16)
	valStyle := lipgloss.NewStyle().Foreground(styles.ColorText)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary).MarginTop(1)

	var sb strings.Builder

	// Основные параметры
	sb.WriteString(sectionStyle.Render("● ОСНОВНЫЕ ПАРАМЕТРЫ") + "\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("ID:"), valStyle.Render(d.ID)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Имя:"), valStyle.Render(d.Name)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Образ:"), valStyle.Render(d.Image)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Статус:"), valStyle.Render(d.Status)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Команда:"), valStyle.Render(d.Command)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Рестарт:"), valStyle.Render(d.RestartPolicy)))

	exitCodeStr := fmt.Sprintf("%d", d.ExitCode)
	if d.OOMKilled || d.ExitCode == 137 {
		exitCodeStr = lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true).Render("137 (OOM KILLED — процесс принудительно убит из-за нехватки памяти)")
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Код выхода:"), exitCodeStr))

	// Ресурсы и ограничения cgroups
	sb.WriteString(sectionStyle.Render("● РЕСУРСЫ И CGROUPS") + "\n")
	throttlingStr := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("0% (троттлинг не зафиксирован)")
	if d.CPUThrottledPercent > 0 {
		throttlingStr = lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true).Render(fmt.Sprintf("⚠️ %.1f%% (CFS Quota Throttling — контейнеру не хватает квоты CPU!)", d.CPUThrottledPercent))
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("CPU Throttling:"), throttlingStr))

	oomRiskStr := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("В норме (< 90%)")
	if d.OOMRisk {
		oomRiskStr = lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true).Render("⚠️ КРИТИЧЕСКИЙ (Использование памяти > 90% лимита!)")
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Риск OOM:"), oomRiskStr))

	// Сеть
	sb.WriteString(sectionStyle.Render("● СЕТЬ") + "\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("IP-адрес:"), valStyle.Render(d.IPAddress)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Шлюз:"), valStyle.Render(d.Gateway)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("MAC-адрес:"), valStyle.Render(d.MACAddress)))

	// Даты
	sb.WriteString(sectionStyle.Render("● ВРЕМЯ") + "\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Создан:"), valStyle.Render(d.CreatedAt)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Запущен:"), valStyle.Render(d.StartedAt)))
	if d.FinishedAt != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", labelKey.Render("Остановлен:"), valStyle.Render(d.FinishedAt)))
	}

	// Тома
	sb.WriteString(sectionStyle.Render("● ТОМА И ПУТИ (MOUNTS)") + "\n")
	if len(d.Mounts) == 0 {
		sb.WriteString("  — Нет примонтированных томов\n")
	} else {
		for _, m := range d.Mounts {
			sb.WriteString(fmt.Sprintf("  • %s\n", m))
		}
	}

	// Переменные окружения
	sb.WriteString(sectionStyle.Render("● ПЕРЕМЕННЫЕ ОКРУЖЕНИЯ (ENV)") + "\n")
	if len(d.Env) == 0 {
		sb.WriteString("  — Нет переменных окружения\n")
	} else {
		for _, e := range d.Env {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				k := lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(parts[0])
				val := lipgloss.NewStyle().Foreground(styles.ColorTextDim).Render(parts[1])
				sb.WriteString(fmt.Sprintf("  • %s=%s\n", k, val))
			} else {
				sb.WriteString(fmt.Sprintf("  • %s\n", e))
			}
		}
	}

	// Лейблы
	if len(d.Labels) > 0 {
		sb.WriteString(sectionStyle.Render("● МЕТКИ (LABELS)") + "\n")
		for k, val := range d.Labels {
			sb.WriteString(fmt.Sprintf("  • %s: %s\n", k, val))
		}
	}

	v.Viewport.SetContent(sb.String())
	v.Viewport.GotoTop()
}

func (v *InspectView) Resize(width, height int) {
	v.Width = width
	v.Height = height
	vpHeight := height - 4
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := width - 6
	if vpWidth < 10 {
		vpWidth = 10
	}
	v.Viewport.Width = vpWidth
	v.Viewport.Height = vpHeight
}

func (v *InspectView) Render(active bool) string {
	availWidth := v.Width - 6
	if availWidth < 40 {
		availWidth = 40
	}

	header := styles.SubTitleStyle.Render(fmt.Sprintf("🔍 ИНСПЕКЦИЯ: %s (%s)", v.Details.Name, v.Details.Image))

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

	return panelStyle.Width(v.Width - 4).Height(v.Height).Render(content)
}
