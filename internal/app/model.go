// Package app implements the arag terminal user interface.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/16ur/arag/internal/player"
	"github.com/16ur/arag/internal/webdav"
)

// DirectoryReader lists the contents of a WebDAV directory.
type DirectoryReader interface {
	ReadDir(context.Context, *url.URL) ([]webdav.Entry, error)
}

// Model stores the state of the arag terminal interface.
type Model struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	requestCancel       context.CancelFunc
	requestID           uint64
	connectionCancel    context.CancelFunc
	connectionAttemptID uint64
	client              DirectoryReader
	player              player.Player
	sessionFactory      SessionFactory
	connection          *connectionForm
	connecting          bool
	currentDirectory    *url.URL
	history             []navigationFrame
	entries             []webdav.Entry
	selected            int
	targetSelection     int
	loading             bool
	showDetails         bool
	confirmQuit         bool
	pendingOpen         *webdav.Entry
	opening             bool
	notice              string
	noticeIsError       bool
	err                 error
	width               int
	height              int
	darkBackground      bool
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

type videoOpenedMsg struct{}

type videoOpenFailedMsg struct {
	err error
}

// NewModel creates a model that loads the configured WebDAV root.
func NewModel(ctx context.Context, client DirectoryReader, videoPlayer player.Player) *Model {
	model := newBaseModel(ctx)
	model.client = client
	model.player = videoPlayer
	model.loading = true
	return model
}

// NewConnectionModel creates a model that starts with a WebDAV connection
// form and transitions to the browser after authentication succeeds.
func NewConnectionModel(
	ctx context.Context,
	factory SessionFactory,
	defaults ConnectionDefaults,
) *Model {
	model := newBaseModel(ctx)
	model.sessionFactory = factory
	model.connection = newConnectionForm(defaults, model.darkBackground)
	return model
}

func newBaseModel(ctx context.Context) *Model {
	if ctx == nil {
		ctx = context.Background()
	}
	modelContext, cancel := context.WithCancel(ctx)
	return &Model{
		ctx:            modelContext,
		cancel:         cancel,
		darkBackground: true,
	}
}

// Init detects the terminal theme and starts loading the WebDAV root outside
// the rendering path.
func (m *Model) Init() tea.Cmd {
	if m.connection != nil {
		return tea.Batch(m.connection.init(), tea.RequestBackgroundColor)
	}
	return tea.Batch(m.startLoad(nil, 0), tea.RequestBackgroundColor)
}

// Update handles WebDAV results, terminal resizing, and keyboard input.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sessionConnectedMsg:
		if m.connection == nil || msg.attemptID != m.connectionAttemptID {
			return m, nil
		}
		m.connection.clearPassword()
		if m.connectionCancel != nil {
			m.connectionCancel()
		}
		m.connection = nil
		m.sessionFactory = nil
		m.connectionCancel = nil
		m.connecting = false
		m.client = msg.session.Client
		m.player = msg.session.Player
		m.entries = sortedEntries(msg.session.Entries)
		m.selected = 0
		m.currentDirectory = nil
		m.history = nil
		m.loading = false
		m.err = nil
		m.clearNotice()
	case connectionFailedMsg:
		if m.connection == nil || msg.attemptID != m.connectionAttemptID {
			return m, nil
		}
		if m.connectionCancel != nil {
			m.connectionCancel()
			m.connectionCancel = nil
		}
		m.connecting = false
		m.connection.err = msg.err
		focus := connectionURLField
		if errors.Is(msg.err, webdav.ErrAuthentication) {
			focus = connectionPasswordField
		}
		return m, m.connection.focusControl(focus)
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
	case videoOpenedMsg:
		m.opening = false
		m.notice = "Video sent to the player."
		m.noticeIsError = false
	case videoOpenFailedMsg:
		m.opening = false
		m.notice = playerError(msg.err)
		m.noticeIsError = true
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.connection != nil {
			m.connection.resize(msg.Width)
		}
	case tea.BackgroundColorMsg:
		m.darkBackground = msg.IsDark()
		if m.connection != nil {
			m.connection.applyTheme(newViewTheme(m.darkBackground))
		}
	case tea.KeyPressMsg:
		if m.connection != nil {
			return m.handleConnectionKey(msg)
		}
		return m.handleKey(msg)
	}
	if m.connection != nil {
		return m, m.connection.update(msg)
	}
	return m, nil
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
	m.showDetails = false
	m.pendingOpen = nil
	m.opening = false
	m.clearNotice()
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
	keystroke := msg.Keystroke()
	if keystroke == "ctrl+c" {
		m.cancel()
		return m, tea.Quit
	}
	if m.confirmQuit {
		switch keystroke {
		case "enter":
			m.cancel()
			return m, tea.Quit
		case "esc":
			m.confirmQuit = false
		}
		return m, nil
	}
	if keystroke == "q" {
		m.confirmQuit = true
		return m, nil
	}
	if m.pendingOpen != nil {
		switch keystroke {
		case "enter":
			entry := cloneEntry(m.pendingOpen)
			m.pendingOpen = nil
			return m, m.openVideo(entry)
		case "esc":
			m.pendingOpen = nil
		}
		return m, nil
	}
	if m.opening {
		return m, nil
	}
	if m.showDetails {
		if keystroke == "i" || keystroke == "esc" {
			m.showDetails = false
		}
		return m, nil
	}

	switch keystroke {
	case "up", "k":
		if !m.loading && m.err == nil && m.selected > 0 {
			m.selected--
			m.clearNotice()
		}
	case "down", "j":
		if !m.loading && m.err == nil && m.selected < len(m.entries)-1 {
			m.selected++
			m.clearNotice()
		}
	case "enter", "l":
		if !m.loading && m.err == nil && len(m.entries) > 0 {
			return m, m.activateSelected()
		}
	case "left", "h", "backspace":
		return m.goBack()
	case "i":
		if !m.loading && m.err == nil && len(m.entries) > 0 {
			m.showDetails = true
		}
	case "r":
		if m.err != nil {
			return m, m.startLoad(m.currentDirectory, m.targetSelection)
		}
	}
	return m, nil
}

func (m *Model) activateSelected() tea.Cmd {
	entry := m.entries[m.selected]
	if !entry.IsCollection {
		if isVideoFile(entry.Name) {
			m.pendingOpen = cloneEntry(&entry)
			m.clearNotice()
		} else {
			m.notice = "Unsupported file type. Only MKV and MP4 videos can be opened."
			m.noticeIsError = true
		}
		return nil
	}
	if entry.URL == nil {
		return nil
	}
	m.history = append(m.history, navigationFrame{
		directory: cloneURL(m.currentDirectory),
		selected:  m.selected,
	})
	m.currentDirectory = cloneURL(entry.URL)
	return m.startLoad(m.currentDirectory, 0)
}

func (m *Model) openVideo(entry *webdav.Entry) tea.Cmd {
	m.opening = true
	m.clearNotice()
	mediaURL := cloneURL(entry.URL)
	return func() tea.Msg {
		if mediaURL == nil {
			return videoOpenFailedMsg{err: errors.New("video URL is unavailable")}
		}
		if m.player == nil {
			return videoOpenFailedMsg{err: player.ErrUnavailable}
		}
		if err := m.player.Open(m.ctx, mediaURL); err != nil {
			return videoOpenFailedMsg{err: err}
		}
		return videoOpenedMsg{}
	}
}

func (m *Model) clearNotice() {
	m.notice = ""
	m.noticeIsError = false
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

func cloneEntry(value *webdav.Entry) *webdav.Entry {
	if value == nil {
		return nil
	}
	copy := *value
	copy.URL = cloneURL(value.URL)
	return &copy
}

func isVideoFile(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".mkv", ".mp4":
		return true
	default:
		return false
	}
}

func displayText(value string) string {
	return strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return '�'
		}
		return character
	}, value)
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

func playerError(err error) string {
	if errors.Is(err, player.ErrUnavailable) {
		return "Player integration is not available yet."
	}
	return "Could not open video: " + displayText(err.Error())
}
