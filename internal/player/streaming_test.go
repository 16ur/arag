package player

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/16ur/arag/internal/streaming"
)

type fakeStreamSession struct {
	localURL   *url.URL
	closeErr   error
	closeCalls int
}

func (session *fakeStreamSession) URL() *url.URL {
	if session.localURL == nil {
		return nil
	}
	copy := *session.localURL
	return &copy
}

func (session *fakeStreamSession) Close() error {
	session.closeCalls++
	return session.closeErr
}

type fakeLauncher struct {
	openedURL *url.URL
	err       error
	calls     int
	open      func(*url.URL) error
}

func (launcher *fakeLauncher) Open(_ context.Context, mediaURL *url.URL) error {
	launcher.calls++
	if mediaURL != nil {
		copy := *mediaURL
		launcher.openedURL = &copy
	}
	if launcher.open != nil {
		return launcher.open(mediaURL)
	}
	return launcher.err
}

func TestStreamingPlayerConnectsProxyToLauncher(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != "seiz" || password != "secret" {
			t.Error("upstream request did not contain the expected Basic authentication")
		}
		if request.Header.Get("Range") != "bytes=0-3" {
			t.Errorf("Range = %q", request.Header.Get("Range"))
		}
		writer.Header().Set("Content-Range", "bytes 0-3/4")
		writer.WriteHeader(http.StatusPartialContent)
		_, _ = writer.Write([]byte("data"))
	}))
	defer upstream.Close()

	launcher := &fakeLauncher{open: func(localURL *url.URL) error {
		request, err := http.NewRequest(http.MethodGet, localURL.String(), nil)
		if err != nil {
			return err
		}
		request.Header.Set("Range", "bytes=0-3")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		if response.StatusCode != http.StatusPartialContent || string(body) != "data" {
			return errors.New("unexpected local stream response")
		}
		return nil
	}}
	proxy := streaming.NewProxy(streaming.Config{Username: "seiz", Password: "secret"})
	videoPlayer := NewStreaming(proxy, launcher)
	defer videoPlayer.Close()

	if err := videoPlayer.Open(context.Background(), mustURL(t, upstream.URL+"/video.mkv")); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if launcher.openedURL == nil || launcher.openedURL.Hostname() != "127.0.0.1" {
		t.Fatalf("launcher URL = %v", launcher.openedURL)
	}
}

func TestStreamingPlayerStartsProxyAndLaunchesLocalURL(t *testing.T) {
	t.Parallel()

	remoteURL := mustURL(t, "https://example.com/webdav/video.mkv")
	localURL := mustURL(t, "http://127.0.0.1:49152/token")
	session := &fakeStreamSession{localURL: localURL}
	launcher := &fakeLauncher{}
	ctx := context.Background()
	var receivedContext context.Context
	var receivedSource *url.URL
	videoPlayer := &Streaming{
		start: func(gotContext context.Context, source *url.URL) (streamSession, error) {
			receivedContext = gotContext
			receivedSource = source
			return session, nil
		},
		launcher: launcher,
	}

	if err := videoPlayer.Open(ctx, remoteURL); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if receivedContext != ctx || receivedSource != remoteURL {
		t.Fatal("Open() did not forward the context and remote URL")
	}
	if launcher.calls != 1 || launcher.openedURL.String() != localURL.String() {
		t.Fatalf("launcher calls = %d, URL = %v", launcher.calls, launcher.openedURL)
	}
	if videoPlayer.active != session || session.closeCalls != 0 {
		t.Fatalf("active session = %v, close calls = %d", videoPlayer.active, session.closeCalls)
	}
}

func TestStreamingPlayerReplacesPreviousSessionAfterSuccessfulLaunch(t *testing.T) {
	t.Parallel()

	first := &fakeStreamSession{localURL: mustURL(t, "http://127.0.0.1:49152/first")}
	second := &fakeStreamSession{localURL: mustURL(t, "http://127.0.0.1:49153/second")}
	sessions := []*fakeStreamSession{first, second}
	videoPlayer := &Streaming{
		start: func(context.Context, *url.URL) (streamSession, error) {
			session := sessions[0]
			sessions = sessions[1:]
			return session, nil
		},
		launcher: &fakeLauncher{},
	}

	if err := videoPlayer.Open(context.Background(), mustURL(t, "https://example.com/first.mkv")); err != nil {
		t.Fatal(err)
	}
	if err := videoPlayer.Open(context.Background(), mustURL(t, "https://example.com/second.mkv")); err != nil {
		t.Fatal(err)
	}
	if first.closeCalls != 1 || second.closeCalls != 0 || videoPlayer.active != second {
		t.Fatalf("first closes = %d, second closes = %d, active = %v", first.closeCalls, second.closeCalls, videoPlayer.active)
	}
}

func TestStreamingPlayerClosesNewSessionWhenLaunchFails(t *testing.T) {
	t.Parallel()

	want := errors.New("IINA failed")
	previous := &fakeStreamSession{localURL: mustURL(t, "http://127.0.0.1:49152/previous")}
	next := &fakeStreamSession{localURL: mustURL(t, "http://127.0.0.1:49153/next")}
	videoPlayer := &Streaming{
		start: func(context.Context, *url.URL) (streamSession, error) {
			return next, nil
		},
		launcher: &fakeLauncher{err: want},
		active:   previous,
	}

	err := videoPlayer.Open(context.Background(), mustURL(t, "https://example.com/video.mkv"))
	if !errors.Is(err, want) {
		t.Fatalf("Open() error = %v", err)
	}
	if next.closeCalls != 1 || previous.closeCalls != 0 || videoPlayer.active != previous {
		t.Fatalf("next closes = %d, previous closes = %d, active = %v", next.closeCalls, previous.closeCalls, videoPlayer.active)
	}
}

func TestStreamingPlayerPropagatesProxyFailure(t *testing.T) {
	t.Parallel()

	want := errors.New("proxy failed")
	launcher := &fakeLauncher{}
	videoPlayer := &Streaming{
		start: func(context.Context, *url.URL) (streamSession, error) {
			return nil, want
		},
		launcher: launcher,
	}
	err := videoPlayer.Open(context.Background(), mustURL(t, "https://example.com/video.mp4"))
	if !errors.Is(err, want) || launcher.calls != 0 {
		t.Fatalf("Open() error = %v, launcher calls = %d", err, launcher.calls)
	}
}

func TestStreamingPlayerRejectsMissingSession(t *testing.T) {
	t.Parallel()

	launcher := &fakeLauncher{}
	videoPlayer := &Streaming{
		start: func(context.Context, *url.URL) (streamSession, error) {
			return nil, nil
		},
		launcher: launcher,
	}
	err := videoPlayer.Open(context.Background(), mustURL(t, "https://example.com/video.mp4"))
	if err == nil || launcher.calls != 0 {
		t.Fatalf("Open() error = %v, launcher calls = %d", err, launcher.calls)
	}
}

func TestStreamingPlayerCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	session := &fakeStreamSession{localURL: mustURL(t, "http://127.0.0.1:49152/token")}
	videoPlayer := &Streaming{active: session}
	if err := videoPlayer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := videoPlayer.Close(); err != nil {
		t.Fatal(err)
	}
	if session.closeCalls != 1 || videoPlayer.active != nil {
		t.Fatalf("close calls = %d, active = %v", session.closeCalls, videoPlayer.active)
	}
}

func TestStreamingPlayerRequiresDependencies(t *testing.T) {
	t.Parallel()

	videoPlayer := NewStreaming(nil, nil)
	if err := videoPlayer.Open(context.Background(), mustURL(t, "https://example.com/video.mkv")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Open() error = %v, want ErrUnavailable", err)
	}
}
