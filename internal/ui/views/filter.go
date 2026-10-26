package views

import (
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Строка быстрого поиска контейнеров [/]
type FilterBar struct {
	Input  textinput.Model
	Active bool
}

func NewFilterBar() FilterBar {
	ti := textinput.New()
	ti.Placeholder = "Введите имя или образ для фильтрации... (Esc — сброс)"
	ti.Prompt = "🔍 /"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.ColorSecondary).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(styles.ColorText)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.ColorMuted)
	ti.CharLimit = 64
	return FilterBar{
		Input:  ti,
		Active: false,
	}
}

func (f *FilterBar) Focus() {
	f.Active = true
	f.Input.Focus()
}

func (f *FilterBar) Blur() {
	f.Active = false
	f.Input.Blur()
}

func (f *FilterBar) Reset() {
	f.Input.Reset()
	f.Blur()
}

func (f *FilterBar) Value() string {
	return strings.TrimSpace(f.Input.Value())
}

// Фильтрация списка по имени, образу или ID
func (f *FilterBar) FilterContainers(containers []model.ContainerInfo) []model.ContainerInfo {
	val := strings.ToLower(f.Value())
	if val == "" {
		return containers
	}

	filtered := make([]model.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		if strings.Contains(strings.ToLower(c.Name), val) ||
			strings.Contains(strings.ToLower(c.Image), val) ||
			strings.Contains(strings.ToLower(c.ComposeProject), val) ||
			strings.Contains(strings.ToLower(c.ComposeService), val) ||
			strings.Contains(strings.ToLower(c.ShortID), val) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (f *FilterBar) Render(width int) string {
	if !f.Active && f.Value() == "" {
		return ""
	}

	availWidth := width - 6
	if availWidth < 30 {
		availWidth = 30
	}
	f.Input.Width = availWidth - 6

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorSecondary).
		Padding(0, 1).
		Width(availWidth)

	return style.Render(f.Input.View())
}
