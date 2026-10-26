package views

import (
	"fmt"
	"strings"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
)

// Модальное окно для выбора и переключения Docker Context
type ContextModal struct {
	Active   bool
	Contexts []model.DockerContext
	Cursor   int
}

func NewContextModal() ContextModal {
	return ContextModal{
		Active: false,
		Cursor: 0,
	}
}

func (m *ContextModal) Show(contexts []model.DockerContext) {
	m.Contexts = contexts
	m.Active = true
	m.Cursor = 0
	// Устанавливаем курсор на текущий контекст
	for i, c := range contexts {
		if c.IsCurrent {
			m.Cursor = i
			break
		}
	}
}

func (m *ContextModal) Hide() {
	m.Active = false
}

func (m *ContextModal) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *ContextModal) MoveDown() {
	if m.Cursor < len(m.Contexts)-1 {
		m.Cursor++
	}
}

func (m *ContextModal) GetSelected() *model.DockerContext {
	if len(m.Contexts) == 0 || m.Cursor < 0 || m.Cursor >= len(m.Contexts) {
		return nil
	}
	return &m.Contexts[m.Cursor]
}

func (m ContextModal) Render(width, height int) string {
	if !m.Active {
		return ""
	}

	boxWidth := 68
	if boxWidth > width-4 {
		boxWidth = width - 4
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorPrimary).
		Render("🌐 ВЫБОР DOCKER CONTEXT (MULTI-HOST)")

	hint := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Render("Используйте ↑/↓ для выбора • [Enter] для переключения • [Esc] отмена")

	var rows []string
	for i, c := range m.Contexts {
		isSelected := (i == m.Cursor)
		pointer := "  "
		if isSelected {
			pointer = "❯ "
		}

		currentBadge := ""
		if c.IsCurrent {
			currentBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#11111B")).
				Background(styles.ColorSuccess).
				Padding(0, 1).
				Render("CURRENT")
		}

		nameStr := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorSecondary).Render(c.Name)
		endpointStr := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render("(" + c.Endpoint + ")")

		line := fmt.Sprintf("%s%-14s %s  %s", pointer, nameStr, endpointStr, currentBadge)

		if isSelected {
			line = lipgloss.NewStyle().
				Background(styles.ColorHighlight).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Render(line)
		}
		rows = append(rows, line)
	}

	boxContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		strings.Join(rows, "\n"),
		"",
		hint,
	)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(1, 2).
		Width(boxWidth).
		Render(boxContent)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
	)
}
