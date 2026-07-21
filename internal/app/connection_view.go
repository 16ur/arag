package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m *Model) renderConnectionForm(theme viewTheme, width, height int) string {
	formWidth := min(60, max(12, width-4))
	if height < 14 {
		return m.renderCompactConnectionForm(theme, width, height, formWidth)
	}
	lines := []string{
		fitLine(theme.section.Render("Connect to WebDAV"), formWidth),
		fitLine(theme.muted.Render("Enter the details for your WebDAV server."), formWidth),
		fitLine("", formWidth),
	}
	labels := []string{"Server URL", "Username", "Password"}
	for index, label := range labels {
		marker := "  "
		labelStyle := theme.label
		if m.connection.focus == index {
			marker = "> "
			labelStyle = theme.section
		}
		lines = append(lines,
			fitLine(marker+labelStyle.Render(label), formWidth),
			fitLine("  "+m.connection.inputs[index].View(), formWidth),
			fitLine("  "+theme.separator.Render(strings.Repeat("─", max(1, formWidth-2))), formWidth),
		)
	}
	button := fitLine("  Connect", 16)
	if m.connection.focus == connectionSubmitButton {
		button = theme.selected.Width(16).Render(fitLine("> Connect", 16))
	} else {
		button = theme.primary.Render(button)
	}
	lines = append(lines, fitLine("", formWidth), fitLine(button, formWidth))
	if m.connection.err != nil {
		errorLine := theme.status.Render("!") + " " + theme.value.Render(friendlyError(m.connection.err))
		lines = append(lines, fitLine("", formWidth), fitLine(errorLine, formWidth))
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	content := strings.Join(lines, "\n")
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) renderCompactConnectionForm(theme viewTheme, width, height, formWidth int) string {
	lines := []string{fitLine(theme.section.Render("Connect to WebDAV"), formWidth)}
	if height >= 6 {
		lines = append(lines,
			fitLine(theme.muted.Render("Enter your server details."), formWidth),
			fitLine("", formWidth),
		)
	}
	if m.connection.focus < len(m.connection.inputs) {
		labels := []string{"Server URL", "Username", "Password"}
		lines = append(lines,
			fitLine("> "+theme.section.Render(labels[m.connection.focus]), formWidth),
			fitLine("  "+m.connection.inputs[m.connection.focus].View(), formWidth),
			fitLine("  "+theme.separator.Render(strings.Repeat("─", max(1, formWidth-2))), formWidth),
		)
	} else {
		button := theme.selected.Width(16).Render(fitLine("> Connect", 16))
		lines = append(lines, fitLine(button, formWidth))
	}
	if m.connection.err != nil {
		errorLine := theme.status.Render("!") + " " + theme.value.Render(friendlyError(m.connection.err))
		renderedError := fitLine(errorLine, formWidth)
		if len(lines) >= height {
			lines[height-1] = renderedError
		} else {
			lines = append(lines, renderedError)
		}
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, strings.Join(lines, "\n"))
}
