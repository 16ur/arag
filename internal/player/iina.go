package player

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"runtime"
)

const (
	macOSOpenPath = "/usr/bin/open"
	iinaAppName   = "IINA"
)

var (
	// ErrInvalidMediaURL indicates that a player received an unsafe or
	// unsupported media URL.
	ErrInvalidMediaURL = errors.New("invalid media URL")
	// ErrUnsupportedPlatform indicates that IINA cannot run on this operating
	// system.
	ErrUnsupportedPlatform = errors.New("IINA is unsupported on this platform")
)

type commandRunner func(context.Context, string, ...string) error

// IINA opens loopback media URLs in the IINA macOS application.
type IINA struct {
	platform string
	run      commandRunner
}

// NewIINA creates an IINA launcher for the current platform.
func NewIINA() *IINA {
	return &IINA{
		platform: runtime.GOOS,
		run:      runCommand,
	}
}

// Open validates the local stream URL and asks macOS to open it in IINA.
func (i *IINA) Open(ctx context.Context, mediaURL *url.URL) error {
	if err := validateLoopbackMediaURL(mediaURL); err != nil {
		return err
	}
	if i.platform != "darwin" {
		return ErrUnsupportedPlatform
	}
	if ctx == nil {
		return errors.New("launch IINA: context is required")
	}
	if i.run == nil {
		return errors.New("launch IINA: command runner is unavailable")
	}
	if err := i.run(
		ctx,
		macOSOpenPath,
		"-a",
		iinaAppName,
		"-u",
		mediaURL.String(),
	); err != nil {
		return fmt.Errorf("launch IINA: %w", err)
	}
	return nil
}

func validateLoopbackMediaURL(mediaURL *url.URL) error {
	if mediaURL == nil {
		return fmt.Errorf("%w: URL is required", ErrInvalidMediaURL)
	}
	if mediaURL.Scheme != "http" {
		return fmt.Errorf("%w: URL must use HTTP", ErrInvalidMediaURL)
	}
	if mediaURL.User != nil {
		return fmt.Errorf("%w: URL must not contain credentials", ErrInvalidMediaURL)
	}
	address := net.ParseIP(mediaURL.Hostname())
	if address == nil || !address.IsLoopback() {
		return fmt.Errorf("%w: URL must use a loopback address", ErrInvalidMediaURL)
	}
	if mediaURL.Port() == "" {
		return fmt.Errorf("%w: URL requires a port", ErrInvalidMediaURL)
	}
	if mediaURL.Path == "" || mediaURL.Path == "/" {
		return fmt.Errorf("%w: URL requires a stream token", ErrInvalidMediaURL)
	}
	if mediaURL.RawQuery != "" || mediaURL.Fragment != "" {
		return fmt.Errorf("%w: query and fragment are not allowed", ErrInvalidMediaURL)
	}
	return nil
}

func runCommand(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

var _ Player = (*IINA)(nil)
