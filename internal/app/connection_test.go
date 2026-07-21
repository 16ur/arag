package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/16ur/arag/internal/player"
	"github.com/16ur/arag/internal/webdav"
	"github.com/charmbracelet/x/ansi"
)

func TestConnectionModelStartsWithoutNetworkIO(t *testing.T) {
	t.Parallel()

	factoryCalls := 0
	model := NewConnectionModel(context.Background(), func(context.Context, ConnectionConfig) (Session, error) {
		factoryCalls++
		return Session{}, nil
	}, ConnectionDefaults{Username: "seiz"})
	_ = model.Init()

	view := ansi.Strip(model.View().Content)
	if !strings.Contains(view, "Connect to WebDAV") ||
		!strings.Contains(view, "Server URL") ||
		!strings.Contains(view, "Username") ||
		!strings.Contains(view, "Password") ||
		!strings.Contains(view, "Not connected") {
		t.Fatalf("View() = %q", view)
	}
	if model.connection.inputs[connectionUsernameField].Value() != "seiz" {
		t.Fatal("connection defaults were not applied")
	}
	if factoryCalls != 0 {
		t.Fatalf("View() performed %d connection attempts", factoryCalls)
	}
}

func TestConnectionModelMasksPassword(t *testing.T) {
	t.Parallel()

	model := newTestConnectionModel(nil)
	model.connection.inputs[connectionPasswordField].SetValue("sensitive-password")
	model.connection.focusControl(connectionPasswordField)
	view := ansi.Strip(model.View().Content)
	if strings.Contains(view, "sensitive-password") {
		t.Fatal("View() exposed the WebDAV password")
	}
	if !strings.Contains(view, "••••") {
		t.Fatalf("View() does not contain a password mask: %q", view)
	}
}

func TestConnectionModelKeepsFocusedFieldVisibleOnSmallTerminal(t *testing.T) {
	t.Parallel()

	model := newTestConnectionModel(nil)
	model.connection.inputs[connectionPasswordField].SetValue("secret")
	model.connection.focusControl(connectionPasswordField)
	model.Update(tea.WindowSizeMsg{Width: 24, Height: 8})
	view := model.View().Content
	plainView := ansi.Strip(view)
	if !strings.Contains(plainView, "Password") || strings.Contains(plainView, "secret") {
		t.Fatalf("compact View() = %q", plainView)
	}
	lines := strings.Split(view, "\n")
	if len(lines) != 8 {
		t.Fatalf("View() height = %d, want 8", len(lines))
	}
	for index, line := range lines {
		if width := ansi.StringWidth(line); width != 24 {
			t.Fatalf("line %d width = %d, want 24", index, width)
		}
	}
}

func TestConnectionModelTreatsQAsInputInsideField(t *testing.T) {
	t.Parallel()

	model := newTestConnectionModel(nil)
	model.Update(key("q"))
	if model.confirmQuit {
		t.Fatal("q opened quit confirmation while editing a field")
	}
	if model.connection.inputs[connectionURLField].Value() != "q" {
		t.Fatalf("URL value = %q", model.connection.inputs[connectionURLField].Value())
	}
}

func TestConnectionModelNavigatesFormBeforeSubmitting(t *testing.T) {
	t.Parallel()

	factoryCalls := 0
	model := newTestConnectionModel(func(context.Context, ConnectionConfig) (Session, error) {
		factoryCalls++
		return Session{}, nil
	})
	model.connection.inputs[connectionURLField].SetValue("https://example.com/webdav")
	model.Update(key("enter"))
	if model.connection.focus != connectionUsernameField || factoryCalls != 0 {
		t.Fatal("Enter did not move from URL to username")
	}
	model.Update(key("tab"))
	if model.connection.focus != connectionPasswordField {
		t.Fatal("Tab did not move to password")
	}
	model.Update(key("shift+tab"))
	if model.connection.focus != connectionUsernameField {
		t.Fatal("Shift+Tab did not move to the previous field")
	}
}

func TestConnectionModelConnectsAndTransitionsToBrowser(t *testing.T) {
	t.Parallel()

	reader := &fakeDirectoryReader{}
	videoPlayer := player.Unavailable{}
	var receivedConfig ConnectionConfig
	model := newTestConnectionModel(func(_ context.Context, config ConnectionConfig) (Session, error) {
		receivedConfig = config
		return Session{
			Client: reader,
			Player: videoPlayer,
			Entries: []webdav.Entry{
				{Name: "video.mkv"},
				{Name: "Movies", IsCollection: true},
			},
		}, nil
	})
	form := model.connection
	form.inputs[connectionURLField].SetValue("https://example.com/webdav")
	form.inputs[connectionUsernameField].SetValue("seiz")
	form.inputs[connectionPasswordField].SetValue("secret")
	form.focusControl(connectionSubmitButton)

	_, command := model.Update(key("enter"))
	if command == nil || !model.connecting {
		t.Fatal("Enter did not start the connection attempt")
	}
	if !strings.Contains(ansi.Strip(model.View().Content), "Connecting to WebDAV") {
		t.Fatal("connecting state is not visible")
	}
	model.Update(command())
	if receivedConfig.BaseURL != "https://example.com/webdav" ||
		receivedConfig.Username != "seiz" ||
		receivedConfig.Password != "secret" {
		t.Fatal("session factory did not receive the form values")
	}
	if model.connection != nil || model.connecting || model.client != reader || model.player != videoPlayer {
		t.Fatal("successful connection did not transition to the browser")
	}
	if len(model.entries) != 2 || model.entries[0].Name != "Movies" {
		t.Fatalf("browser entries = %+v", model.entries)
	}
	if form.inputs[connectionPasswordField].Value() != "" {
		t.Fatal("password field was not cleared after connection")
	}
}

func TestConnectionModelKeepsFormAfterAuthenticationFailure(t *testing.T) {
	t.Parallel()

	model := newTestConnectionModel(func(context.Context, ConnectionConfig) (Session, error) {
		return Session{}, webdav.ErrAuthentication
	})
	model.connection.inputs[connectionURLField].SetValue("https://example.com/webdav")
	model.connection.inputs[connectionPasswordField].SetValue("wrong-password")
	model.connection.focusControl(connectionSubmitButton)
	_, command := model.Update(key("enter"))
	model.Update(command())

	if model.connection == nil || model.connecting {
		t.Fatal("authentication failure closed the connection form")
	}
	if model.connection.focus != connectionPasswordField {
		t.Fatalf("focused control = %d, want password field", model.connection.focus)
	}
	view := ansi.Strip(model.View().Content)
	if !strings.Contains(view, "server rejected the credentials") {
		t.Fatalf("View() = %q", view)
	}
	if strings.Contains(view, "wrong-password") {
		t.Fatal("authentication error view exposed the password")
	}
}

func TestConnectionModelValidatesURLBeforeConnecting(t *testing.T) {
	t.Parallel()

	factoryCalls := 0
	model := newTestConnectionModel(func(context.Context, ConnectionConfig) (Session, error) {
		factoryCalls++
		return Session{}, nil
	})
	model.connection.focusControl(connectionSubmitButton)
	_, command := model.Update(key("enter"))
	if command != nil || model.connecting || factoryCalls != 0 {
		t.Fatal("empty URL started a connection attempt")
	}
	if !strings.Contains(ansi.Strip(model.View().Content), "WebDAV URL is required") {
		t.Fatal("missing URL error is not visible")
	}
}

func TestConnectionModelCancelsAttemptWithEscape(t *testing.T) {
	t.Parallel()

	model := newTestConnectionModel(func(ctx context.Context, _ ConnectionConfig) (Session, error) {
		<-ctx.Done()
		return Session{}, ctx.Err()
	})
	model.connection.inputs[connectionURLField].SetValue("https://example.com/webdav")
	model.connection.focusControl(connectionSubmitButton)
	_, command := model.Update(key("enter"))
	if command == nil {
		t.Fatal("connection command is nil")
	}
	model.Update(key("esc"))
	if model.connecting {
		t.Fatal("Escape did not cancel the connection attempt")
	}
	model.Update(command())
	if model.connection.err != nil {
		t.Fatalf("stale canceled attempt changed the form error: %v", model.connection.err)
	}
}

func newTestConnectionModel(factory SessionFactory) *Model {
	if factory == nil {
		factory = func(context.Context, ConnectionConfig) (Session, error) {
			return Session{}, errors.New("unexpected connection attempt")
		}
	}
	model := NewConnectionModel(context.Background(), factory, ConnectionDefaults{})
	_ = model.Init()
	return model
}
