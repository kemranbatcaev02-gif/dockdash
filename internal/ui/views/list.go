package views

import (
	"fmt"
	"sort"
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Тип строки в таблице списка
type RowType int

const (
	RowContainer RowType = iota
	RowProjectHeader
)

// Строка таблицы: контейнер либо заголовок Compose-проекта
type ListRow struct {
	Type         RowType
	ProjectName  string
	RunningCount int
	TotalCount   int
	IsCollapsed  bool
	Container    *model.ContainerInfo
}

// Главная таблица со списком контейнеров
type ContainersListView struct {
	Cursor            int
	Offset            int
	Width             int
	Height            int
	Spinner           spinner.Model
	GroupByProject    bool
	CollapsedProjects map[string]bool
}

func NewContainersListView() ContainersListView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.ColorWarning)
	return ContainersListView{
		Spinner:           s,
		GroupByProject:    true,
		CollapsedProjects: make(map[string]bool),
	}
}

// Строит плоский список строк для отображения с учетом группировки и сворачивания
func (v *ContainersListView) BuildVisibleRows(containers []model.ContainerInfo) []ListRow {
	if len(containers) == 0 {
		return nil
	}

	if !v.GroupByProject {
		rows := make([]ListRow, len(containers))
		for i := range containers {
			rows[i] = ListRow{
				Type:      RowContainer,
				Container: &containers[i],
			}
		}
		return rows
	}

	// Группируем контейнеры по ComposeProject
	projectGroups := make(map[string][]*model.ContainerInfo)
	var projectNames []string

	for i := range containers {
		proj := containers[i].ComposeProject
		if _, exists := projectGroups[proj]; !exists {
			projectNames = append(projectNames, proj)
		}
		projectGroups[proj] = append(projectGroups[proj], &containers[i])
	}

	// Сортируем проекты: именованные проекты по алфавиту, standalone ("") всегда в конце
	sort.SliceStable(projectNames, func(i, j int) bool {
		if projectNames[i] == "" {
			return false
		}
		if projectNames[j] == "" {
			return true
		}
		return strings.ToLower(projectNames[i]) < strings.ToLower(projectNames[j])
	})

	var rows []ListRow
	for _, proj := range projectNames {
		groupList := projectGroups[proj]
		runningCount := 0
		for _, c := range groupList {
			if c.Status == model.StatusRunning {
				runningCount++
			}
		}

		isCollapsed := false
		if v.CollapsedProjects != nil {
			isCollapsed = v.CollapsedProjects[proj]
		}

		// Добавляем строку-заголовок проекта
		rows = append(rows, ListRow{
			Type:         RowProjectHeader,
			ProjectName:  proj,
			RunningCount: runningCount,
			TotalCount:   len(groupList),
			IsCollapsed:  isCollapsed,
		})

		// Если проект не свернут, добавляем его контейнеры
		if !isCollapsed {
			for _, c := range groupList {
				rows = append(rows, ListRow{
					Type:        RowContainer,
					ProjectName: proj,
					Container:   c,
				})
			}
		}
	}

	return rows
}

func (v *ContainersListView) ToggleProjectCollapse(projectName string) {
	if v.CollapsedProjects == nil {
		v.CollapsedProjects = make(map[string]bool)
	}
	v.CollapsedProjects[projectName] = !v.CollapsedProjects[projectName]
}

func (v *ContainersListView) ToggleGrouping() bool {
	v.GroupByProject = !v.GroupByProject
	return v.GroupByProject
}

func (v *ContainersListView) Clamp(total int) {
	if total == 0 {
		v.Cursor = 0
		v.Offset = 0
		return
	}
	if v.Cursor >= total {
		v.Cursor = total - 1
	}
	if v.Cursor < 0 {
		v.Cursor = 0
	}
	if v.Offset > v.Cursor {
		v.Offset = v.Cursor
	}
}

func (v *ContainersListView) GetSelectedRow(containers []model.ContainerInfo) *ListRow {
	rows := v.BuildVisibleRows(containers)
	if len(rows) == 0 || v.Cursor < 0 || v.Cursor >= len(rows) {
		return nil
	}
	return &rows[v.Cursor]
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

	rowsList := v.BuildVisibleRows(containers)
	if len(rowsList) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(styles.ColorMuted).
			Italic(true).
			Render("Контейнеры не найдены. Нажмите [r] для обновления.")
		return panelStyle.Width(boxWidth).Height(v.Height).Render(emptyMsg)
	}

	v.Clamp(len(rowsList))

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

	showBars := textWidth >= 135
	barLen := 12
	if showBars {
		cpuWidth = barLen + 1 + 6
		memWidth = barLen + 1 + 20
		nameWidth = 18
		imageWidth = 20
		portsWidth = 22
	}

	showUptime := textWidth >= 120
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

	// Заголовок таблицы
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
	if endIdx > len(rowsList) {
		endIdx = len(rowsList)
	}

	var renderedRows []string
	for i := v.Offset; i < endIdx; i++ {
		row := rowsList[i]
		isSelected := (i == v.Cursor)

		if row.Type == RowProjectHeader {
			// Отрисовка разделителя проекта / стека
			arrow := "▼"
			actionHint := "[Tab: скрыть]"
			if row.IsCollapsed {
				arrow = "▶"
				actionHint = "[Tab: открыть]"
			}

			displayName := row.ProjectName
			if displayName == "" {
				displayName = "STANDALONE"
			} else {
				displayName = "STACK: " + displayName
			}

			runningStatus := fmt.Sprintf("(%d/%d running)", row.RunningCount, row.TotalCount)
			pointer := "  "
			if isSelected {
				pointer = "❯ "
			}

			headerTitle := fmt.Sprintf("%s%s %s %s", pointer, arrow, displayName, runningStatus)
			remainSpace := textWidth - len([]rune(headerTitle)) - len([]rune(actionHint)) - 1
			if remainSpace < 1 {
				remainSpace = 1
			}
			headerLine := headerTitle + strings.Repeat(" ", remainSpace) + actionHint
			headerLine = padOrTrunc(headerLine, textWidth)

			style := lipgloss.NewStyle().Bold(true)
			if isSelected {
				style = style.
					Foreground(lipgloss.Color("#FFFFFF")).
					Background(styles.ColorHighlight)
			} else {
				style = style.
					Foreground(styles.ColorPrimary).
					Background(lipgloss.Color("#2A283E"))
			}

			renderedRows = append(renderedRows, style.Render(headerLine))
			continue
		}

		c := row.Container

		// Определение статуса и цвета
		var statusText string
		var statusColor lipgloss.TerminalColor = styles.ColorMuted

		switch c.Status {
		case model.StatusRunning:
			statusText = "● RUNNING"
			statusColor = styles.ColorSuccess
		case model.StatusExited:
			if c.OOMKilled || c.ExitCode == 137 {
				statusText = "💀 OOM KILLED"
				statusColor = styles.ColorDanger
			} else {
				statusText = "○ EXITED"
				statusColor = styles.ColorMuted
			}
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

		displayName := c.Name
		if v.GroupByProject && row.ProjectName != "" {
			displayName = "  " + c.Name
		}

		var rowParts []string
		rowParts = append(rowParts, cell(pointerStr, pointerFg, true))
		rowParts = append(rowParts, cell(padOrTrunc(statusText, 12), statusColor, true))
		rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
		rowParts = append(rowParts, cell(padOrTrunc(displayName, nameWidth), styles.ColorText, isSelected))

		if showImage {
			rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
			rowParts = append(rowParts, cell(padOrTrunc(c.Image, imageWidth), styles.ColorTextDim, false))
		}

		if showPorts {
			rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
			rowParts = append(rowParts, cell(padOrTrunc(c.Ports, portsWidth), styles.ColorTextDim, false))
		}

		// CPU Column с индикацией cgroups throttling
		cpuStr := c.FormatCPU()
		cpuFg := styles.ColorText
		if c.CPUThrottledPercent > 0 && c.Status == model.StatusRunning {
			cpuStr = fmt.Sprintf("%.0f%%⚠️", c.CPUPercent)
			cpuFg = styles.ColorWarning
		}

		rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
		if showBars && c.Status == model.StatusRunning {
			bar := styles.RenderMiniBar(c.CPUPercent, barLen, isSelected)
			rowParts = append(rowParts, bar, cell(" ", styles.ColorMuted, false), cell(padOrTrunc(cpuStr, cpuWidth-barLen-1), cpuFg, false))
		} else {
			rowParts = append(rowParts, cell(padOrTrunc(cpuStr, cpuWidth), cpuFg, false))
		}

		// Memory Column с предупреждением OOM Risk
		memStr := c.FormatMemory()
		memFg := styles.ColorTextDim
		if c.OOMRisk && c.Status == model.StatusRunning {
			memFg = styles.ColorDanger
		}

		rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
		if showBars && c.Status == model.StatusRunning && c.MemoryUsage > 0 {
			memPct := 0.0
			if c.MemoryLimit > 0 {
				memPct = (float64(c.MemoryUsage) / float64(c.MemoryLimit)) * 100.0
			}
			bar := styles.RenderMiniBar(memPct, barLen, isSelected)
			if c.OOMRisk {
				memStr = "⚠️" + memStr
			}
			rowParts = append(rowParts, bar, cell(" ", styles.ColorMuted, false), cell(padOrTrunc(memStr, memWidth-barLen-1), memFg, c.OOMRisk))
		} else {
			if c.OOMRisk {
				memStr = "⚠️" + memStr
			}
			rowParts = append(rowParts, cell(padOrTrunc(memStr, memWidth), memFg, c.OOMRisk))
		}

		if showUptime {
			rowParts = append(rowParts, cell(" ", styles.ColorMuted, false))
			rowParts = append(rowParts, cell(padOrTrunc(c.StateText, uptimeWidth), styles.ColorMuted, false))
		}

		rowContent := strings.Join(rowParts, "")

		rowLen := lipgloss.Width(rowContent)
		if rowLen < textWidth {
			fill := strings.Repeat(" ", textWidth-rowLen)
			if isSelected {
				fill = lipgloss.NewStyle().Background(styles.ColorHighlight).Render(fill)
			}
			rowContent += fill
		}

		renderedRows = append(renderedRows, rowContent)
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		renderedHeader,
		strings.Join(renderedRows, "\n"),
	)

	return panelStyle.Width(boxWidth).Height(v.Height).Render(content)
}
