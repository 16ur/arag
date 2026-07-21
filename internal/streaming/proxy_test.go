package streaming

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestProxyForwardsAuthenticatedRangeRequest(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != "seiz" || password != "secret" {
			t.Errorf("BasicAuth() = %q, %q, %v", username, password, ok)
		}
		if request.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", request.Method)
		}
		if got := request.Header.Get("Range"); got != "bytes=5-9" {
			t.Errorf("Range = %q", got)
		}
		if got := request.Header.Get("If-Range"); got != "video-tag" {
			t.Errorf("If-Range = %q", got)
		}
		if got := request.Header.Get("Accept-Encoding"); got != "identity" {
			t.Errorf("Accept-Encoding = %q", got)
		}

		writer.Header().Set("Accept-Ranges", "bytes")
		writer.Header().Set("Content-Length", "5")
		writer.Header().Set("Content-Range", "bytes 5-9/10")
		writer.Header().Set("Content-Type", "video/x-matroska")
		writer.Header().Set("ETag", "video-tag")
		writer.WriteHeader(http.StatusPartialContent)
		_, _ = writer.Write([]byte("56789"))
	}))
	defer upstream.Close()

	proxy := NewProxy(Config{
		Username:   "seiz",
		Password:   "secret",
		HTTPClient: upstream.Client(),
	})
	session := startSession(t, proxy, upstream.URL+"/video.mkv")
	defer session.Close()

	localURL := session.URL()
	if localURL.User != nil || strings.Contains(localURL.String(), "secret") || strings.Contains(localURL.String(), upstream.URL) {
		t.Fatalf("local URL exposes upstream data: %s", localURL)
	}
	if localURL.Hostname() != "127.0.0.1" {
		t.Fatalf("local hostname = %q", localURL.Hostname())
	}

	request, err := http.NewRequest(http.MethodGet, localURL.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Range", "bytes=5-9")
	request.Header.Set("If-Range", "video-tag")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("local request error = %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusPartialContent || string(body) != "56789" {
		t.Fatalf("response = HTTP %d, %q", response.StatusCode, body)
	}
	if response.Header.Get("Content-Range") != "bytes 5-9/10" ||
		response.Header.Get("Accept-Ranges") != "bytes" ||
		response.Header.Get("ETag") != "video-tag" {
		t.Fatalf("response headers = %v", response.Header)
	}
}

func TestProxyForwardsHeadWithoutBody(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodHead {
			t.Errorf("method = %q, want HEAD", request.Method)
		}
		writer.Header().Set("Content-Length", "1024")
		writer.Header().Set("Content-Type", "video/mp4")
		writer.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	session := startSession(t, NewProxy(Config{}), upstream.URL+"/video.mp4")
	defer session.Close()
	request, err := http.NewRequest(http.MethodHead, session.URL().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) != 0 || response.ContentLength != 1024 || response.Header.Get("Content-Type") != "video/mp4" {
		t.Fatalf("HEAD response = length %d, content length %d, headers %v", len(body), response.ContentLength, response.Header)
	}
}

func TestProxyRejectsWrongTokenAndMethod(t *testing.T) {
	t.Parallel()

	var upstreamCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		upstreamCalls.Add(1)
	}))
	defer upstream.Close()
	session := startSession(t, NewProxy(Config{}), upstream.URL+"/video.mkv")
	defer session.Close()

	wrongURL := session.URL()
	wrongURL.Path = "/wrong-token"
	response, err := http.Get(wrongURL.String())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong token status = %d", response.StatusCode)
	}

	request, err := http.NewRequest(http.MethodPost, session.URL().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != "GET, HEAD" {
		t.Fatalf("POST response = HTTP %d, Allow %q", response.StatusCode, response.Header.Get("Allow"))
	}
	if upstreamCalls.Load() != 0 {
		t.Fatalf("upstream calls = %d, want 0", upstreamCalls.Load())
	}
}

func TestProxyRejectsCrossOriginRedirect(t *testing.T) {
	t.Parallel()

	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		destinationCalls.Add(1)
	}))
	defer destination.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, destination.URL+"/stolen", http.StatusFound)
	}))
	defer upstream.Close()

	session := startSession(t, NewProxy(Config{Username: "seiz", Password: "secret"}), upstream.URL+"/video.mkv")
	defer session.Close()
	response, err := http.Get(session.URL().String())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadGateway {
		t.Fatalf("redirect response status = %d", response.StatusCode)
	}
	if destinationCalls.Load() != 0 {
		t.Fatalf("redirect destination calls = %d, want 0", destinationCalls.Load())
	}
}

func TestSessionStopsWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	ctx, cancel := context.WithCancel(context.Background())
	source, err := url.Parse(upstream.URL + "/video.mkv")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewProxy(Config{}).Start(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-session.Done():
	case <-time.After(time.Second):
		t.Fatal("session did not stop after context cancellation")
	}
}

func TestProxyValidatesSourceURL(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"ftp://example.com/video.mkv",
		"https:///video.mkv",
		"https://user:secret@example.com/video.mkv",
	}
	proxy := NewProxy(Config{})
	for _, value := range tests {
		var source *url.URL
		if value != "" {
			parsed, err := url.Parse(value)
			if err != nil {
				t.Fatal(err)
			}
			source = parsed
		}
		if _, err := proxy.Start(context.Background(), source); err == nil {
			t.Errorf("Start(%q) error = nil", value)
		}
	}
}

func TestSessionCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer upstream.Close()
	session := startSession(t, NewProxy(Config{}), upstream.URL+"/video.mkv")
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.Done():
	case <-time.After(time.Second):
		t.Fatal("session server did not stop")
	}
}

func startSession(t *testing.T, proxy *Proxy, sourceValue string) *Session {
	t.Helper()
	source, err := url.Parse(sourceValue)
	if err != nil {
		t.Fatal(err)
	}
	session, err := proxy.Start(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	return session
}
