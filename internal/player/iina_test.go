package player

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestIINAOpensValidatedLoopbackURL(t *testing.T) {
	t.Parallel()

	mediaURL := mustURL(t, "http://127.0.0.1:49152/random-token")
	ctx := context.Background()
	var commandName string
	var commandArgs []string
	var receivedContext context.Context
	launcher := &IINA{
		platform: "darwin",
		run: func(gotContext context.Context, name string, args ...string) error {
			receivedContext = gotContext
			commandName = name
			commandArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := launcher.Open(ctx, mediaURL); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if receivedContext != ctx {
		t.Fatal("Open() did not forward the context")
	}
	if commandName != macOSOpenPath {
		t.Fatalf("command name = %q", commandName)
	}
	wantArgs := []string{"-a", "IINA", "-u", mediaURL.String()}
	if !reflect.DeepEqual(commandArgs, wantArgs) {
		t.Fatalf("command args = %#v, want %#v", commandArgs, wantArgs)
	}
}

func TestIINARejectsUnsafeMediaURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mediaURL *url.URL
	}{
		{name: "missing URL"},
		{name: "HTTPS", mediaURL: mustURL(t, "https://127.0.0.1:49152/token")},
		{name: "remote host", mediaURL: mustURL(t, "http://example.com:49152/token")},
		{name: "credentials", mediaURL: mustURL(t, "http://user:secret@127.0.0.1:49152/token")},
		{name: "missing port", mediaURL: mustURL(t, "http://127.0.0.1/token")},
		{name: "missing token", mediaURL: mustURL(t, "http://127.0.0.1:49152/")},
		{name: "query", mediaURL: mustURL(t, "http://127.0.0.1:49152/token?secret=value")},
		{name: "fragment", mediaURL: mustURL(t, "http://127.0.0.1:49152/token#fragment")},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			called := false
			launcher := &IINA{
				platform: "darwin",
				run: func(context.Context, string, ...string) error {
					called = true
					return nil
				},
			}
			err := launcher.Open(context.Background(), test.mediaURL)
			if !errors.Is(err, ErrInvalidMediaURL) {
				t.Fatalf("Open() error = %v, want ErrInvalidMediaURL", err)
			}
			if called {
				t.Fatal("command runner was called for an invalid URL")
			}
		})
	}
}

func TestIINARejectsUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	called := false
	launcher := &IINA{
		platform: "linux",
		run: func(context.Context, string, ...string) error {
			called = true
			return nil
		},
	}
	err := launcher.Open(context.Background(), mustURL(t, "http://127.0.0.1:49152/token"))
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("Open() error = %v, want ErrUnsupportedPlatform", err)
	}
	if called {
		t.Fatal("command runner was called on an unsupported platform")
	}
}

func TestIINAPropagatesLaunchFailureWithoutURL(t *testing.T) {
	t.Parallel()

	want := errors.New("application not found")
	launcher := &IINA{
		platform: "darwin",
		run: func(context.Context, string, ...string) error {
			return want
		},
	}
	mediaURL := mustURL(t, "http://127.0.0.1:49152/sensitive-token")
	err := launcher.Open(context.Background(), mediaURL)
	if !errors.Is(err, want) {
		t.Fatalf("Open() error = %v", err)
	}
	if errorText := err.Error(); errorText == "" || strings.Contains(errorText, "sensitive-token") {
		t.Fatalf("Open() leaked the local token: %q", errorText)
	}
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
