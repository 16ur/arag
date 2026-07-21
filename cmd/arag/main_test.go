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

	"github.com/16ur/arag/internal/webdav"
)

type fakeReader struct{}

func (fakeReader) ReadDir(context.Context, *url.URL) ([]webdav.Entry, error) {
	return nil, nil
}

func TestRunBuildsClientAndStartsInterface(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	var receivedConfig webdav.Config
	passwordReads := 0
	interfaceStarted := false

	err := run(
		context.Background(),
		[]string{"-url", "https://example.com/webdav", "-user", "axel", "-timeout", "5s"},
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
		func(context.Context, directoryReader, uintptr, io.Writer) error {
			interfaceStarted = true
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
		receivedConfig.Username != "axel" ||
		receivedConfig.Password != "secret" ||
		receivedConfig.RequestTimeout != 5*time.Second {
		t.Errorf("config = %+v", receivedConfig)
	}
	if !interfaceStarted {
		t.Error("interface was not started")
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
		[]string{"-url", "https://example.com/webdav", "-user", "axel"},
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
		successfulInterface,
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if receivedPassword != "from-environment" {
		t.Errorf("password = %q", receivedPassword)
	}
}

func TestRunRequiresURL(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := run(
		context.Background(), nil, 0, &bytes.Buffer{}, &stderr,
		func(string) string { return "" },
		func(uintptr) ([]byte, error) { return nil, nil },
		func(webdav.Config) (directoryReader, error) {
			t.Fatal("client factory must not be called")
			return nil, nil
		},
		successfulInterface,
	)
	if err == nil || !strings.Contains(err.Error(), "-url") {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("stderr = %q", stderr.String())
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
		func(context.Context, directoryReader, uintptr, io.Writer) error { return want },
	)
	if !errors.Is(err, want) {
		t.Fatalf("run() error = %v", err)
	}
}

func successfulInterface(context.Context, directoryReader, uintptr, io.Writer) error {
	return nil
}
