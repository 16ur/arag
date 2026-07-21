// Package player opens remote media in an external video player.
package player

import (
	"context"
	"errors"
	"net/url"
)

// ErrUnavailable indicates that no external player is configured yet.
var ErrUnavailable = errors.New("video player is unavailable")

// Player opens a media URL in an external application.
type Player interface {
	Open(context.Context, *url.URL) error
}

// Unavailable is a placeholder used until a real player is configured.
type Unavailable struct{}

// Open reports that no external player is configured.
func (Unavailable) Open(context.Context, *url.URL) error {
	return ErrUnavailable
}
