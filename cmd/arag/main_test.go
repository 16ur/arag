package main

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/16ur/arag/internal/webdav"
)

type fakeReader struct {
	entries []webdav.Entry
	err     error
}

func (reader fakeReader) ReadDir(context.Context, *url.URL) ([]webdav.Entry, error) {
	return reader.entries, reader.err
}

func TestRunListsRootAndPromptsForPassword(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var receivedConfig webdav.Config
	passwordReads := 0

	err := run(
		context.Background(),
		[]string{"-url", "https://example.com/webdav", "-user", "axel", "-timeout", "5s"},
		0,
		&stdout,
		&stderr,
		func(string) string { return "" },
		func(uintptr) ([]byte, error) {
			passwordReads++
			return []byte("secret"), nil
		},
		func(config webdav.Config) (directoryReader, error) {
			receivedConfig = config
			return fakeReader{entries: []webdav.Entry{
				{Name: "video.mkv", Size: 2048},
				{Name: "Films", IsCollection: true},
			}}, nil
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
	if !strings.Contains(stderr.String(), "WebDAV password") {
		t.Errorf("stderr = %q", stderr.String())
	}
	output := stdout.String()
	if strings.Index(output, "Films") > strings.Index(output, "video.mkv") {
		t.Errorf("directories should be listed first:\n%s", output)
	}
	if !strings.Contains(output, "DIRECTORY") || !strings.Contains(output, "2.0 KiB") {
		t.Errorf("stdout = %q", output)
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
	)
	if err == nil || !strings.Contains(err.Error(), "-url") {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestRunPropagatesWebDAVError(t *testing.T) {
	t.Parallel()

	err := run(
		context.Background(),
		[]string{"-url", "https://example.com/webdav"},
		0,
		&bytes.Buffer{},
		&bytes.Buffer{},
		func(string) string { return "" },
		func(uintptr) ([]byte, error) { return nil, nil },
		func(webdav.Config) (directoryReader, error) {
			return fakeReader{err: webdav.ErrAuthentication}, nil
		},
	)
	if !errors.Is(err, webdav.ErrAuthentication) {
		t.Fatalf("run() error = %v", err)
	}
	if got := friendlyError(err); got != "the server rejected the credentials" {
		t.Errorf("friendlyError() = %q", got)
	}
}

func TestPrintEntriesEmpty(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	printEntries(&output, nil)
	if output.String() != "Empty directory.\n" {
		t.Errorf("output = %q", output.String())
	}
}
