package views

import (
	"fmt"
	"regexp"
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// Опции фильтрации по времени (--since)
type SinceOption int

const (
	SinceAll SinceOption = iota
	Since15m
	Since1h
	Since24h
)

// Окно профессионального просмотра логов контейнера
type LogsView struct {
	Viewport       viewport.Model
	ContainerID    string
	ContainerName  string
	Image          string
	RawLines       []string // Кольцевой буфер (до 5000 строк)
	Follow         bool
	ShowTimestamps bool
	ErrorIsolation bool            // Режим изоляции ошибок (ERROR, FATAL, PANIC, 5xx)
	SearchActive   bool            // Активен ли ввод поискового запроса
	SearchInput    textinput.Model // Поле ввода regex/поиска
	SearchQuery    string
	SearchRegex    *regexp.Regexp
	MatchLines     []int // Номера строк в отображаемом тексте, где найдены совпадения
	CurrentMatch   int   // Индекс текущего совпадения в MatchLines
	SinceMode      SinceOption
	Width          int
	Height         int
}

// Создаем окно логов с поддержкой поиска и фильтров
func NewLogsView() LogsView {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Foreground(styles.ColorText)

	ti := textinput.New()
	ti.Placeholder = "Поиск по логам (regex / текст)... Enter для подтверждения"
	ti.Prompt = "🔎 /"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.ColorWarning).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.ColorText)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.ColorMuted)
	ti.CharLimit = 128

	return LogsView{
		Viewport:       vp,
		SearchInput:    ti,
		Follow:         true,
		ShowTimestamps: true,
		ErrorIsolation: false,
		SinceMode:      SinceAll,
		RawLines:       make([]string, 0, 1000),
		MatchLines:     make([]int, 0),
		CurrentMatch:   -1,
	}
}

// Подключаем окно к новому контейнеру
func (v *LogsView) SetContainer(c model.ContainerInfo) {
	v.ContainerID = c.ID
	v.ContainerName = c.Name
	v.Image = c.Image
	v.RawLines = v.RawLines[:0]
	v.Follow = true
	v.SearchActive = false
	v.SearchQuery = ""
	v.SearchRegex = nil
	v.SearchInput.Reset()
	v.MatchLines = v.MatchLines[:0]
	v.CurrentMatch = -1
	v.Viewport.SetContent(lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("Подключаемся к логам контейнера...\n"))
}

// Добавляем новые строки в лог (кольцевой буфер максимум 5000 строк)
func (v *LogsView) AppendLines(newLines []string) {
	for _, line := range newLines {
		if trimmed := strings.TrimRight(line, "\r\n"); trimmed != "" {
			v.RawLines = append(v.RawLines, trimmed)
		}
	}

	// Ограничиваем буфер 5000 строк, чтобы избежать утечек памяти (OOM)
	maxLines := 5000
	if len(v.RawLines) > maxLines {
		v.RawLines = v.RawLines[len(v.RawLines)-maxLines:]
	}

	v.RebuildContent()

	if v.Follow && !v.SearchActive {
		v.Viewport.GotoBottom()
	}
}

// Проверяет, является ли строка лога ошибкой или предупреждением
func isErrorOrWarnLine(raw string) bool {
	upper := strings.ToUpper(raw)
	return strings.Contains(upper, "ERROR") ||
		strings.Contains(upper, "FATAL") ||
		strings.Contains(upper, "PANIC") ||
		strings.Contains(upper, "EXCEPTION") ||
		strings.Contains(upper, "WARN") ||
		strings.Contains(upper, "WARNING") ||
		strings.Contains(upper, " 500 ") ||
		strings.Contains(upper, " 502 ") ||
		strings.Contains(upper, " 503 ") ||
		strings.Contains(upper, " 504 ")
}

// Переключение изоляции ошибок
func (v *LogsView) ToggleErrorIsolation() bool {
	v.ErrorIsolation = !v.ErrorIsolation
	v.RebuildContent()
	if v.Follow {
		v.Viewport.GotoBottom()
	}
	return v.ErrorIsolation
}

// Переключение интервала времени --since
func (v *LogsView) CycleSince() string {
	v.SinceMode = (v.SinceMode + 1) % 4
	switch v.SinceMode {
	case Since15m:
		return "15m"
	case Since1h:
		return "1h"
	case Since24h:
		return "24h"
	default:
		return ""
	}
}

func (v *LogsView) SinceLabel() string {
	switch v.SinceMode {
	case Since15m:
		return "15m"
	case Since1h:
		return "1h"
	case Since24h:
		return "24h"
	default:
		return "All"
	}
}

// Фокусировка на строке поиска
func (v *LogsView) FocusSearch() {
	v.SearchActive = true
	v.Follow = false // Отключаем автоскролл при поиске, чтобы не сбивать фокус
	v.SearchInput.Focus()
}

// Сброс поиска
func (v *LogsView) ClearSearch() {
	v.SearchActive = false
	v.SearchQuery = ""
	v.SearchRegex = nil
	v.SearchInput.Reset()
	v.SearchInput.Blur()
	v.MatchLines = v.MatchLines[:0]
	v.CurrentMatch = -1
	v.RebuildContent()
}

// Применение поискового запроса (regex или подстрока)
func (v *LogsView) ApplySearch(query string) {
	v.SearchQuery = strings.TrimSpace(query)
	v.SearchActive = false
	v.SearchInput.Blur()

	if v.SearchQuery == "" {
		v.SearchRegex = nil
		v.MatchLines = v.MatchLines[:0]
		v.CurrentMatch = -1
		v.RebuildContent()
		return
	}

	// Пробуем скомпилировать как regex (без учета регистра), иначе экранируем
	re, err := regexp.Compile("(?i)" + v.SearchQuery)
	if err != nil {
		re = regexp.MustCompile("(?i)" + regexp.QuoteMeta(v.SearchQuery))
	}
	v.SearchRegex = re
	v.RebuildContent()

	// Переходим к первому найденному совпадению
	if len(v.MatchLines) > 0 {
		v.CurrentMatch = 0
		v.jumpToMatch(0)
	} else {
		v.CurrentMatch = -1
	}
}

// Переход к следующему совпадению (n)
func (v *LogsView) NextMatch() {
	if len(v.MatchLines) == 0 {
		return
	}
	v.Follow = false
	v.CurrentMatch = (v.CurrentMatch + 1) % len(v.MatchLines)
	v.jumpToMatch(v.CurrentMatch)
}

// Переход к предыдущему совпадению (N)
func (v *LogsView) PrevMatch() {
	if len(v.MatchLines) == 0 {
		return
	}
	v.Follow = false
	v.CurrentMatch = (v.CurrentMatch - 1 + len(v.MatchLines)) % len(v.MatchLines)
	v.jumpToMatch(v.CurrentMatch)
}

func (v *LogsView) jumpToMatch(matchIdx int) {
	if matchIdx < 0 || matchIdx >= len(v.MatchLines) {
		return
	}
	targetLine := v.MatchLines[matchIdx]
	// Центрируем строку в окне просмотра
	offset := targetLine - (v.Viewport.Height / 2)
	if offset < 0 {
		offset = 0
	}
	v.Viewport.SetYOffset(offset)
}

// Включить или выключить показ времени
func (v *LogsView) ToggleTimestamps() {
	v.ShowTimestamps = !v.ShowTimestamps
	v.RebuildContent()
}

// Очистить логи
func (v *LogsView) Clear() {
	v.RawLines = v.RawLines[:0]
	v.MatchLines = v.MatchLines[:0]
	v.CurrentMatch = -1
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
	if v.SearchActive {
		vpHeight -= 3 // Место для строки поиска
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := width - 6
	if vpWidth < 10 {
		vpWidth = 10
	}
	v.Viewport.Width = vpWidth
	v.Viewport.Height = vpHeight
	v.SearchInput.Width = vpWidth - 6
}

// Перерисовываем строки с подсветкой синтаксиса и результатами поиска
func (v *LogsView) RebuildContent() {
	v.MatchLines = v.MatchLines[:0]
	formatted := make([]string, 0, len(v.RawLines))
	lineIdx := 0

	for _, raw := range v.RawLines {
		// Фильтрация в режиме изоляции ошибок
		if v.ErrorIsolation && !isErrorOrWarnLine(raw) {
			continue
		}

		// Проверка на совпадение с поисковым запросом
		matchesSearch := false
		if v.SearchRegex != nil && v.SearchRegex.MatchString(raw) {
			matchesSearch = true
			v.MatchLines = append(v.MatchLines, lineIdx)
		}

		formatted = append(formatted, v.colorizeLogLine(raw, matchesSearch))
		lineIdx++
	}

	if len(formatted) == 0 {
		if v.ErrorIsolation {
			v.Viewport.SetContent(lipgloss.NewStyle().Foreground(styles.ColorSuccess).Italic(true).Render("✔ В логах не обнаружено ошибок или предупреждений."))
		} else {
			v.Viewport.SetContent(lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("Логи отсутствуют."))
		}
		return
	}

	v.Viewport.SetContent(strings.Join(formatted, "\n"))
}

// Подсветка текста: уровни логов (ERROR, WARN, INFO), HTTP коды и поисковые совпадения
func (v *LogsView) colorizeLogLine(raw string, isSearchMatch bool) string {
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

	// Если строка совпала с поиском, подсвечиваем найденные фрагменты
	if isSearchMatch && v.SearchRegex != nil {
		highlightStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#FAB387")).
			Foreground(lipgloss.Color("#11111B")).
			Bold(true)

		line = v.SearchRegex.ReplaceAllStringFunc(line, func(match string) string {
			return highlightStyle.Render(match)
		})
		return line
	}

	upper := strings.ToUpper(line)

	// Подсветка ошибок и предупреждений
	if strings.Contains(upper, "ERROR") || strings.Contains(upper, "FATAL") || strings.Contains(upper, "PANIC") || strings.Contains(upper, "EXCEPTION") {
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
	if strings.Contains(line, " 404 ") || strings.Contains(line, " 500 ") || strings.Contains(line, " 502 ") || strings.Contains(line, " 503 ") {
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

	// Бейдж автоскролла
	followBadge := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[Follow: OFF]")
	if v.Follow {
		followBadge = lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true).Render("[Follow: ON]")
	}

	// Бейдж времени
	timeBadge := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("[Time: OFF]")
	if v.ShowTimestamps {
		timeBadge = lipgloss.NewStyle().Foreground(styles.ColorSecondary).Render("[Time: ON]")
	}

	// Бейдж фильтра --since
	sinceBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89B4FA")).
		Render(fmt.Sprintf("[Since: %s]", v.SinceLabel()))

	// Бейдж изоляции ошибок
	errorBadge := ""
	if v.ErrorIsolation {
		errorBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(styles.ColorDanger).
			Padding(0, 1).
			Render("⚠️ ERRORS ONLY")
	}

	// Бейдж поисковых совпадений
	searchBadge := ""
	if v.SearchQuery != "" {
		if len(v.MatchLines) > 0 {
			searchBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#11111B")).
				Background(lipgloss.Color("#FAB387")).
				Padding(0, 1).
				Render(fmt.Sprintf("FIND (%d/%d): \"%s\"", v.CurrentMatch+1, len(v.MatchLines), v.SearchQuery))
		} else {
			searchBadge = lipgloss.NewStyle().
				Foreground(styles.ColorDanger).
				Render(fmt.Sprintf("FIND (0): \"%s\"", v.SearchQuery))
		}
	}

	linesCount := lipgloss.NewStyle().Foreground(styles.ColorTextDim).Render(fmt.Sprintf("%d lines", len(v.RawLines)))

	var rightBadges []string
	if searchBadge != "" {
		rightBadges = append(rightBadges, searchBadge, "  ")
	}
	if errorBadge != "" {
		rightBadges = append(rightBadges, errorBadge, "  ")
	}
	rightBadges = append(rightBadges, linesCount, "  ", sinceBadge, "  ", timeBadge, "  ", followBadge)

	headerLeft := lipgloss.JoinHorizontal(
		lipgloss.Left,
		styles.SubTitleStyle.Render(fmt.Sprintf("LOGS: %s ", v.ContainerName)),
		lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("(%s)", v.Image)),
	)

	headerRight := lipgloss.JoinHorizontal(
		lipgloss.Right,
		rightBadges...,
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

	var elements []string
	elements = append(elements, header)
	elements = append(elements, lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", availWidth)))

	if v.SearchActive {
		searchBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorWarning).
			Padding(0, 1).
			Width(availWidth - 4).
			Render(v.SearchInput.View())
		elements = append(elements, searchBox)
	}

	elements = append(elements, v.Viewport.View())

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		elements...,
	)

	return panelStyle.Width(availWidth).Height(v.Height).Render(content)
}
