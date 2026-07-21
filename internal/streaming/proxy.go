// Package streaming exposes authenticated remote media through a temporary
// loopback-only HTTP endpoint.
package streaming

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	connectionTimeout = 10 * time.Second
	maximumRedirects  = 10
	tokenBytes        = 32
)

// Config configures authenticated upstream media requests.
type Config struct {
	Username   string
	Password   string
	HTTPClient *http.Client
}

// Proxy creates temporary loopback streaming sessions.
type Proxy struct {
	username   string
	password   string
	httpClient *http.Client
}

// Session represents one temporary local media endpoint.
type Session struct {
	url       *url.URL
	server    *http.Server
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

// NewProxy creates a streaming proxy. Credentials remain in memory and are
// added only to requests sent to the fixed upstream media origin.
func NewProxy(config Config) *Proxy {
	return &Proxy{
		username:   config.Username,
		password:   config.Password,
		httpClient: config.HTTPClient,
	}
}

// Start exposes source through a random loopback URL until the context is
// canceled or the returned session is closed.
func (p *Proxy) Start(ctx context.Context, source *url.URL) (*Session, error) {
	upstream, err := validateSource(source)
	if err != nil {
		return nil, err
	}
	token, err := randomToken()
	if err != nil {
		return nil, fmt.Errorf("generate stream token: %w", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start loopback stream: %w", err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	localURL := &url.URL{
		Scheme: "http",
		Host:   listener.Addr().String(),
		Path:   "/" + token,
	}
	session := &Session{
		url:  localURL,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	handler := &streamHandler{
		tokenPath:  localURL.Path,
		source:     upstream,
		username:   p.username,
		password:   p.password,
		httpClient: p.clientFor(upstream),
	}
	session.server = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: connectionTimeout,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}

	go func() {
		_ = session.server.Serve(listener)
		close(session.done)
	}()
	go func() {
		select {
		case <-ctx.Done():
			_ = session.Close()
		case <-session.stop:
		}
	}()
	return session, nil
}

// URL returns a copy of the temporary loopback media URL.
func (s *Session) URL() *url.URL {
	copy := *s.url
	return &copy
}

// Done is closed after the local streaming server stops.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// Close stops the local streaming server and interrupts active streams.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.stop)
		s.closeErr = s.server.Close()
		if errors.Is(s.closeErr, http.ErrServerClosed) {
			s.closeErr = nil
		}
	})
	return s.closeErr
}

func (p *Proxy) clientFor(source *url.URL) *http.Client {
	var client http.Client
	if p.httpClient != nil {
		client = *p.httpClient
	} else {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.DialContext = (&net.Dialer{Timeout: connectionTimeout}).DialContext
		transport.ResponseHeaderTimeout = connectionTimeout
		client.Transport = transport
	}
	configuredRedirect := client.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= maximumRedirects {
			return errors.New("too many upstream redirects")
		}
		if !sameOrigin(source, request.URL) {
			return errors.New("upstream redirect changed origin")
		}
		if configuredRedirect != nil {
			return configuredRedirect(request, via)
		}
		return nil
	}
	return &client
}

type streamHandler struct {
	tokenPath  string
	source     *url.URL
	username   string
	password   string
	httpClient *http.Client
}

func (h *streamHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !isLoopbackRequest(request) {
		http.Error(writer, "forbidden", http.StatusForbidden)
		return
	}
	if !matchesToken(request.URL.Path, h.tokenPath) || request.URL.RawQuery != "" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upstreamRequest, err := http.NewRequestWithContext(
		request.Context(),
		request.Method,
		h.source.String(),
		nil,
	)
	if err != nil {
		http.Error(writer, "could not create upstream request", http.StatusBadGateway)
		return
	}
	copyRequestHeaders(upstreamRequest.Header, request.Header)
	upstreamRequest.Header.Set("Accept-Encoding", "identity")
	if h.username != "" || h.password != "" {
		upstreamRequest.SetBasicAuth(h.username, h.password)
	}

	response, err := h.httpClient.Do(upstreamRequest)
	if err != nil {
		http.Error(writer, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	copyResponseHeaders(writer.Header(), response.Header)
	writer.WriteHeader(response.StatusCode)
	if request.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(writer, response.Body)
}

func validateSource(source *url.URL) (*url.URL, error) {
	if source == nil {
		return nil, errors.New("stream source URL is required")
	}
	if source.Scheme != "http" && source.Scheme != "https" {
		return nil, errors.New("stream source URL must use HTTP or HTTPS")
	}
	if source.Host == "" {
		return nil, errors.New("stream source URL requires a host")
	}
	if source.User != nil {
		return nil, errors.New("stream source URL must not contain credentials")
	}
	copy := *source
	copy.Fragment = ""
	return &copy, nil
}

func randomToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func matchesToken(actual, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func isLoopbackRequest(request *http.Request) bool {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return false
	}
	address := net.ParseIP(strings.Trim(host, "[]"))
	return address != nil && address.IsLoopback()
}

func sameOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) &&
		strings.EqualFold(left.Host, right.Host)
}

func copyRequestHeaders(destination, source http.Header) {
	for _, name := range []string{"Range", "If-Range", "If-None-Match", "If-Modified-Since"} {
		if value := source.Get(name); value != "" {
			destination.Set(name, value)
		}
	}
}

func copyResponseHeaders(destination, source http.Header) {
	for _, name := range []string{
		"Accept-Ranges",
		"Cache-Control",
		"Content-Length",
		"Content-Range",
		"Content-Type",
		"ETag",
		"Last-Modified",
	} {
		if values := source.Values(name); len(values) > 0 {
			destination[name] = append([]string(nil), values...)
		}
	}
}
