package player

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/16ur/arag/internal/streaming"
)

type streamSession interface {
	URL() *url.URL
	Close() error
}

type streamStarter func(context.Context, *url.URL) (streamSession, error)

// Streaming combines an authenticated loopback proxy with an external player.
type Streaming struct {
	mu       sync.Mutex
	start    streamStarter
	launcher Player
	active   streamSession
}

// NewStreaming creates a player that proxies remote media before launching it.
func NewStreaming(proxy *streaming.Proxy, launcher Player) *Streaming {
	var start streamStarter
	if proxy != nil {
		start = func(ctx context.Context, source *url.URL) (streamSession, error) {
			return proxy.Start(ctx, source)
		}
	}
	return &Streaming{
		start:    start,
		launcher: launcher,
	}
}

// Open starts an authenticated local stream and opens its URL in the external
// player. A previously active stream is closed only after the new launch
// succeeds.
func (p *Streaming) Open(ctx context.Context, source *url.URL) error {
	if p.start == nil || p.launcher == nil {
		return ErrUnavailable
	}
	session, err := p.start(ctx, source)
	if err != nil {
		return fmt.Errorf("start secure media stream: %w", err)
	}
	if session == nil {
		return errors.New("start secure media stream: session is unavailable")
	}
	localURL := session.URL()
	if localURL == nil {
		_ = session.Close()
		return errors.New("start secure media stream: local URL is unavailable")
	}
	if err := p.launcher.Open(ctx, localURL); err != nil {
		_ = session.Close()
		return fmt.Errorf("open external player: %w", err)
	}

	p.mu.Lock()
	previous := p.active
	p.active = session
	p.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return nil
}

// Close stops the active local streaming session.
func (p *Streaming) Close() error {
	p.mu.Lock()
	active := p.active
	p.active = nil
	p.mu.Unlock()
	if active == nil {
		return nil
	}
	return active.Close()
}

var _ Player = (*Streaming)(nil)
