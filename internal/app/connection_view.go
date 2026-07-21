package app

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const fullConnectionFormHeight = 17

func (m *Model) renderConnectionForm(theme viewTheme, width, height int) string {
	formWidth := min(60, max(12, width-4))
	requiredHeight := fullConnectionFormHeight
	if m.connection.err != nil {
		requiredHeight += 2
	}
	if height < requiredHeight {
		return m.renderCompactConnectionForm(theme, width, height, formWidth)
	}

	lines := []string{
		fitLine(theme.section.Render("Connect to WebDAV"), formWidth),
		fitLine(theme.muted.Render("Choose a server preset and enter your credentials."), formWidth),
		fitLine("", formWidth),
	}
	lines = append(lines, m.renderPresetControl(theme, formWidth)...)
	if m.connection.preset == customWebDAVPreset {
		lines = append(lines, m.renderInputControl(theme, "Server URL", connectionURLControl, connectionURLField, formWidth)...)
	}
	lines = append(lines, m.renderInputControl(theme, "Username", connectionUsernameControl, connectionUsernameField, formWidth)...)
	if m.connection.preset == seedhostPreset {
		lines = append(lines, m.renderSeedhostURL(theme, formWidth)...)
	}
	lines = append(lines, m.renderInputControl(theme, "Password", connectionPasswordControl, connectionPasswordField, formWidth)...)
	lines = append(lines, fitLine("", formWidth), m.renderConnectButton(theme, formWidth))
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

func (m *Model) renderPresetControl(theme viewTheme, width int) []string {
	marker := "  "
	labelStyle := theme.label
	if m.connection.focus == connectionPresetControl {
		marker = "> "
		labelStyle = theme.section
	}
	seedhostMarker := "○"
	customMarker := "○"
	seedhostStyle := theme.muted
	customStyle := theme.muted
	if m.connection.preset == seedhostPreset {
		seedhostMarker = "●"
		seedhostStyle = theme.value
	} else {
		customMarker = "●"
		customStyle = theme.value
	}
	options := seedhostStyle.Render(seedhostMarker+" Seedhost") +
		theme.separator.Render("   ") +
		customStyle.Render(customMarker+" Custom WebDAV")
	return []string{
		fitLine(marker+labelStyle.Render("Server"), width),
		fitLine("  "+options, width),
		connectionSeparator(theme, width),
	}
}

func (m *Model) renderInputControl(theme viewTheme, label string, control, input, width int) []string {
	marker := "  "
	labelStyle := theme.label
	if m.connection.focus == control {
		marker = "> "
		labelStyle = theme.section
	}
	return []string{
		fitLine(marker+labelStyle.Render(label), width),
		fitLine("  "+m.connection.inputs[input].View(), width),
		connectionSeparator(theme, width),
	}
}

func (m *Model) renderSeedhostURL(theme viewTheme, width int) []string {
	return []string{
		fitLine("  "+theme.label.Render("Server URL · generated"), width),
		fitLine("  "+theme.value.Render(m.connection.seedhostURL), width),
		connectionSeparator(theme, width),
	}
}

func (m *Model) renderConnectButton(theme viewTheme, width int) string {
	button := fitLine("  Connect", 16)
	if m.connection.focus == connectionSubmitButton {
		button = theme.selected.Width(16).Render(fitLine("> Connect", 16))
	} else {
		button = theme.primary.Render(button)
	}
	return fitLine(button, width)
}

func connectionSeparator(theme viewTheme, width int) string {
	return fitLine("  "+theme.separator.Render(strings.Repeat("─", max(1, width-2))), width)
}

func (m *Model) renderCompactConnectionForm(theme viewTheme, width, height, formWidth int) string {
	lines := []string{fitLine(theme.section.Render("Connect to WebDAV"), formWidth)}
	if height >= 6 {
		lines = append(lines,
			fitLine(theme.muted.Render("Enter your server details."), formWidth),
			fitLine("", formWidth),
		)
	}

	switch m.connection.focus {
	case connectionPresetControl:
		lines = append(lines, m.renderPresetControl(theme, formWidth)...)
	case connectionURLControl:
		lines = append(lines, m.renderInputControl(theme, "Server URL", connectionURLControl, connectionURLField, formWidth)...)
	case connectionUsernameControl:
		lines = append(lines, m.renderInputControl(theme, "Username", connectionUsernameControl, connectionUsernameField, formWidth)...)
	case connectionPasswordControl:
		lines = append(lines, m.renderInputControl(theme, "Password", connectionPasswordControl, connectionPasswordField, formWidth)...)
	case connectionSubmitButton:
		lines = append(lines, m.renderConnectButton(theme, formWidth))
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
