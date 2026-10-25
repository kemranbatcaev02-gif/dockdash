package views

import (
	"fmt"
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Окно для просмотра логов контейнера со скроллом
type LogsView struct {
	Viewport       viewport.Model
	ContainerID    string
	ContainerName  string
	Image          string
	RawLines       []string
	Follow         bool
	ShowTimestamps bool
	Width          int
	Height         int
}

// Создаем окно логов
func NewLogsView() LogsView {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Foreground(styles.ColorText)
	return LogsView{
		Viewport:       vp,
		Follow:         true,
		ShowTimestamps: true,
		RawLines:       make([]string, 0, 500),
	}
}

// Переключаем окно на выбранный контейнер
func (v *LogsView) SetContainer(c model.ContainerInfo) {
	v.ContainerID = c.ID
	v.ContainerName = c.Name
	v.Image = c.Image
	v.RawLines = v.RawLines[:0]
	v.Follow = true
	v.Viewport.SetContent(lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("Подключаемся к логам контейнера...\n"))
}

// Добавляем новые строки в лог
func (v *LogsView) AppendLines(newLines []string) {
	for _, line := range newLines {
		if trimmed := strings.TrimRight(line, "\r\n"); trimmed != "" {
			v.RawLines = append(v.RawLines, trimmed)
		}
	}

	// Держим последние 1500 строк, чтобы не забивать память
	if len(v.RawLines) > 1500 {
		v.RawLines = v.RawLines[len(v.RawLines)-1500:]
	}

	v.rebuildViewportContent()

	if v.Follow {
		v.Viewport.GotoBottom()
	}
}

// Включить или выключить показ времени
func (v *LogsView) ToggleTimestamps() {
	v.ShowTimestamps = !v.ShowTimestamps
	v.rebuildViewportContent()
}

// Очистить логи
func (v *LogsView) Clear() {
	v.RawLines = v.RawLines[:0]
	v.Viewport.SetContent("")
}

// Включить или выключить автопрокрутку
func (v *LogsView) ToggleFollow() {
	v.Follow = !v.Follow
	if v.Follow {
		v.Viewport.GotoBottom()
	}
}

// Подгоняем размер под окно терминала
func (v *LogsView) Resize(width, height int) {
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

// Перерисовываем строки с подсветкой и временем
func (v *LogsView) rebuildViewportContent() {
	formatted := make([]string, 0, len(v.RawLines))
	for _, line := range v.RawLines {
		formatted = append(formatted, v.colorizeLogLine(line))
	}
	v.Viewport.SetContent(strings.Join(formatted, "\n"))
}

// Подсветка текста: уровни логов (ERROR, WARN, INFO) и HTTP коды
func (v *LogsView) colorizeLogLine(raw string) string {
	line := raw

	// Если в начале строки есть время вида "15:04:05"
	if len(raw) >= 9 && raw[2] == ':' && raw[5] == ':' && raw[8] == ' ' {
		timePart := raw[:8]
		rest := raw[9:]
		if !v.ShowTimestamps {
			line = rest
		} else {
			styledTime := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(timePart)
			line = styledTime + " " + rest
		}
	}

	upper := strings.ToUpper(line)

	// Подсветка ошибок и предупреждений
	if strings.Contains(upper, "ERROR") || strings.Contains(upper, "FATAL") || strings.Contains(upper, "PANIC") {
		return lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true).Render(line)
	}
	if strings.Contains(upper, "WARN") || strings.Contains(upper, "WARNING") {
		return lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(line)
	}
	if strings.Contains(upper, "INFO") {
		return lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render(line)
	}
	if strings.Contains(upper, "DEBUG") || strings.Contains(upper, "TRACE") {
		return lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(line)
	}

	// Подсветка HTTP статусов
	if strings.Contains(line, " 200 ") || strings.Contains(line, " 201 ") {
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(line)
	}
	if strings.Contains(line, " 404 ") || strings.Contains(line, " 500 ") || strings.Contains(line, " 502 ") {
		return lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(line)
	}

	return lipgloss.NewStyle().Foreground(styles.ColorText).Render(line)
}

// Отрисовка всей панели логов
func (v *LogsView) Render(active bool) string {
	availWidth := v.Width - 6
	if availWidth < 40 {
		availWidth = 40
	}

	followBadge := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[Follow: OFF]")
	if v.Follow {
		followBadge = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true).Render("[Follow: ON]")
	}

	timeBadge := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[Time: OFF]")
	if v.ShowTimestamps {
		timeBadge = lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render("[Time: ON]")
	}

	linesCount := lipgloss.NewStyle().Foreground(styles.ColorTextDim).Render(fmt.Sprintf("%d lines", len(v.RawLines)))

	headerLeft := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.SubTitleStyle.Render(fmt.Sprintf("LOGS: %s ", v.ContainerName)),
		lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("(%s)", v.Image)),
	)

	headerRight := lipgloss.JoinHorizontal(
		lipgloss.Right,
		linesCount,
		"  ",
		timeBadge,
		"  ",
		followBadge,
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
