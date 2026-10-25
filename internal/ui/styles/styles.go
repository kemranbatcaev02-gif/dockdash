package styles

import (
	"strings"

	"dockdash/internal/model"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Палитра цветов (в стиле Catppuccin Mocha - мягкие, не режут глаза)
	ColorPrimary   = lipgloss.Color("#CBA6F7") // Нежно-сиреневый
	ColorSecondary = lipgloss.Color("#89DCEB") // Мягкий голубой
	ColorAccent    = lipgloss.Color("#F5C2E7") // Розовый
	ColorSuccess   = lipgloss.Color("#A6E3A1") // Мятно-зеленый
	ColorWarning   = lipgloss.Color("#FAB387") // Персиковый / оранжевый
	ColorDanger    = lipgloss.Color("#F38BA8") // Кораллово-красный
	ColorMuted     = lipgloss.Color("#6C7086") // Серый для неактивных элементов
	ColorBorder    = lipgloss.Color("#45475A") // Цвет рамок
	ColorHighlight = lipgloss.Color("#313244") // Фон выбранной строки
	ColorText      = lipgloss.Color("#CDD6F4") // Основной текст
	ColorTextDim   = lipgloss.Color("#9399B2") // Приглушенный текст

	// Стили шапки и плашек
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#181825")).
			Background(ColorPrimary).
			Padding(0, 1).
			MarginRight(1)

	SubTitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	DemoBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#181825")).
			Background(ColorWarning).
			Padding(0, 1)

	LiveBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#181825")).
			Background(ColorSuccess).
			Padding(0, 1)

	SortBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#181825")).
			Background(ColorSecondary).
			Padding(0, 1)

	// Стили панелей и таблицы
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorBorder).
				MarginBottom(1)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(ColorHighlight).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	NormalRowStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Статусы контейнеров с иконками
	RunningStatus = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true).
			SetString("● RUNNING")

	ExitedStatus = lipgloss.NewStyle().
			Foreground(ColorMuted).
			SetString("○ EXITED")

	PausedStatus = lipgloss.NewStyle().
			Foreground(ColorWarning).
			SetString("⏸ PAUSED")

	RestartingStatus = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true).
				SetString("◌ RESTARTING")

	// Возвращает нужную плашку статуса
	RenderStatus = func(s model.ContainerStatus) string {
		switch s {
		case model.StatusRunning:
			return RunningStatus.String()
		case model.StatusExited:
			return ExitedStatus.String()
		case model.StatusPaused:
			return PausedStatus.String()
		case model.StatusRestarting:
			return RestartingStatus.String()
		default:
			return ExitedStatus.String()
		}
	}

	// Подсказки клавиш внизу экрана
	HelpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	HelpSeparator = lipgloss.NewStyle().
			Foreground(ColorBorder).
			SetString(" • ")

	// Всплывающие уведомления (успех / ошибка)
	SuccessToast = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	ErrorToast = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	// Окно подтверждения опасных действий
	ModalBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorDanger).
			Padding(1, 2).
			Align(lipgloss.Center)
)

// Рисует полоску нагрузки [■■■□□□]
func RenderMiniBar(percent float64, barWidth int, isSelected bool) string {
	if barWidth <= 2 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	fillWidth := barWidth - 2
	filledCount := int((percent / 100.0) * float64(fillWidth))
	if filledCount < 0 {
		filledCount = 0
	}
	if filledCount > fillWidth {
		filledCount = fillWidth
	}
	emptyCount := fillWidth - filledCount

	barColor := ColorSuccess
	if percent >= 80 {
		barColor = ColorDanger
	} else if percent >= 50 {
		barColor = ColorWarning
	}

	filledStr := strings.Repeat("■", filledCount)
	emptyStr := strings.Repeat("□", emptyCount)

	barStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	if isSelected {
		barStyle = barStyle.Background(ColorHighlight)
		emptyStyle = emptyStyle.Background(ColorHighlight)
	}

	return barStyle.Render("["+filledStr) + emptyStyle.Render(emptyStr+"]")
}
