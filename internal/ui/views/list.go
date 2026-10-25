package views

import (
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Главная таблица со списком контейнеров
type ContainersListView struct {
	Cursor  int
	Offset  int
	Width   int
	Height  int
	Spinner spinner.Model
}

func NewContainersListView() ContainersListView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.ColorWarning)
	return ContainersListView{
		Spinner: s,
	}
}

func (v *ContainersListView) MoveUp(total int) {
	if total == 0 {
		return
	}
	if v.Cursor > 0 {
		v.Cursor--
		if v.Cursor < v.Offset {
			v.Offset = v.Cursor
		}
	}
}

func (v *ContainersListView) MoveDown(total int) {
	if total == 0 {
		return
	}
	if v.Cursor < total-1 {
		v.Cursor++
		visibleRows := v.Height - 3
		if visibleRows < 1 {
			visibleRows = 1
		}
		if v.Cursor >= v.Offset+visibleRows {
			v.Offset = v.Cursor - visibleRows + 1
		}
	}
}

// Обрезка или дополнение пробелами ровно до нужной ширины (без переноса строк)
func padOrTrunc(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > width {
		if width > 3 {
			return string(runes[:width-3]) + "..."
		}
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// Отрисовка таблицы контейнеров
func (v *ContainersListView) Render(containers []model.ContainerInfo, active bool) string {
	panelStyle := styles.PanelStyle
	if active {
		panelStyle = styles.ActivePanelStyle
	}

	boxWidth := v.Width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	// Точная ширина текста внутри рамки с учетом отступов
	textWidth := boxWidth - panelStyle.GetHorizontalPadding()
	if textWidth < 30 {
		textWidth = 30
	}

	if len(containers) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(styles.ColorMuted).
			Italic(true).
			Render("Контейнеры не найдены. Нажмите [r] для обновления.")
		return panelStyle.Width(boxWidth).Height(v.Height).Render(emptyMsg)
	}

	// Расчет ширины колонок под размер экрана
	nameWidth := 16
	cpuWidth := 6
	memWidth := 18

	if textWidth < 60 {
		nameWidth = 10
		cpuWidth = 5
		memWidth = 11
	}

	showImage := textWidth >= 75
	imageWidth := 18

	showPorts := textWidth >= 105
	portsWidth := 20

	// Detailed 2x wider gauge bars (12 chars: [■■■■■□□□□□]) on wide displays
	showBars := textWidth >= 135
	barLen := 12
	if showBars {
		cpuWidth = barLen + 1 + 6   // e.g. [■■■■■□□□□□] 50.0% -> 19 chars
		memWidth = barLen + 1 + 20  // e.g. [■■■■■□□□□□] 215.0 MB / 2.00 GB -> 33 chars
		nameWidth = 18
		imageWidth = 20
		portsWidth = 22
	}

	showUptime := textWidth >= 120
	// Calculate base width of preceding columns
	// pointer(2) + status(12) + gap(1) + nameWidth + gap(1) + cpuWidth + gap(1) + memWidth
	used := 2 + 12 + 1 + nameWidth + 1 + cpuWidth + 1 + memWidth
	if showImage {
		used += 1 + imageWidth
	}
	if showPorts {
		used += 1 + portsWidth
	}

	uptimeWidth := 0
	if showUptime && (textWidth-used) >= 14 {
		uptimeWidth = textWidth - used - 1
		if uptimeWidth > 28 {
			uptimeWidth = 28
		}
	} else {
		showUptime = false
	}

	// Construct header with centered labels over gauges
	var headerParts []string
	headerParts = append(headerParts, "  ", padOrTrunc("STATUS", 12), " ")
	headerParts = append(headerParts, padOrTrunc("NAME", nameWidth), " ")

	if showImage {
		headerParts = append(headerParts, padOrTrunc("IMAGE", imageWidth), " ")
	}
	if showPorts {
		headerParts = append(headerParts, padOrTrunc("PORTS", portsWidth), " ")
	}

	headerParts = append(headerParts, padOrTrunc("CPU", cpuWidth), " ")
	headerParts = append(headerParts, padOrTrunc("MEM USAGE", memWidth))

	if showUptime {
		headerParts = append(headerParts, " ", padOrTrunc("UPTIME", uptimeWidth))
	}

	headerStr := strings.Join(headerParts, "")
	renderedHeader := styles.TableHeaderStyle.Render(headerStr)

	visibleRows := v.Height - 4
	if visibleRows < 1 {
		visibleRows = 1
	}

	endIdx := v.Offset + visibleRows
	if endIdx > len(containers) {
		endIdx = len(containers)
	}

	var rows []string
	for i := v.Offset; i < endIdx; i++ {
		c := containers[i]
		isSelected := (i == v.Cursor)

		// Determine status text and color
		var statusText string
		var statusColor lipgloss.TerminalColor = styles.ColorMuted

		switch c.Status {
		case model.StatusRunning:
			statusText = "● RUNNING"
			statusColor = styles.ColorSuccess
		case model.StatusExited:
			statusText = "○ EXITED"
			statusColor = styles.ColorMuted
		case model.StatusPaused:
			statusText = "⏸ PAUSED"
			statusColor = styles.ColorWarning
		case model.StatusRestarting:
			statusText = "◌ RESTARTING"
			statusColor = styles.ColorWarning
		default:
			statusText = "○ " + strings.ToUpper(string(c.Status))
		}

		if c.IsLoading {
			statusText = v.Spinner.View() + " " + c.LoadingMsg
			statusColor = styles.ColorWarning
		}

		// Helper to render individual cell strictly without Lipgloss wrapping
		cell := func(text string, fg lipgloss.TerminalColor, bold bool) string {
			s := lipgloss.NewStyle().Foreground(fg).Bold(bold)
			if isSelected {
				s = s.Background(styles.ColorHighlight)
			}
			return s.Render(text)
		}

		pointerStr := "  "
		pointerFg := styles.ColorMuted
		if isSelected {
			pointerStr = "❯ "
			pointerFg = styles.ColorPrimary
		}

		var rowParts []string
		rowParts = append(rowParts, cell(pointerStr, pointerFg, true))
		rowParts = append(rowParts, cell(padOrTrunc(statusText, 12), statusColor, true))
		rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
		rowParts = append(rowParts, cell(padOrTrunc(c.Name, nameWidth), styles.ColorText, isSelected))

		if showImage {
			rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
			rowParts = append(rowParts, cell(padOrTrunc(c.Image, imageWidth), styles.ColorTextDim, false))
		}

		if showPorts {
			rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
			rowParts = append(rowParts, cell(padOrTrunc(c.Ports, portsWidth), styles.ColorTextDim, false))
		}

		// CPU Column (with detailed 10-step progress gauge)
		rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
		if showBars && c.Status == model.StatusRunning {
			bar := styles.RenderMiniBar(c.CPUPercent, barLen, isSelected)
			rowParts = append(rowParts, bar, cell(" ", styles.ColorMuted, false), cell(padOrTrunc(c.FormatCPU(), cpuWidth-barLen-1), styles.ColorText, false))
		} else {
			rowParts = append(rowParts, cell(padOrTrunc(c.FormatCPU(), cpuWidth), styles.ColorText, false))
		}

		// Memory Column (with detailed 10-step progress gauge)
		rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
		if showBars && c.Status == model.StatusRunning && c.MemoryUsage > 0 {
			memPct := 0.0
			if c.MemoryLimit > 0 {
				memPct = (float64(c.MemoryUsage) / float64(c.MemoryLimit)) * 100.0
			}
			bar := styles.RenderMiniBar(memPct, barLen, isSelected)
			rowParts = append(rowParts, bar, cell(" ", styles.ColorMuted, false), cell(padOrTrunc(c.FormatMemory(), memWidth-barLen-1), styles.ColorTextDim, false))
		} else {
			rowParts = append(rowParts, cell(padOrTrunc(c.FormatMemory(), memWidth), styles.ColorTextDim, false))
		}

		if showUptime {
			rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
			rowParts = append(rowParts, cell(padOrTrunc(c.StateText, uptimeWidth), styles.ColorMuted, false))
		}

		rowContent := strings.Join(rowParts, "")

		// Fill trailing gap up to textWidth so background highlight spans cleanly
		rowLen := lipgloss.Width(rowContent)
		if rowLen < textWidth {
			fill := strings.Repeat(" ", textWidth-rowLen)
			if isSelected {
				fill = lipgloss.NewStyle().Background(styles.ColorHighlight).Render(fill)
			}
			rowContent += fill
		}

		rows = append(rows, rowContent)
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		renderedHeader,
		strings.Join(rows, "\n"),
	)

	return panelStyle.Width(boxWidth).Height(v.Height).Render(content)
}
