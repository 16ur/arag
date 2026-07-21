// Package app implements the arag terminal user interface.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/16ur/arag/internal/webdav"
)

const defaultVisibleRows = 10

// DirectoryReader lists the contents of a WebDAV directory.
type DirectoryReader interface {
	ReadDir(context.Context, *url.URL) ([]webdav.Entry, error)
}

// Model stores the state of the arag terminal interface.
type Model struct {
	ctx              context.Context
	cancel           context.CancelFunc
	requestCancel    context.CancelFunc
	requestID        uint64
	client           DirectoryReader
	currentDirectory *url.URL
	history          []navigationFrame
	entries          []webdav.Entry
	selected         int
	targetSelection  int
	loading          bool
	err              error
	width            int
	height           int
}

type entriesLoadedMsg struct {
	requestID uint64
	entries   []webdav.Entry
	selected  int
}

type loadFailedMsg struct {
	requestID uint64
	err       error
}

type navigationFrame struct {
	directory *url.URL
	selected  int
}

// NewModel creates a model that loads the configured WebDAV root.
func NewModel(ctx context.Context, client DirectoryReader) *Model {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Model{
		ctx:     ctx,
		cancel:  cancel,
		client:  client,
		loading: true,
	}
}

// Init starts loading the WebDAV root outside the rendering path.
func (m *Model) Init() tea.Cmd {
	return m.startLoad(nil, 0)
}

// Update handles WebDAV results, terminal resizing, and keyboard input.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case entriesLoadedMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.entries = sortedEntries(msg.entries)
		m.selected = clampSelection(msg.selected, len(m.entries))
		m.loading = false
		m.err = nil
		m.finishRequest()
	case loadFailedMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		m.finishRequest()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// View renders the current state without performing I/O or business logic.
func (m *Model) View() tea.View {
	var content strings.Builder
	content.WriteString("arag\n")
	fmt.Fprintf(&content, "Location: %s\n\n", m.location())

	switch {
	case m.loading:
		content.WriteString("Loading directory...\n\nq quit")
	case m.err != nil:
		fmt.Fprintf(&content, "Error: %s\n\nr retry  •  h/← back  •  q quit", friendlyError(m.err))
	case len(m.entries) == 0:
		content.WriteString("Empty directory.\n\nh/← back  •  q quit")
	default:
		m.renderEntries(&content)
	}

	view := tea.NewView(content.String())
	view.AltScreen = true
	view.WindowTitle = "arag"
	return view
}

func (m *Model) startLoad(directory *url.URL, selected int) tea.Cmd {
	if m.requestCancel != nil {
		m.requestCancel()
	}
	requestContext, cancel := context.WithCancel(m.ctx)
	m.requestCancel = cancel
	m.requestID++
	m.targetSelection = selected
	m.loading = true
	m.err = nil

	requestID := m.requestID
	return func() tea.Msg {
		entries, err := m.client.ReadDir(requestContext, directory)
		if err != nil {
			return loadFailedMsg{requestID: requestID, err: err}
		}
		return entriesLoadedMsg{
			requestID: requestID,
			entries:   entries,
			selected:  selected,
		}
	}
}

func (m *Model) finishRequest() {
	if m.requestCancel != nil {
		m.requestCancel()
		m.requestCancel = nil
	}
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Keystroke() {
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "up", "k":
		if !m.loading && m.err == nil && m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if !m.loading && m.err == nil && m.selected < len(m.entries)-1 {
			m.selected++
		}
	case "enter", "l":
		if !m.loading && m.err == nil && len(m.entries) > 0 {
			return m, m.openSelected()
		}
	case "left", "h", "backspace":
		return m.goBack()
	case "r":
		if m.err != nil {
			return m, m.startLoad(m.currentDirectory, m.targetSelection)
		}
	}
	return m, nil
}

func (m *Model) openSelected() tea.Cmd {
	entry := m.entries[m.selected]
	if !entry.IsCollection || entry.URL == nil {
		return nil
	}
	m.history = append(m.history, navigationFrame{
		directory: cloneURL(m.currentDirectory),
		selected:  m.selected,
	})
	m.currentDirectory = cloneURL(entry.URL)
	return m.startLoad(m.currentDirectory, 0)
}

func (m *Model) goBack() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	last := len(m.history) - 1
	frame := m.history[last]
	m.history = m.history[:last]
	m.currentDirectory = cloneURL(frame.directory)
	return m, m.startLoad(m.currentDirectory, frame.selected)
}

func (m *Model) renderEntries(content *strings.Builder) {
	start, end := visibleRange(m.selected, len(m.entries), m.visibleRows())
	for index := start; index < end; index++ {
		entry := m.entries[index]
		marker := "  "
		if index == m.selected {
			marker = "> "
		}
		kind := "[F]"
		details := formatSize(entry.Size)
		if entry.IsCollection {
			kind = "[D]"
			details = ""
		}
		name := truncate(entry.Name, m.nameWidth())
		fmt.Fprintf(content, "%s%s %-8s %s\n", marker, kind, details, name)
	}
	content.WriteString("\n↑/k up  •  ↓/j down  •  enter/l open  •  h/← back  •  q quit")
}

func (m *Model) location() string {
	if m.currentDirectory == nil {
		return "/"
	}
	if m.currentDirectory.Path == "" {
		return "/"
	}
	return m.currentDirectory.Path
}

func (m *Model) visibleRows() int {
	if m.height <= 0 {
		return defaultVisibleRows
	}
	rows := m.height - 6
	if rows < 1 {
		return 1
	}
	return rows
}

func (m *Model) nameWidth() int {
	if m.width <= 0 {
		return 60
	}
	width := m.width - 18
	if width < 8 {
		return 8
	}
	return width
}

func sortedEntries(entries []webdav.Entry) []webdav.Entry {
	result := append([]webdav.Entry(nil), entries...)
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].IsCollection != result[right].IsCollection {
			return result[left].IsCollection
		}
		return strings.ToLower(result[left].Name) < strings.ToLower(result[right].Name)
	})
	return result
}

func visibleRange(selected, total, limit int) (int, int) {
	if total <= limit {
		return 0, total
	}
	start := selected - limit + 1
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > total {
		end = total
	}
	return start, end
}

func clampSelection(selected, total int) int {
	if total == 0 || selected < 0 {
		return 0
	}
	if selected >= total {
		return total - 1
	}
	return selected
}

func cloneURL(value *url.URL) *url.URL {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func truncate(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	unit := "B"
	for _, candidate := range units {
		value /= 1024
		unit = candidate
		if value < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

func friendlyError(err error) string {
	switch {
	case errors.Is(err, webdav.ErrAuthentication):
		return "the server rejected the credentials"
	case errors.Is(err, webdav.ErrUnexpectedStatus):
		return "the server did not return a valid WebDAV response"
	case errors.Is(err, webdav.ErrInvalidResponse):
		return "the WebDAV XML response is invalid"
	case errors.Is(err, context.DeadlineExceeded):
		return "the server took too long to respond"
	case errors.Is(err, context.Canceled):
		return "operation canceled"
	default:
		return err.Error()
	}
}
