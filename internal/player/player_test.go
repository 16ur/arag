package player

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

func TestUnavailablePlayer(t *testing.T) {
	t.Parallel()

	mediaURL, err := url.Parse("http://127.0.0.1/video")
	if err != nil {
		t.Fatal(err)
	}
	if err := (Unavailable{}).Open(context.Background(), mediaURL); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Open() error = %v, want ErrUnavailable", err)
	}
}
