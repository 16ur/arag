package webdav

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestClientReadDir(t *testing.T) {
	t.Parallel()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "PROPFIND" {
			t.Errorf("method = %q, want PROPFIND", request.Method)
		}
		if depth := request.Header.Get("Depth"); depth != "1" {
			t.Errorf("Depth = %q, want 1", depth)
		}
		username, password, ok := request.BasicAuth()
		if !ok || username != "seiz" || password != "secret" {
			t.Error("request did not contain the expected Basic authentication")
		}

		writer.Header().Set("Content-Type", "application/xml")
		writer.WriteHeader(http.StatusMultiStatus)
		_, _ = fmt.Fprintf(writer, `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/webdav/</d:href>
    <d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
  </d:response>
  <d:response>
    <d:href>/webdav/Films/</d:href>
    <d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype><d:getetag>folder-tag</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
  </d:response>
  <d:response>
    <d:href>%s/webdav/Mon%%20film.mkv</d:href>
    <d:propstat><d:prop><d:resourcetype/><d:getcontentlength>4096</d:getcontentlength><d:getlastmodified>Mon, 02 Jan 2006 15:04:05 GMT</d:getlastmodified><d:getetag>video-tag</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
    <d:propstat><d:prop><d:creationdate/></d:prop><d:status>HTTP/1.1 404 Not Found</d:status></d:propstat>
  </d:response>
</d:multistatus>`, server.URL)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL + "/webdav",
		Username: "seiz",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	entries, err := client.ReadDir(context.Background(), nil)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}

	folder := entries[0]
	if folder.Name != "Films" || !folder.IsCollection || folder.ETag != "folder-tag" {
		t.Errorf("folder = %+v", folder)
	}
	if got := folder.URL.String(); got != server.URL+"/webdav/Films/" {
		t.Errorf("folder URL = %q", got)
	}

	video := entries[1]
	if video.Name != "Mon film.mkv" || video.IsCollection || video.Size != 4096 || video.ETag != "video-tag" {
		t.Errorf("video = %+v", video)
	}
	wantDate := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	if !video.ModTime.Equal(wantDate) {
		t.Errorf("video ModTime = %v, want %v", video.ModTime, wantDate)
	}
}

func TestClientReadDirSupportsDefaultNamespaceAndCreationDate(t *testing.T) {
	t.Parallel()

	server := newWebDAVServer(t, http.StatusMultiStatus, `
<multistatus xmlns="DAV:">
  <response>
    <href>/root/video.mp4</href>
    <propstat>
      <prop>
        <resourcetype/>
        <creationdate>2025-06-01T10:30:00Z</creationdate>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`)
	defer server.Close()

	client := mustClient(t, Config{BaseURL: server.URL + "/root/"})
	entries, err := client.ReadDir(context.Background(), nil)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].ModTime.Format(time.RFC3339) != "2025-06-01T10:30:00Z" {
		t.Errorf("ModTime = %v", entries[0].ModTime)
	}
}

func TestClientReadDirErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		want       error
	}{
		{name: "unauthorized", statusCode: http.StatusUnauthorized, want: ErrAuthentication},
		{name: "forbidden", statusCode: http.StatusForbidden, want: ErrAuthentication},
		{name: "not WebDAV", statusCode: http.StatusOK, want: ErrUnexpectedStatus},
		{name: "invalid XML", statusCode: http.StatusMultiStatus, body: `<multistatus>`, want: ErrInvalidResponse},
		{name: "not WebDAV XML", statusCode: http.StatusMultiStatus, body: `<multistatus/>`, want: ErrInvalidResponse},
		{name: "invalid size", statusCode: http.StatusMultiStatus, body: multiStatusWithSize("large"), want: ErrInvalidResponse},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := newWebDAVServer(t, test.statusCode, test.body)
			defer server.Close()

			client := mustClient(t, Config{BaseURL: server.URL + "/webdav/"})
			_, err := client.ReadDir(context.Background(), nil)
			if !errors.Is(err, test.want) {
				t.Fatalf("ReadDir() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestClientReadDirTimesOut(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
			writer.WriteHeader(http.StatusMultiStatus)
		case <-request.Context().Done():
		}
	}))
	defer server.Close()

	client := mustClient(t, Config{
		BaseURL:        server.URL + "/webdav/",
		RequestTimeout: 10 * time.Millisecond,
	})
	_, err := client.ReadDir(context.Background(), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ReadDir() error = %v, want context deadline exceeded", err)
	}
}

func TestClientRejectsAnotherOrigin(t *testing.T) {
	t.Parallel()

	client := mustClient(t, Config{BaseURL: "https://example.com/webdav/"})
	directory, err := url.Parse("https://attacker.example/webdav/")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ReadDir(context.Background(), directory)
	if err == nil || !strings.Contains(err.Error(), "configured WebDAV origin") {
		t.Fatalf("ReadDir() error = %v", err)
	}
}

func TestClientRejectsDirectoryOutsideRoot(t *testing.T) {
	t.Parallel()

	client := mustClient(t, Config{BaseURL: "https://example.com/webdav/"})
	directory, err := url.Parse("https://example.com/private/")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ReadDir(context.Background(), directory)
	if err == nil || !strings.Contains(err.Error(), "configured WebDAV root") {
		t.Fatalf("ReadDir() error = %v", err)
	}
}

func TestNewClientValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
	}{
		{name: "missing scheme", config: Config{BaseURL: "example.com/webdav"}},
		{name: "unsupported scheme", config: Config{BaseURL: "ftp://example.com/webdav"}},
		{name: "missing host", config: Config{BaseURL: "https:///webdav"}},
		{name: "credentials in URL", config: Config{BaseURL: "https://user:secret@example.com/webdav"}},
		{name: "query", config: Config{BaseURL: "https://example.com/webdav?token=secret"}},
		{name: "negative timeout", config: Config{BaseURL: "https://example.com/webdav", RequestTimeout: -time.Second}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewClient(test.config); err == nil {
				t.Fatal("NewClient() error = nil")
			}
		})
	}
}

func TestBaseURLReturnsCopy(t *testing.T) {
	t.Parallel()

	client := mustClient(t, Config{BaseURL: "https://example.com/webdav"})
	first := client.BaseURL()
	first.Path = "/changed/"
	if got := client.BaseURL().String(); got != "https://example.com/webdav/" {
		t.Fatalf("BaseURL() = %q", got)
	}
}

func mustClient(t *testing.T, config Config) *Client {
	t.Helper()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func newWebDAVServer(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/xml")
		writer.WriteHeader(statusCode)
		_, _ = writer.Write([]byte(body))
	}))
}

func multiStatusWithSize(size string) string {
	return `<multistatus xmlns="DAV:">
  <response>
    <href>/webdav/video.mkv</href>
    <propstat>
      <prop><resourcetype/><getcontentlength>` + size + `</getcontentlength></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`
}
