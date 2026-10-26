package views

import (
	"strings"

	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
)

// Нижняя плашка подсказок с горячими клавишами
type HelpView struct{}

type KeyBinding struct {
	Key  string
	Desc string
}

func (h HelpView) Render(activeMode string, isFiltered bool, isModal bool, isSplitMode bool) string {
	var bindings []KeyBinding

	if isModal {
		bindings = []KeyBinding{
			{Key: "y", Desc: "Подтвердить"},
			{Key: "n / Esc", Desc: "Отмена"},
		}
	} else if activeMode == "logs" {
		bindings = []KeyBinding{
			{Key: "Esc", Desc: "Назад"},
			{Key: "↑/↓", Desc: "Скролл"},
			{Key: "f", Desc: "Автоскролл"},
			{Key: "t", Desc: "Время"},
			{Key: "c", Desc: "Очистить"},
			{Key: "q", Desc: "Выход"},
		}
	} else if activeMode == "inspect" {
		bindings = []KeyBinding{
			{Key: "↑/↓", Desc: "След/Пред контейнер"},
			{Key: "PgUp/PgDn", Desc: "Скролл"},
			{Key: "Esc", Desc: "Назад"},
			{Key: "q", Desc: "Выход"},
		}
	} else {
		bindings = []KeyBinding{
			{Key: "↑/↓", Desc: "Выбор"},
			{Key: "Space", Desc: "Старт/Стоп"},
			{Key: "x", Desc: "Шелл"},
			{Key: "r", Desc: "Рестарт"},
			{Key: "s", Desc: "Сортировка"},
			{Key: "Enter", Desc: "Логи"},
			{Key: "i", Desc: "Инспекция"},
			{Key: "d", Desc: "Удалить"},
			{Key: "Shift+K", Desc: "Kill"},
			{Key: "/", Desc: "Поиск"},
			{Key: "q", Desc: "Выход"},
		}
		if isSplitMode {
			bindings = append([]KeyBinding{{Key: "Tab", Desc: "Панель"}}, bindings...)
		}
		if isFiltered {
			bindings = append([]KeyBinding{{Key: "Esc", Desc: "Сброс"}}, bindings...)
		}
	}

	var items []string
	for _, b := range bindings {
		keyPill := lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorPrimary).
			Render("[" + b.Key + "]")

		item := lipgloss.JoinHorizontal(
			lipgloss.Left,
			keyPill,
			" ",
			styles.HelpDescStyle.Render(b.Desc),
		)
		items = append(items, item)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		strings.Join(items, styles.HelpSeparator.String()),
	)
}
