package views

import (
	"fmt"

	"dockdash/internal/model"
	"dockdash/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
)

// Окно подтверждения опасных действий (Удаление, Force Kill)
type ConfirmationModal struct {
	Active        bool
	ContainerID   string
	ContainerName string
	Action        model.ActionType
	Prompt        string
}

func (m *ConfirmationModal) Show(id, name string, action model.ActionType) {
	m.Active = true
	m.ContainerID = id
	m.ContainerName = name
	m.Action = action
	actionName := "УДАЛИТЬ"
	if action == model.ActionKill {
		actionName = "ПРИНУДИТЕЛЬНО ЗАВЕРШИТЬ (SIGKILL)"
	}
	m.Prompt = fmt.Sprintf("Вы уверены, что хотите %s контейнер '%s'?", actionName, name)
}

func (m *ConfirmationModal) Hide() {
	m.Active = false
	m.ContainerID = ""
	m.ContainerName = ""
}

func (m *ConfirmationModal) Render(width, height int) string {
	if !m.Active {
		return ""
	}

	title := styles.ErrorToast.Render("⚠️  ПОДТВЕРЖДЕНИЕ ДЕЙСТВИЯ")
	message := lipgloss.NewStyle().Foreground(styles.ColorText).Bold(true).Render(m.Prompt)
	warning := lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true).Render("Это действие нельзя отменить.")
	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Background(styles.ColorDanger).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Padding(0, 2).Render("[Y / Да] Подтвердить"),
		"    ",
		lipgloss.NewStyle().Background(styles.ColorHighlight).Foreground(styles.ColorTextDim).Bold(true).Padding(0, 2).Render("[N / Esc] Отмена"),
	)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		message,
		warning,
		"",
		buttons,
	)

	dialog := styles.ModalBox.Width(58).Render(content)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
