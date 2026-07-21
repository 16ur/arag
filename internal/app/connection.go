package app

import (
	"context"
	"errors"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/16ur/arag/internal/player"
	"github.com/16ur/arag/internal/webdav"
)

const (
	connectionURLField = iota
	connectionUsernameField
	connectionPasswordField
	connectionSubmitButton
	connectionControlCount
)

// ConnectionConfig contains credentials entered for one WebDAV session.
type ConnectionConfig struct {
	BaseURL  string
	Username string
	Password string
}

// ConnectionDefaults pre-fills non-sensitive connection fields.
type ConnectionDefaults struct {
	BaseURL  string
	Username string
}

// Session contains the authenticated dependencies required by the browser.
type Session struct {
	Client  DirectoryReader
	Player  player.Player
	Entries []webdav.Entry
}

// SessionFactory authenticates a WebDAV session and loads its root entries.
type SessionFactory func(context.Context, ConnectionConfig) (Session, error)

type connectionForm struct {
	inputs [3]textinput.Model
	focus  int
	err    error
}

type sessionConnectedMsg struct {
	attemptID uint64
	session   Session
}

type connectionFailedMsg struct {
	attemptID uint64
	err       error
}

func newConnectionForm(defaults ConnectionDefaults, darkBackground bool) *connectionForm {
	form := &connectionForm{}
	form.inputs[connectionURLField] = newConnectionInput("https://example.com/webdav")
	form.inputs[connectionUsernameField] = newConnectionInput("username")
	form.inputs[connectionPasswordField] = newConnectionInput("password")
	form.inputs[connectionPasswordField].EchoMode = textinput.EchoPassword
	form.inputs[connectionPasswordField].EchoCharacter = '•'
	form.inputs[connectionURLField].SetValue(strings.TrimSpace(defaults.BaseURL))
	form.inputs[connectionUsernameField].SetValue(strings.TrimSpace(defaults.Username))
	form.resize(defaultScreenWidth)
	form.applyTheme(newViewTheme(darkBackground))
	return form
}

func newConnectionInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 2048
	return input
}

func (form *connectionForm) init() tea.Cmd {
	return form.focusControl(form.focus)
}

func (form *connectionForm) focusControl(control int) tea.Cmd {
	form.focus = clampSelection(control, connectionControlCount)
	var command tea.Cmd
	for index := range form.inputs {
		if index == form.focus {
			command = form.inputs[index].Focus()
		} else {
			form.inputs[index].Blur()
		}
	}
	return command
}

func (form *connectionForm) focusNext() tea.Cmd {
	return form.focusControl((form.focus + 1) % connectionControlCount)
}

func (form *connectionForm) focusPrevious() tea.Cmd {
	return form.focusControl((form.focus - 1 + connectionControlCount) % connectionControlCount)
}

func (form *connectionForm) update(msg tea.Msg) tea.Cmd {
	if form.focus >= len(form.inputs) {
		return nil
	}
	var command tea.Cmd
	form.inputs[form.focus], command = form.inputs[form.focus].Update(msg)
	return command
}

func (form *connectionForm) config() (ConnectionConfig, error) {
	baseURL := strings.TrimSpace(form.inputs[connectionURLField].Value())
	if baseURL == "" {
		return ConnectionConfig{}, errors.New("WebDAV URL is required")
	}
	return ConnectionConfig{
		BaseURL:  baseURL,
		Username: strings.TrimSpace(form.inputs[connectionUsernameField].Value()),
		Password: form.inputs[connectionPasswordField].Value(),
	}, nil
}

func (form *connectionForm) clearPassword() {
	form.inputs[connectionPasswordField].Reset()
	form.inputs[connectionPasswordField].Blur()
}

func (form *connectionForm) resize(screenWidth int) {
	width := min(56, max(12, screenWidth-8))
	for index := range form.inputs {
		form.inputs[index].SetWidth(width)
	}
}

func (form *connectionForm) applyTheme(theme viewTheme) {
	styles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        theme.value,
			Placeholder: theme.muted,
			Suggestion:  theme.muted,
			Prompt:      theme.status,
		},
		Blurred: textinput.StyleState{
			Text:        theme.value,
			Placeholder: theme.muted,
			Suggestion:  theme.muted,
			Prompt:      theme.muted,
		},
		Cursor: textinput.CursorStyle{
			Color: lipgloss.Color(accentColor),
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
	for index := range form.inputs {
		form.inputs[index].SetStyles(styles)
	}
}

func (m *Model) startConnection() tea.Cmd {
	config, err := m.connection.config()
	if err != nil {
		m.connection.err = err
		return nil
	}
	if m.sessionFactory == nil {
		m.connection.err = errors.New("connection service is unavailable")
		return nil
	}
	if m.connectionCancel != nil {
		m.connectionCancel()
	}
	attemptContext, cancel := context.WithCancel(m.ctx)
	m.connectionCancel = cancel
	m.connectionAttemptID++
	attemptID := m.connectionAttemptID
	m.connecting = true
	m.connection.err = nil
	factory := m.sessionFactory
	return func() tea.Msg {
		session, err := factory(attemptContext, config)
		if err != nil {
			return connectionFailedMsg{attemptID: attemptID, err: err}
		}
		if session.Client == nil || session.Player == nil {
			return connectionFailedMsg{
				attemptID: attemptID,
				err:       errors.New("connection service returned an incomplete session"),
			}
		}
		return sessionConnectedMsg{attemptID: attemptID, session: session}
	}
}

func (m *Model) cancelConnection() {
	if m.connectionCancel != nil {
		m.connectionCancel()
		m.connectionCancel = nil
	}
	m.connectionAttemptID++
	m.connecting = false
}

func (m *Model) handleConnectionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
	if m.connecting {
		if keystroke == "esc" {
			m.cancelConnection()
			return m, m.connection.focusControl(connectionSubmitButton)
		}
		return m, nil
	}

	switch keystroke {
	case "esc":
		m.confirmQuit = true
		return m, nil
	case "tab", "down":
		m.connection.err = nil
		return m, m.connection.focusNext()
	case "shift+tab", "backtab", "up":
		m.connection.err = nil
		return m, m.connection.focusPrevious()
	case "enter":
		m.connection.err = nil
		if m.connection.focus == connectionSubmitButton {
			return m, m.startConnection()
		}
		return m, m.connection.focusNext()
	case "q":
		if m.connection.focus == connectionSubmitButton {
			m.confirmQuit = true
			return m, nil
		}
	}

	m.connection.err = nil
	return m, m.connection.update(msg)
}
