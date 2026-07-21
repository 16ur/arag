package app

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/16ur/arag/internal/webdav"
)

const (
	defaultScreenWidth  = 96
	defaultScreenHeight = 24
	screenChromeHeight  = 4
	splitLayoutMinWidth = 80
	listPaneRatio       = 0.65
	maximumNameWidth    = 60
	minimumNameWidth    = 4
	fileSizeWidth       = 10
	maximumModalWidth   = 64
	minimumFramedHeight = 8
	minimumFramedWidth  = 30
)

// View renders the current state without performing I/O or business logic.
func (m *Model) View() tea.View {
	theme := newViewTheme(m.darkBackground)
	width, height := m.screenSize()
	bodyHeight := height - screenChromeHeight
	if m.notice != "" && m.canShowBrowser() {
		bodyHeight--
	}
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	header := m.renderHeader(theme, width)
	separator := theme.separator.Render(strings.Repeat("─", width))
	body := m.renderBody(theme, width, bodyHeight)
	parts := []string{header, separator, body}
	if m.notice != "" && m.canShowBrowser() {
		parts = append(parts, m.renderNotice(theme, width))
	}
	parts = append(parts, separator, m.renderFooter(theme, width))

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	view.AltScreen = true
	view.WindowTitle = "arag"
	return view
}

func (m *Model) renderHeader(theme viewTheme, width int) string {
	left := theme.brand.Render("arag")
	if m.connection != nil {
		left += theme.muted.Render(" · ") + theme.breadcrumb.Render("Connect")
	} else {
		left += theme.muted.Render(" · ") + theme.breadcrumb.Render(m.location())
	}
	statusText := "● Connected"
	if m.connection != nil {
		statusText = "○ Not connected"
		if m.connecting {
			statusText = "… Connecting"
		}
	} else if m.loading {
		statusText = "… Loading"
	} else if m.err != nil {
		statusText = "! Connection issue"
	}
	status := theme.status.Render(statusText)
	statusWidth := ansi.StringWidth(status)
	if statusWidth >= width {
		return fitLine(status, width)
	}
	left = ansi.Truncate(left, width-statusWidth-1, "…")
	return left + strings.Repeat(" ", width-statusWidth-ansi.StringWidth(left)) + status
}

func (m *Model) renderBody(theme viewTheme, width, height int) string {
	switch {
	case m.confirmQuit:
		return m.renderQuitModal(theme, width, height)
	case m.connection != nil && m.connecting:
		return renderCenteredState(theme, width, height, "Connecting to WebDAV…", "Authenticating and loading the root directory")
	case m.connection != nil:
		return m.renderConnectionForm(theme, width, height)
	case m.pendingOpen != nil:
		return m.renderOpenModal(theme, width, height)
	case m.showDetails:
		return m.renderDetailsModal(theme, width, height)
	case m.loading:
		return renderCenteredState(theme, width, height, "Loading directory…", "Please wait")
	case m.err != nil:
		return renderCenteredState(theme, width, height, "Connection issue", friendlyError(m.err))
	case len(m.entries) == 0:
		return renderCenteredState(theme, width, height, "Empty directory", "There are no entries here")
	case m.opening:
		return renderCenteredState(theme, width, height, "Opening video…", "Preparing the secure local stream")
	default:
		return m.renderBrowser(theme, width, height)
	}
}

func (m *Model) renderBrowser(theme viewTheme, width, height int) string {
	if width < splitLayoutMinWidth {
		return m.renderEntryList(theme, width, height)
	}
	listWidth := int(float64(width-1) * listPaneRatio)
	detailsWidth := width - listWidth - 1
	list := m.renderEntryList(theme, listWidth, height)
	details := m.renderEssentialDetails(theme, detailsWidth, height)
	divider := theme.separator.Render(strings.TrimSuffix(strings.Repeat("│\n", height), "\n"))
	return lipgloss.JoinHorizontal(lipgloss.Top, list, divider, details)
}

func (m *Model) renderEntryList(theme viewTheme, width, height int) string {
	start, end := visibleRange(m.selected, len(m.entries), height)
	lines := make([]string, 0, height)
	for index := start; index < end; index++ {
		entry := m.entries[index]
		marker := "  "
		if index == m.selected {
			marker = "> "
		}
		name := displayText(entry.Name)
		if entry.IsCollection {
			name += "/"
		}
		nameWidth := rowNameWidth(width, entry.IsCollection)
		name = fitLine(ansi.Truncate(name, nameWidth, "…"), nameWidth)
		row := marker + name
		if !entry.IsCollection {
			row += " " + fitLineLeft(formatSize(entry.Size), fileSizeWidth)
		}
		row = fitLine(row, width)
		if index == m.selected {
			lines = append(lines, theme.selected.Width(width).Render(row))
		} else {
			lines = append(lines, theme.primary.Render(row))
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderEssentialDetails(theme viewTheme, width, height int) string {
	entry := m.selectedEntry()
	if entry == nil {
		return strings.TrimSuffix(strings.Repeat(strings.Repeat(" ", width)+"\n", height), "\n")
	}
	contentWidth := max(1, width-2)
	lines := []string{
		fitLine("", width),
		fitLine("  "+theme.section.Render("SELECTED"), width),
		fitLine("  "+theme.value.Bold(true).Render(ansi.Truncate(displayText(entry.Name), contentWidth, "…")), width),
		fitLine("", width),
	}
	lines = appendMetadata(lines, theme, width, "Type", essentialEntryType(*entry))
	if !entry.IsCollection {
		lines = appendMetadata(lines, theme, width, "Size", formatSize(entry.Size))
	}
	modified := "Not available"
	if !entry.ModTime.IsZero() {
		modified = entry.ModTime.Format("Jan 2, 2006 15:04")
	}
	lines = appendMetadata(lines, theme, width, "Modified", modified)
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func appendMetadata(lines []string, theme viewTheme, width int, label, value string) []string {
	lines = append(lines, fitLine("  "+theme.label.Render(label), width))
	return append(lines, fitLine("  "+theme.value.Render(ansi.Truncate(value, max(1, width-2), "…")), width))
}

func (m *Model) renderOpenModal(theme viewTheme, width, height int) string {
	entry := m.pendingOpen
	lines := []string{
		theme.value.Bold(true).Render(displayText(entry.Name)),
		theme.label.Render("Video · ") + theme.value.Render(formatSize(entry.Size)),
	}
	return renderModal(theme, width, height, "Open video?", lines)
}

func (m *Model) renderQuitModal(theme viewTheme, width, height int) string {
	return renderModal(theme, width, height, "Quit arag?", []string{
		theme.value.Render("Your current navigation session will be closed."),
	})
}

func (m *Model) renderDetailsModal(theme viewTheme, width, height int) string {
	entry := m.selectedEntry()
	if entry == nil {
		return renderCenteredState(theme, width, height, "Details", "No entry selected")
	}
	entryType := "File"
	size := formatSize(entry.Size)
	if entry.IsCollection {
		entryType = "Directory"
		size = "Not applicable"
	}
	modified := "Not available"
	if !entry.ModTime.IsZero() {
		modified = entry.ModTime.Format(time.RFC3339)
	}
	etag := displayText(entry.ETag)
	if etag == "" {
		etag = "Not available"
	}
	entryPath := "Not available"
	if entry.URL != nil {
		entryPath = displayText(entry.URL.Path)
	}
	lines := []string{
		metadataLine(theme, "Name", displayText(entry.Name)),
		metadataLine(theme, "Type", entryType),
		metadataLine(theme, "Size", size),
		metadataLine(theme, "Modified", modified),
		metadataLine(theme, "ETag", etag),
		metadataLine(theme, "Path", entryPath),
	}
	return renderModal(theme, width, height, "Details", lines)
}

func metadataLine(theme viewTheme, label, value string) string {
	return theme.label.Width(10).Render(label) + theme.value.Render(value)
}

func renderModal(theme viewTheme, width, height int, title string, lines []string) string {
	content := theme.modalTitle.Render(title) + "\n\n" + strings.Join(lines, "\n")
	if width < minimumFramedWidth || height < minimumFramedHeight {
		return fitBlock(content, width, height)
	}
	modalWidth := min(maximumModalWidth, width-6)
	modal := theme.modal.Width(modalWidth).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

func renderCenteredState(theme viewTheme, width, height int, title, detail string) string {
	content := theme.section.Render(title)
	if detail != "" {
		content += "\n" + theme.muted.Render(detail)
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) renderNotice(theme viewTheme, width int) string {
	prefix := "✓"
	if m.noticeIsError {
		prefix = "!"
	}
	message := theme.status.Render(prefix) + " " + theme.notice.Render(m.notice)
	return fitLine(message, width)
}

func (m *Model) renderFooter(theme viewTheme, width int) string {
	var shortcuts []string
	switch {
	case m.confirmQuit:
		shortcuts = []string{"enter", "confirm", "esc", "cancel"}
	case m.connection != nil && m.connecting:
		shortcuts = []string{"esc", "cancel"}
	case m.connection != nil && m.connection.focus == connectionSubmitButton:
		shortcuts = []string{"enter", "connect", "shift+tab", "previous", "esc", "quit"}
	case m.connection != nil && m.connection.focus == connectionPresetControl:
		shortcuts = []string{"←/→", "change", "enter/tab", "next", "esc", "quit"}
	case m.connection != nil:
		shortcuts = []string{"tab", "next", "shift+tab", "previous", "enter", "next", "esc", "quit"}
	case m.pendingOpen != nil:
		shortcuts = []string{"enter", "open", "esc", "cancel", "q", "quit"}
	case m.showDetails:
		shortcuts = []string{"i/esc", "close", "q", "quit"}
	case m.err != nil:
		shortcuts = []string{"r", "retry", "h/←", "back", "q", "quit"}
	case m.loading || m.opening:
		shortcuts = []string{"q", "quit"}
	case len(m.entries) == 0:
		shortcuts = []string{"h/←", "back", "q", "quit"}
	default:
		switch {
		case width < 32:
			shortcuts = []string{"↑↓", "move", "q", "quit"}
		case width < 64:
			shortcuts = []string{"↑↓/jk", "move", "enter", "open", "q", "quit"}
		default:
			shortcuts = []string{"↑↓/jk", "navigate", "enter/l", "open", "i", "details", "h/←", "back", "q", "quit"}
		}
	}
	if width < 32 {
		switch {
		case m.confirmQuit:
			shortcuts = []string{"enter", "yes", "esc", "no"}
		case m.connection != nil && m.connecting:
			shortcuts = []string{"esc", "cancel"}
		case m.connection != nil && m.connection.focus == connectionSubmitButton:
			shortcuts = []string{"enter", "connect", "esc", "quit"}
		case m.connection != nil && m.connection.focus == connectionPresetControl:
			shortcuts = []string{"←/→", "change", "tab", "next"}
		case m.connection != nil:
			shortcuts = []string{"tab", "next", "esc", "quit"}
		case m.pendingOpen != nil:
			shortcuts = []string{"enter", "open", "esc", "back"}
		case m.showDetails:
			shortcuts = []string{"esc", "close", "q", "quit"}
		case m.err != nil:
			shortcuts = []string{"r", "retry", "q", "quit"}
		}
	}
	items := make([]string, 0, len(shortcuts)/2)
	for index := 0; index < len(shortcuts); index += 2 {
		items = append(items, theme.shortcutKey.Render(shortcuts[index])+" "+theme.muted.Render(shortcuts[index+1]))
	}
	return fitLine(strings.Join(items, theme.separator.Render("   ")), width)
}

func (m *Model) location() string {
	if m.currentDirectory == nil || m.currentDirectory.Path == "" {
		return "/"
	}
	return displayText(m.currentDirectory.Path)
}

func (m *Model) screenSize() (int, int) {
	width := m.width
	if width <= 0 {
		width = defaultScreenWidth
	}
	height := m.height
	if height <= 0 {
		height = defaultScreenHeight
	}
	return max(1, width), max(screenChromeHeight+1, height)
}

func (m *Model) nameWidth() int {
	width, _ := m.screenSize()
	if width >= splitLayoutMinWidth {
		width = int(float64(width-1) * listPaneRatio)
	}
	return min(maximumNameWidth, rowNameWidth(width, false))
}

func (m *Model) selectedEntry() *webdav.Entry {
	if len(m.entries) == 0 || m.selected < 0 || m.selected >= len(m.entries) {
		return nil
	}
	return &m.entries[m.selected]
}

func (m *Model) canShowBrowser() bool {
	return m.connection == nil && !m.loading && m.err == nil && !m.opening && m.pendingOpen == nil && !m.showDetails && !m.confirmQuit && len(m.entries) > 0
}

func essentialEntryType(entry webdav.Entry) string {
	if entry.IsCollection {
		return "Directory"
	}
	if isVideoFile(entry.Name) {
		return "Video"
	}
	return "File"
}

func rowNameWidth(width int, directory bool) int {
	reserved := 2
	if !directory {
		reserved += fileSizeWidth + 1
	}
	return max(minimumNameWidth, min(maximumNameWidth, width-reserved))
}

func fitLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = ansi.Truncate(value, width, "")
	return value + strings.Repeat(" ", width-ansi.StringWidth(value))
}

func fitLineLeft(value string, width int) string {
	value = ansi.Truncate(value, width, "")
	return strings.Repeat(" ", max(0, width-ansi.StringWidth(value))) + value
}

func fitBlock(value string, width, height int) string {
	lines := strings.Split(value, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for index := range lines {
		lines[index] = fitLine(lines[index], width)
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}
