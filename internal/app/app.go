package app

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"dockdash/internal/docker"
	"dockdash/internal/model"
	"dockdash/internal/ui/styles"
	"dockdash/internal/ui/views"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Текущий активный экран
type ActiveView int

const (
	ViewList ActiveView = iota
	ViewLogs
	ViewInspect
)

// Режим сортировки контейнеров
type SortMode int

const (
	SortDefault SortMode = iota
	SortName
	SortStatus
	SortCPU
	SortMemory
)

// Внутренние события программы
type containersMsg []model.ContainerInfo
type daemonInfoMsg model.DaemonInfo
type inspectDetailsMsg model.ContainerDetails
type actionResultMsg model.ActionResult
type logLineMsg string
type statsUpdateMsg model.StatsUpdate
type toastClearMsg struct{}
type tickMsg time.Time
type execFinishedMsg struct{ err error }

// Главное состояние программы
type Model struct {
	docker      docker.Service
	daemonInfo  model.DaemonInfo
	containers  []model.ContainerInfo
	listView    views.ContainersListView
	logsView    views.LogsView
	inspectView views.InspectView
	filterBar   views.FilterBar
	modal       views.ConfirmationModal
	help        views.HelpView
	activeView  ActiveView
	sortMode    SortMode
	width       int
	height      int
	toast       string
	toastIsErr  bool
	logsCancel  context.CancelFunc
	logChan     chan string
	statsCancel context.CancelFunc
}

// Создаем главное состояние программы
func New(service docker.Service) Model {
	return Model{
		docker:      service,
		listView:    views.NewContainersListView(),
		logsView:    views.NewLogsView(),
		inspectView: views.NewInspectView(),
		filterBar:   views.NewFilterBar(),
		activeView:  ViewList,
	}
}

// Старт программы: загружаем контейнеры и запускаем таймеры
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadContainersCmd(),
		m.loadDaemonInfoCmd(),
		m.listView.Spinner.Tick,
		m.tickCmd(),
	)
}

// Обработка нажатий клавиш и событий
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcDimensions()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.listView.Spinner, cmd = m.listView.Spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tickMsg:
		cmds = append(cmds, m.pollStatsCmd(), m.loadDaemonInfoCmd(), m.tickCmd())

	case toastClearMsg:
		m.toast = ""

	case daemonInfoMsg:
		m.daemonInfo = model.DaemonInfo(msg)

	case containersMsg:
		m.containers = msg

	case inspectDetailsMsg:
		m.inspectView.SetDetails(model.ContainerDetails(msg))

	case statsUpdateMsg:
		for i := range m.containers {
			if m.containers[i].ID == msg.ContainerID || m.containers[i].ShortID == msg.ContainerID {
				m.containers[i].CPUPercent = msg.CPUPercent
				m.containers[i].MemoryUsage = msg.MemoryUsage
				m.containers[i].MemoryLimit = msg.MemoryLimit
				m.containers[i].CPUThrottledPercent = msg.CPUThrottledPercent
				m.containers[i].OOMRisk = msg.OOMRisk
				break
			}
		}

	case logLineMsg:
		if m.activeView == ViewLogs {
			m.logsView.AppendLines([]string{string(msg)})
			if m.logChan != nil {
				cmds = append(cmds, m.waitForNextLogCmd())
			}
		}

	case actionResultMsg:
		for i := range m.containers {
			if m.containers[i].ID == msg.ContainerID || m.containers[i].ShortID == msg.ContainerID {
				m.containers[i].IsLoading = false
				m.containers[i].LoadingMsg = ""
				break
			}
		}

		if msg.Err != nil {
			m.toast = fmt.Sprintf("Ошибка: %v", msg.Err)
			m.toastIsErr = true
		} else {
			m.toast = fmt.Sprintf("Успешно выполнено: %s на '%s'", msg.Action, msg.ContainerName)
			m.toastIsErr = false
		}
		cmds = append(cmds, m.clearToastCmd(), m.loadContainersCmd(), m.loadDaemonInfoCmd())

	case execFinishedMsg:
		if msg.err != nil {
			m.toast = fmt.Sprintf("✖ Ошибка шелла: %v", msg.err)
			m.toastIsErr = true
		} else {
			m.toast = "✔ Сессия шелла завершена"
			m.toastIsErr = false
		}
		return m, tea.Batch(tea.ClearScreen, m.loadContainersCmd(), m.clearToastCmd())

	case tea.KeyMsg:
		key := strings.ToLower(msg.String())

		// Выход из программы по q, й или Ctrl+C
		if !m.filterBar.Active {
			switch key {
			case "q", "й", "ctrl+c":
				m.stopLogsStream()
				return m, tea.Quit
			}
		}

		// Если открыто окно подтверждения
		if m.modal.Active {
			switch key {
			case "y", "н", "enter":
				action := m.modal.Action
				id := m.modal.ContainerID
				name := m.modal.ContainerName
				m.modal.Hide()
				cmds = append(cmds, m.executeActionCmd(id, name, action))
			case "n", "т", "esc", "q", "й":
				m.modal.Hide()
			}
			return m, nil
		}

		// Режим ввода поискового фильтра
		if m.filterBar.Active {
			switch msg.Type {
			case tea.KeyEsc:
				m.filterBar.Reset()
			case tea.KeyEnter:
				m.filterBar.Blur()
			default:
				var cmd tea.Cmd
				m.filterBar.Input, cmd = m.filterBar.Input.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Управление на экране информации (Inspect)
		if m.activeView == ViewInspect {
			rows := m.getNavigableRows()
			switch key {
			case "esc", "backspace":
				m.activeView = ViewList
				m.recalcDimensions()
				return m, nil
			case "up", "k", "л":
				m.listView.MoveUp(len(rows))
				if c := m.getSelectedContainer(); c != nil {
					return m, m.inspectContainerCmd(c.ID)
				}
				return m, nil
			case "down", "j", "о":
				m.listView.MoveDown(len(rows))
				if c := m.getSelectedContainer(); c != nil {
					return m, m.inspectContainerCmd(c.ID)
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.inspectView.Viewport, cmd = m.inspectView.Viewport.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
		}

		// Управление на экране логов (Logs)
		if m.activeView == ViewLogs {
			// Если активен ввод поисковой строки в логах
			if m.logsView.SearchActive {
				switch msg.Type {
				case tea.KeyEsc:
					m.logsView.ClearSearch()
					m.recalcDimensions()
					return m, nil
				case tea.KeyEnter:
					m.logsView.ApplySearch(m.logsView.SearchInput.Value())
					m.recalcDimensions()
					return m, nil
				default:
					var cmd tea.Cmd
					m.logsView.SearchInput, cmd = m.logsView.SearchInput.Update(msg)
					return m, cmd
				}
			}

			// Если нажата клавиша с Shift (например заглавные N или S)
			rawKey := msg.String()
			if rawKey == "N" || rawKey == "Т" {
				m.logsView.PrevMatch()
				return m, nil
			}
			if rawKey == "S" || rawKey == "Ы" {
				sinceStr := m.logsView.CycleSince()
				m.logsView.Clear()
				m.toast = fmt.Sprintf("Фильтр времени логов: %s", m.logsView.SinceLabel())
				m.toastIsErr = false
				return m, tea.Batch(
					m.startLogsStreamCmd(m.logsView.ContainerID, sinceStr),
					m.clearToastCmd(),
				)
			}

			switch key {
			case "esc", "backspace":
				if m.logsView.SearchQuery != "" {
					m.logsView.ClearSearch()
					return m, nil
				}
				m.stopLogsStream()
				m.activeView = ViewList
				m.recalcDimensions()
				return m, nil
			case "/", "ю":
				m.logsView.FocusSearch()
				m.recalcDimensions()
				return m, nil
			case "n", "т":
				m.logsView.NextMatch()
				return m, nil
			case "e", "у":
				errOnly := m.logsView.ToggleErrorIsolation()
				if errOnly {
					m.toast = "⚠️ Изоляция ошибок (Error Isolation) включена"
				} else {
					m.toast = "Отображаются все строки лога"
				}
				m.toastIsErr = false
				return m, m.clearToastCmd()
			case "f", "а":
				m.logsView.ToggleFollow()
				return m, nil
			case "t", "е":
				m.logsView.ToggleTimestamps()
				return m, nil
			case "c", "с":
				m.logsView.Clear()
				return m, nil
			default:
				var cmd tea.Cmd
				m.logsView.Viewport, cmd = m.logsView.Viewport.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
		}

		// Управление на главном списке контейнеров
		navRows := m.getNavigableRows()

		// Заглавная K / Л вызывает принудительное завершение (Kill)
		if msg.String() == "K" || msg.String() == "Л" {
			if c := m.getSelectedContainer(); c != nil {
				m.modal.Show(c.ID, c.Name, model.ActionKill)
			}
			return m, nil
		}

		switch key {
		case "up", "k", "л":
			m.listView.MoveUp(len(navRows))

		case "down", "j", "о":
			m.listView.MoveDown(len(navRows))

		case "home":
			m.listView.Cursor = 0
			m.listView.Offset = 0

		case "end":
			if len(navRows) > 0 {
				m.listView.Cursor = len(navRows) - 1
			}

		case "tab": // Свернуть / развернуть стек
			selectedRow := m.getSelectedRow()
			if selectedRow != nil {
				if selectedRow.Type == views.RowProjectHeader {
					m.listView.ToggleProjectCollapse(selectedRow.ProjectName)
					m.recalcDimensions()
				} else if selectedRow.ProjectName != "" {
					m.listView.ToggleProjectCollapse(selectedRow.ProjectName)
					m.recalcDimensions()
				}
			}

		case "g", "п": // Переключение режима группировки (Compose Stacks / Flat)
			isGrouped := m.listView.ToggleGrouping()
			if isGrouped {
				m.toast = "Группировка: по стекам Docker Compose"
			} else {
				m.toast = "Группировка: плоский список контейнеров"
			}
			m.toastIsErr = false
			cmds = append(cmds, m.clearToastCmd())

		case " ": // Space: Toggle Start / Stop
			selectedRow := m.getSelectedRow()
			if selectedRow == nil {
				return m, nil
			}
			if selectedRow.Type == views.RowProjectHeader {
				// Управление всеми контейнерами в стеке
				projName := selectedRow.ProjectName
				visible := m.getVisibleContainers()
				var stackContainers []model.ContainerInfo
				for _, c := range visible {
					if c.ComposeProject == projName {
						stackContainers = append(stackContainers, c)
					}
				}
				if len(stackContainers) == 0 {
					return m, nil
				}

				allStopped := (selectedRow.RunningCount == 0)
				targetAction := model.ActionStop
				actionName := "Остановка"
				if allStopped {
					targetAction = model.ActionStart
					actionName = "Запуск"
				}

				displayName := projName
				if displayName == "" {
					displayName = "standalone"
				}
				m.toast = fmt.Sprintf("%s стека '%s' (%d контейнеров)...", actionName, displayName, len(stackContainers))
				m.toastIsErr = false

				for _, c := range stackContainers {
					msg := "stopping..."
					if allStopped {
						msg = "starting..."
					}
					m.markLoading(c.ID, msg)
					cmds = append(cmds, m.executeActionCmd(c.ID, c.Name, targetAction))
				}
				cmds = append(cmds, m.clearToastCmd())
			} else if c := selectedRow.Container; c != nil {
				if c.Status == model.StatusRunning {
					m.markLoading(c.ID, "stopping...")
					cmds = append(cmds, m.executeActionCmd(c.ID, c.Name, model.ActionStop))
				} else {
					m.markLoading(c.ID, "starting...")
					cmds = append(cmds, m.executeActionCmd(c.ID, c.Name, model.ActionStart))
				}
			}

		case "x", "ч", "e", "у": // Интерактивный шелл в контейнере
			selectedRow := m.getSelectedRow()
			if selectedRow == nil {
				return m, nil
			}
			if selectedRow.Type == views.RowProjectHeader {
				m.toast = "⚠️ Выберите контейнер в стеке для шелла"
				m.toastIsErr = true
				return m, m.clearToastCmd()
			}
			c := selectedRow.Container
			if c == nil {
				return m, nil
			}
			if c.Status != model.StatusRunning {
				m.toast = "⚠️ Нельзя открыть шелл в остановленном контейнере"
				m.toastIsErr = true
				return m, m.clearToastCmd()
			}
			cmd, err := m.docker.GetExecCommand(context.Background(), c.ID)
			if err != nil {
				m.toast = fmt.Sprintf("✖ Ошибка подготовки шелла: %v", err)
				m.toastIsErr = true
				return m, m.clearToastCmd()
			}
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return execFinishedMsg{err: err}
			})

		case "r", "к": // Restart
			selectedRow := m.getSelectedRow()
			if selectedRow == nil {
				return m, nil
			}
			if selectedRow.Type == views.RowProjectHeader {
				projName := selectedRow.ProjectName
				visible := m.getVisibleContainers()
				var stackContainers []model.ContainerInfo
				for _, c := range visible {
					if c.ComposeProject == projName {
						stackContainers = append(stackContainers, c)
					}
				}
				if len(stackContainers) == 0 {
					return m, nil
				}
				displayName := projName
				if displayName == "" {
					displayName = "standalone"
				}
				m.toast = fmt.Sprintf("Перезапуск стека '%s' (%d контейнеров)...", displayName, len(stackContainers))
				m.toastIsErr = false

				for _, c := range stackContainers {
					m.markLoading(c.ID, "restarting...")
					cmds = append(cmds, m.executeActionCmd(c.ID, c.Name, model.ActionRestart))
				}
				cmds = append(cmds, m.clearToastCmd())
			} else if c := selectedRow.Container; c != nil {
				m.markLoading(c.ID, "restarting...")
				cmds = append(cmds, m.executeActionCmd(c.ID, c.Name, model.ActionRestart))
			}

		case "s", "ы": // Cycle Sort Mode
			m.sortMode = (m.sortMode + 1) % 5
			sortNames := []string{"Default", "Name (A-Z)", "Status", "CPU%", "Memory"}
			m.toast = fmt.Sprintf("Sorted by: %s", sortNames[m.sortMode])
			m.toastIsErr = false
			cmds = append(cmds, m.clearToastCmd())

		case "i", "ш": // Inspect deep metadata
			if c := m.getSelectedContainer(); c != nil {
				m.activeView = ViewInspect
				m.recalcDimensions()
				cmds = append(cmds, m.inspectContainerCmd(c.ID))
			} else {
				m.toast = "⚠️ Выберите контейнер для инспекции"
				m.toastIsErr = true
				cmds = append(cmds, m.clearToastCmd())
			}

		case "d", "в": // Delete with confirmation
			if c := m.getSelectedContainer(); c != nil {
				m.modal.Show(c.ID, c.Name, model.ActionRemove)
			} else {
				m.toast = "⚠️ Выберите контейнер для удаления"
				m.toastIsErr = true
				cmds = append(cmds, m.clearToastCmd())
			}

		case "enter", "l", "д": // Open Logs View or toggle stack
			selectedRow := m.getSelectedRow()
			if selectedRow == nil {
				return m, nil
			}
			if selectedRow.Type == views.RowProjectHeader {
				m.listView.ToggleProjectCollapse(selectedRow.ProjectName)
				m.recalcDimensions()
				return m, nil
			}
			c := selectedRow.Container
			if c != nil {
				m.activeView = ViewLogs
				m.logsView.SetContainer(*c)
				m.recalcDimensions()
				cmds = append(cmds, m.startLogsStreamCmd(c.ID, ""))
			}

		case "/", "ю": // Search
			m.filterBar.Focus()

		case "esc":
			if m.filterBar.Value() != "" {
				m.filterBar.Reset()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// Отрисовка интерфейса
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Инициализация DockDash..."
	}

	// Если окно терминала слишком сильно сжали
	if m.width < 50 || m.height < 10 {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true).Render("⚠️  Окно терминала слишком маленькое. Пожалуйста, растяните его."),
		)
	}

	// 1. Верхняя панель (шапка)
	title := styles.TitleStyle.Render("⚡ DOCKDASH")
	modeBadge := styles.LiveBadge.Render("● DOCKER CONNECTED")
	if m.docker.IsDemo() {
		modeBadge = styles.DemoBadge.Render("⚠️  DEMO MODE (Daemon Offline)")
	}

	// Статистика Docker-демона
	daemonStats := ""
	if m.daemonInfo.TotalContainers > 0 {
		statsStr := fmt.Sprintf(
			"v%s • %d running, %d stopped • %d CPUs",
			m.daemonInfo.ServerVersion,
			m.daemonInfo.RunningCount,
			m.daemonInfo.StoppedCount,
			m.daemonInfo.CPUs,
		)
		daemonStats = lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(statsStr)
	}

	sortBadge := ""
	if m.sortMode != SortDefault {
		sortNames := []string{"Default", "Name (A-Z)", "Status", "CPU%", "RAM"}
		sortBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1E1E2E")).
			Background(styles.ColorSecondary).
			Padding(0, 1).
			Render("SORT: " + sortNames[m.sortMode])
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		title,
		modeBadge,
		"  ",
		daemonStats,
	)
	if sortBadge != "" {
		header = lipgloss.JoinHorizontal(
			lipgloss.Center,
			header,
			"  ",
			sortBadge,
		)
	}

	// Всплывающее уведомление (статус операции)
	if m.toast != "" {
		icon := "✔ "
		toastStyle := styles.SuccessToast
		if m.toastIsErr {
			icon = "✖ "
			toastStyle = styles.ErrorToast
		}
		header = lipgloss.JoinHorizontal(
			lipgloss.Center,
			header,
			"   ",
			toastStyle.Render(icon+m.toast),
		)
	}

	// 2. Основная рабочая область
	var mainContent string
	if m.activeView == ViewLogs {
		mainContent = m.logsView.Render(true)
	} else if m.activeView == ViewInspect {
		mainContent = m.inspectView.Render(true)
	} else {
		filterView := m.filterBar.Render(m.width)
		visible := m.getVisibleContainers()
		listView := m.listView.Render(visible, true)
		if filterView != "" {
			mainContent = lipgloss.JoinVertical(lipgloss.Left, filterView, listView)
		} else {
			mainContent = listView
		}
	}

	// 3. Нижняя панель подсказок
	isFiltered := m.filterBar.Value() != ""
	activeModeStr := "list"
	if m.activeView == ViewLogs {
		activeModeStr = "logs"
	} else if m.activeView == ViewInspect {
		activeModeStr = "inspect"
	}

	helpContent := m.help.Render(activeModeStr, isFiltered, m.modal.Active, false)

	layout := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		mainContent,
		"",
		helpContent,
	)

	// 4. Поверх экрана рисуем модалку, если запрошено удаление
	if m.modal.Active {
		modalOverlay := m.modal.Render(m.width, m.height)
		return modalOverlay
	}

	return layout
}

// Вспомогательные функции и фоновые команды

func (m *Model) recalcDimensions() {
	headerFooterHeight := 6
	if m.filterBar.Active || m.filterBar.Value() != "" {
		headerFooterHeight += 3
	}

	availHeight := m.height - headerFooterHeight
	if availHeight < 5 {
		availHeight = 5
	}

	m.listView.Width = m.width
	m.listView.Height = availHeight
	m.logsView.Resize(m.width, availHeight)
	m.inspectView.Resize(m.width, availHeight)
}

func (m *Model) getVisibleContainers() []model.ContainerInfo {
	filtered := m.filterBar.FilterContainers(m.containers)
	switch m.sortMode {
	case SortName:
		sort.SliceStable(filtered, func(i, j int) bool {
			return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
		})
	case SortStatus:
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Status == model.StatusRunning && filtered[j].Status != model.StatusRunning
		})
	case SortCPU:
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].CPUPercent > filtered[j].CPUPercent
		})
	case SortMemory:
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].MemoryUsage > filtered[j].MemoryUsage
		})
	}
	return filtered
}

func (m *Model) getNavigableRows() []views.ListRow {
	return m.listView.BuildVisibleRows(m.getVisibleContainers())
}

func (m *Model) getSelectedRow() *views.ListRow {
	return m.listView.GetSelectedRow(m.getVisibleContainers())
}

func (m *Model) getSelectedContainer() *model.ContainerInfo {
	row := m.getSelectedRow()
	if row == nil || row.Type != views.RowContainer {
		return nil
	}
	return row.Container
}

func (m *Model) markLoading(id, msg string) {
	for i := range m.containers {
		if m.containers[i].ID == id || m.containers[i].ShortID == id {
			m.containers[i].IsLoading = true
			m.containers[i].LoadingMsg = msg
			break
		}
	}
}

func (m *Model) loadContainersCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		containers, err := m.docker.ListContainers(ctx)
		if err != nil {
			return actionResultMsg{Action: "list", Err: err}
		}
		return containersMsg(containers)
	}
}

func (m *Model) loadDaemonInfoCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		info, err := m.docker.DaemonInfo(ctx)
		if err != nil {
			return nil
		}
		return daemonInfoMsg(info)
	}
}

func (m *Model) inspectContainerCmd(id string) tea.Cmd {
	var liveThrottling float64
	var liveOOMRisk bool
	for _, c := range m.containers {
		if c.ID == id || c.ShortID == id {
			liveThrottling = c.CPUThrottledPercent
			liveOOMRisk = c.OOMRisk
			break
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		details, err := m.docker.InspectContainer(ctx, id)
		if err != nil {
			return actionResultMsg{Action: "inspect", Err: err}
		}
		if liveThrottling > 0 {
			details.CPUThrottledPercent = liveThrottling
		}
		if liveOOMRisk {
			details.OOMRisk = liveOOMRisk
		}
		return inspectDetailsMsg(details)
	}
}

func (m *Model) executeActionCmd(id, name string, action model.ActionType) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var err error
		switch action {
		case model.ActionStart:
			err = m.docker.StartContainer(ctx, id)
		case model.ActionStop:
			err = m.docker.StopContainer(ctx, id)
		case model.ActionRestart:
			err = m.docker.RestartContainer(ctx, id)
		case model.ActionKill:
			err = m.docker.KillContainer(ctx, id)
		case model.ActionRemove:
			err = m.docker.RemoveContainer(ctx, id, true)
		}

		return actionResultMsg{
			ContainerID:   id,
			ContainerName: name,
			Action:        action,
			Err:           err,
		}
	}
}

func (m *Model) startLogsStreamCmd(id string, since string) tea.Cmd {
	m.stopLogsStream()
	ctx, cancel := context.WithCancel(context.Background())
	m.logsCancel = cancel
	m.logChan = make(chan string, 100)

	tail := 150
	if since != "" {
		tail = 1000
	}

	go func() {
		reader, err := m.docker.GetLogsStream(ctx, id, tail, since)
		if err != nil {
			select {
			case <-ctx.Done():
			case m.logChan <- fmt.Sprintf("[ERROR] Failed to stream logs: %v", err):
			}
			return
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case m.logChan <- scanner.Text():
			}
		}
	}()

	return m.waitForNextLogCmd()
}

func (m *Model) waitForNextLogCmd() tea.Cmd {
	ch := m.logChan
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		line, ok := <-ch
		if !ok {
			return nil
		}
		return logLineMsg(line)
	}
}

func (m *Model) stopLogsStream() {
	if m.logsCancel != nil {
		m.logsCancel()
		m.logsCancel = nil
	}
	m.logChan = nil
}

func (m *Model) pollStatsCmd() tea.Cmd {
	return func() tea.Msg {
		c := m.getSelectedContainer()
		if c == nil || c.Status != model.StatusRunning {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		stats, err := m.docker.GetStats(ctx, c.ID)
		if err != nil {
			return nil
		}
		return statsUpdateMsg(stats)
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) clearToastCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(time.Time) tea.Msg {
		return toastClearMsg{}
	})
}
