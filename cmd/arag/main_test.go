package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/16ur/arag/internal/app"
	"github.com/16ur/arag/internal/player"
	"github.com/16ur/arag/internal/webdav"
)

type fakeReader struct{}

func (fakeReader) ReadDir(context.Context, *url.URL) ([]webdav.Entry, error) {
	return nil, nil
}

type recordingReader struct {
	entries   []webdav.Entry
	calls     int
	directory *url.URL
}

func (reader *recordingReader) ReadDir(_ context.Context, directory *url.URL) ([]webdav.Entry, error) {
	reader.calls++
	reader.directory = directory
	return reader.entries, nil
}

func TestRunBuildsClientAndStartsInterface(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	var receivedConfig webdav.Config
	passwordReads := 0
	interfaceStarted := false
	var playerUsername string
	var playerPassword string
	videoPlayer := player.Unavailable{}

	err := run(
		context.Background(),
		[]string{"-url", "https://example.com/webdav", "-user", "seiz", "-timeout", "5s"},
		0,
		&bytes.Buffer{},
		&stderr,
		func(string) string { return "" },
		func(uintptr) ([]byte, error) {
			passwordReads++
			return []byte("secret"), nil
		},
		func(config webdav.Config) (directoryReader, error) {
			receivedConfig = config
			return fakeReader{}, nil
		},
		func(username, password string) player.Player {
			playerUsername = username
			playerPassword = password
			return videoPlayer
		},
		func(_ context.Context, model *app.Model, _ uintptr, _ io.Writer) error {
			interfaceStarted = true
			if model == nil {
				t.Error("interface model is nil")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if passwordReads != 1 {
		t.Errorf("password reads = %d, want 1", passwordReads)
	}
	if receivedConfig.BaseURL != "https://example.com/webdav" ||
		receivedConfig.Username != "seiz" ||
		receivedConfig.Password != "secret" ||
		receivedConfig.RequestTimeout != 5*time.Second {
		t.Error("client factory did not receive the expected configuration")
	}
	if !interfaceStarted {
		t.Error("interface was not started")
	}
	if playerUsername != "seiz" || playerPassword != "secret" {
		t.Error("player factory did not receive the WebDAV credentials")
	}
	if !strings.Contains(stderr.String(), "WebDAV password") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestRunUsesEnvironmentPassword(t *testing.T) {
	t.Parallel()

	var receivedPassword string
	err := run(
		context.Background(),
		[]string{"-url", "https://example.com/webdav", "-user", "seiz"},
		0,
		&bytes.Buffer{},
		&bytes.Buffer{},
		func(name string) string {
			if name == passwordEnvironmentVariable {
				return "from-environment"
			}
			return ""
		},
		func(uintptr) ([]byte, error) {
			t.Fatal("password reader must not be called")
			return nil, nil
		},
		func(config webdav.Config) (directoryReader, error) {
			receivedPassword = config.Password
			return fakeReader{}, nil
		},
		successfulPlayerFactory,
		successfulInterface,
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if receivedPassword != "from-environment" {
		t.Error("client factory did not receive the environment password")
	}
}

func TestSessionFactoryAuthenticatesAndLoadsRoot(t *testing.T) {
	t.Parallel()

	reader := &recordingReader{entries: []webdav.Entry{{Name: "Movies", IsCollection: true}}}
	var receivedConfig webdav.Config
	var playerUsername string
	var playerPassword string
	factory := newSessionFactory(
		5*time.Second,
		func(config webdav.Config) (directoryReader, error) {
			receivedConfig = config
			return reader, nil
		},
		func(username, password string) player.Player {
			playerUsername = username
			playerPassword = password
			return player.Unavailable{}
		},
	)
	session, err := factory(context.Background(), app.ConnectionConfig{
		BaseURL:  "https://example.com/webdav",
		Username: "seiz",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("factory() error = %v", err)
	}
	if receivedConfig.BaseURL != "https://example.com/webdav" ||
		receivedConfig.Username != "seiz" ||
		receivedConfig.Password != "secret" ||
		receivedConfig.RequestTimeout != 5*time.Second {
		t.Error("client factory did not receive the expected connection form values")
	}
	if reader.calls != 1 || reader.directory != nil {
		t.Fatalf("root reads = %d, directory = %v", reader.calls, reader.directory)
	}
	if playerUsername != "seiz" || playerPassword != "secret" {
		t.Error("player factory did not receive the WebDAV credentials")
	}
	if session.Client != reader || len(session.Entries) != 1 || session.Entries[0].Name != "Movies" {
		t.Fatal("session does not contain the authenticated root data")
	}
}

func TestRunStartsConnectionScreenWithoutURL(t *testing.T) {
	t.Parallel()

	interfaceStarted := false
	err := run(
		context.Background(), nil, 0, &bytes.Buffer{}, &bytes.Buffer{},
		func(string) string { return "" },
		func(uintptr) ([]byte, error) {
			t.Fatal("password reader must not be called before the connection form")
			return nil, nil
		},
		func(webdav.Config) (directoryReader, error) {
			t.Fatal("client factory must not be called before form submission")
			return nil, nil
		},
		func(string, string) player.Player {
			t.Fatal("player factory must not be called before form submission")
			return nil
		},
		func(_ context.Context, model *app.Model, _ uintptr, _ io.Writer) error {
			interfaceStarted = model != nil
			return nil
		},
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !interfaceStarted {
		t.Fatal("connection interface was not started")
	}
}

func TestRunPropagatesInterfaceError(t *testing.T) {
	t.Parallel()

	want := errors.New("interface failed")
	err := run(
		context.Background(),
		[]string{"-url", "https://example.com/webdav"},
		0,
		&bytes.Buffer{},
		&bytes.Buffer{},
		func(string) string { return "" },
		func(uintptr) ([]byte, error) { return nil, nil },
		func(webdav.Config) (directoryReader, error) { return fakeReader{}, nil },
		successfulPlayerFactory,
		func(context.Context, *app.Model, uintptr, io.Writer) error { return want },
	)
	if !errors.Is(err, want) {
		t.Fatalf("run() error = %v", err)
	}
}

func successfulPlayerFactory(string, string) player.Player {
	return player.Unavailable{}
}

func successfulInterface(context.Context, *app.Model, uintptr, io.Writer) error {
	return nil
}
