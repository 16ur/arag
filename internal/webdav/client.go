// Package webdav provides access to remote files through WebDAV.
package webdav

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConnectionTimeout = 10 * time.Second
	defaultRequestTimeout    = 30 * time.Second
	maxResponseSize          = 16 << 20 // 16 MiB
)

var (
	// ErrAuthentication indicates that the server rejected the credentials.
	ErrAuthentication = errors.New("webdav authentication failed")
	// ErrUnexpectedStatus indicates that the server did not return a WebDAV
	// Multi-Status response.
	ErrUnexpectedStatus = errors.New("unexpected webdav status")
	// ErrInvalidResponse indicates that the server returned invalid WebDAV XML.
	ErrInvalidResponse = errors.New("invalid webdav response")
)

// Config configures a WebDAV client.
type Config struct {
	BaseURL        string
	Username       string
	Password       string
	HTTPClient     *http.Client
	RequestTimeout time.Duration
}

// Entry describes a file or collection returned by a WebDAV server.
type Entry struct {
	Name         string
	URL          *url.URL
	IsCollection bool
	Size         int64
	ModTime      time.Time
	ETag         string
}

// Client lists remote WebDAV collections.
type Client struct {
	baseURL        *url.URL
	username       string
	password       string
	httpClient     *http.Client
	requestTimeout time.Duration
}

// NewClient validates config and creates a WebDAV client.
func NewClient(config Config) (*Client, error) {
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse WebDAV URL: %w", err)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("parse WebDAV URL: scheme must be http or https")
	}
	if baseURL.Host == "" {
		return nil, fmt.Errorf("parse WebDAV URL: host is required")
	}
	if baseURL.User != nil {
		return nil, fmt.Errorf("parse WebDAV URL: credentials must not be included in the URL")
	}
	if baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("parse WebDAV URL: query and fragment are not supported")
	}

	baseURL.Path = ensureTrailingSlash(baseURL.Path)
	baseURL.RawPath = ""

	requestTimeout := config.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = defaultRequestTimeout
	}
	if requestTimeout < 0 {
		return nil, fmt.Errorf("request timeout must not be negative")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	return &Client{
		baseURL:        baseURL,
		username:       config.Username,
		password:       config.Password,
		httpClient:     httpClient,
		requestTimeout: requestTimeout,
	}, nil
}

// BaseURL returns a copy of the configured WebDAV root URL.
func (c *Client) BaseURL() *url.URL {
	copy := *c.baseURL
	return &copy
}

// ReadDir returns the direct children of directory. A nil directory selects
// the configured WebDAV root.
func (c *Client) ReadDir(ctx context.Context, directory *url.URL) ([]Entry, error) {
	directoryURL, err := c.resolveDirectory(directory)
	if err != nil {
		return nil, err
	}

	requestContext, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestContext,
		"PROPFIND",
		directoryURL.String(),
		strings.NewReader(propfindBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create PROPFIND request: %w", err)
	}
	request.Header.Set("Depth", "1")
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")
	if c.username != "" || c.password != "" {
		request.SetBasicAuth(c.username, c.password)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute PROPFIND: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: HTTP %d", ErrAuthentication, response.StatusCode)
	}
	if response.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUnexpectedStatus, response.StatusCode)
	}

	entries, err := parseResponse(io.LimitReader(response.Body, maxResponseSize), directoryURL)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *Client) resolveDirectory(directory *url.URL) (*url.URL, error) {
	if directory == nil {
		return c.BaseURL(), nil
	}
	if !sameOrigin(c.baseURL, directory) {
		return nil, fmt.Errorf("directory URL must use the configured WebDAV origin")
	}
	if !withinRoot(c.baseURL, directory) {
		return nil, fmt.Errorf("directory URL must remain inside the configured WebDAV root")
	}
	if directory.User != nil {
		return nil, fmt.Errorf("directory URL must not contain credentials")
	}

	resolved := *directory
	resolved.Path = ensureTrailingSlash(resolved.Path)
	resolved.RawPath = ""
	resolved.RawQuery = ""
	resolved.Fragment = ""
	return &resolved, nil
}

func parseResponse(reader io.Reader, directoryURL *url.URL) ([]Entry, error) {
	var document multiStatus
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if document.XMLName.Local != "multistatus" || document.XMLName.Space != "DAV:" {
		return nil, fmt.Errorf("%w: root element must be DAV: multistatus", ErrInvalidResponse)
	}

	entries := make([]Entry, 0, len(document.Responses))
	for _, response := range document.Responses {
		entry, ok, err := response.entry(directoryURL)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (response davResponse) entry(directoryURL *url.URL) (Entry, bool, error) {
	href, err := url.Parse(strings.TrimSpace(response.Href))
	if err != nil || response.Href == "" {
		return Entry{}, false, fmt.Errorf("invalid href %q", response.Href)
	}
	entryURL := directoryURL.ResolveReference(href)
	if !sameOrigin(directoryURL, entryURL) {
		return Entry{}, false, fmt.Errorf("href %q uses a different origin", response.Href)
	}
	if !withinRoot(directoryURL, entryURL) {
		return Entry{}, false, fmt.Errorf("href %q is outside the requested collection", response.Href)
	}
	entryURL.User = nil
	entryURL.RawQuery = ""
	entryURL.Fragment = ""

	if sameResource(directoryURL, entryURL) {
		return Entry{}, false, nil
	}

	properties, ok := response.successfulProperties()
	if !ok {
		return Entry{}, false, nil
	}

	entry := Entry{
		Name:         path.Base(strings.TrimSuffix(entryURL.Path, "/")),
		URL:          entryURL,
		IsCollection: properties.ResourceType.Collection != nil,
		ETag:         strings.TrimSpace(properties.ETag),
	}
	if entry.Name == "." || entry.Name == "/" || entry.Name == "" {
		return Entry{}, false, fmt.Errorf("href %q has no resource name", response.Href)
	}

	if size := strings.TrimSpace(properties.ContentLength); size != "" {
		entry.Size, err = strconv.ParseInt(size, 10, 64)
		if err != nil || entry.Size < 0 {
			return Entry{}, false, fmt.Errorf("invalid content length %q", size)
		}
	}
	entry.ModTime = parseDate(properties.LastModified, properties.CreationDate)
	return entry, true, nil
}

func (response davResponse) successfulProperties() (properties, bool) {
	for _, propstat := range response.PropStats {
		if statusCode(propstat.Status) >= 200 && statusCode(propstat.Status) < 300 {
			return propstat.Properties, true
		}
	}
	return properties{}, false
}

func statusCode(status string) int {
	fields := strings.Fields(status)
	if len(fields) < 2 {
		return 0
	}
	code, _ := strconv.Atoi(fields[1])
	return code
}

func parseDate(lastModified, creationDate string) time.Time {
	if parsed, err := http.ParseTime(strings.TrimSpace(lastModified)); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(creationDate)); err == nil {
		return parsed
	}
	return time.Time{}
}

func sameOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) &&
		strings.EqualFold(left.Host, right.Host)
}

func sameResource(left, right *url.URL) bool {
	return sameOrigin(left, right) &&
		strings.TrimSuffix(left.EscapedPath(), "/") == strings.TrimSuffix(right.EscapedPath(), "/")
}

func withinRoot(root, candidate *url.URL) bool {
	rootPath := ensureTrailingSlash(root.EscapedPath())
	candidatePath := ensureTrailingSlash(candidate.EscapedPath())
	return sameOrigin(root, candidate) && strings.HasPrefix(candidatePath, rootPath)
}

func ensureTrailingSlash(value string) string {
	if value == "" {
		return "/"
	}
	if strings.HasSuffix(value, "/") {
		return value
	}
	return value + "/"
}

func defaultHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: defaultConnectionTimeout}).DialContext
	transport.ResponseHeaderTimeout = defaultConnectionTimeout
	return &http.Client{Transport: transport}
}

type multiStatus struct {
	XMLName   xml.Name      `xml:"multistatus"`
	Responses []davResponse `xml:"response"`
}

type davResponse struct {
	Href      string     `xml:"href"`
	PropStats []propStat `xml:"propstat"`
}

type propStat struct {
	Properties properties `xml:"prop"`
	Status     string     `xml:"status"`
}

type properties struct {
	ResourceType  resourceType `xml:"resourcetype"`
	ContentLength string       `xml:"getcontentlength"`
	CreationDate  string       `xml:"creationdate"`
	LastModified  string       `xml:"getlastmodified"`
	ETag          string       `xml:"getetag"`
}

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

const propfindBody = `<?xml version="1.0" encoding="utf-8"?>
<propfind xmlns="DAV:">
  <prop>
    <resourcetype/>
    <getcontentlength/>
    <creationdate/>
    <getlastmodified/>
    <getetag/>
  </prop>
</propfind>`
