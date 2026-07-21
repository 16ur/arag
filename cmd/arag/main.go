package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/16ur/arag/internal/app"
	"github.com/16ur/arag/internal/player"
	"github.com/16ur/arag/internal/webdav"
	"github.com/charmbracelet/x/term"
)

const passwordEnvironmentVariable = "ARAG_PASSWORD"

type directoryReader interface {
	ReadDir(context.Context, *url.URL) ([]webdav.Entry, error)
}

type clientFactory func(webdav.Config) (directoryReader, error)

type passwordReader func(uintptr) ([]byte, error)

type interfaceRunner func(context.Context, directoryReader, uintptr, io.Writer) error

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := run(
		ctx,
		os.Args[1:],
		os.Stdin.Fd(),
		os.Stdout,
		os.Stderr,
		os.Getenv,
		term.ReadPassword,
		func(config webdav.Config) (directoryReader, error) {
			return webdav.NewClient(config)
		},
		runInterface,
	)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	args []string,
	stdinFD uintptr,
	stdout io.Writer,
	stderr io.Writer,
	getenv func(string) string,
	readPassword passwordReader,
	newClient clientFactory,
	startInterface interfaceRunner,
) error {
	flags := flag.NewFlagSet("arag", flag.ContinueOnError)
	flags.SetOutput(stderr)

	baseURL := flags.String("url", "", "WebDAV root URL (required)")
	username := flags.String("user", "", "WebDAV username")
	timeout := flags.Duration("timeout", 30*time.Second, "maximum duration of a WebDAV request")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: arag -url URL [-user USERNAME] [-timeout DURATION]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Browses the contents of a WebDAV server root.")
		fmt.Fprintln(stderr, "The password is entered without echo and is not stored.")
		fmt.Fprintln(stderr)
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	if strings.TrimSpace(*baseURL) == "" {
		flags.Usage()
		return fmt.Errorf("the -url option is required")
	}

	password, err := getPassword(*username, stdinFD, stderr, getenv, readPassword)
	if err != nil {
		return err
	}

	client, err := newClient(webdav.Config{
		BaseURL:        *baseURL,
		Username:       *username,
		Password:       password,
		RequestTimeout: *timeout,
	})
	if err != nil {
		return fmt.Errorf("invalid WebDAV configuration: %w", err)
	}

	if err := startInterface(ctx, client, stdinFD, stdout); err != nil {
		return fmt.Errorf("run terminal interface: %w", err)
	}
	return nil
}

func runInterface(ctx context.Context, client directoryReader, stdinFD uintptr, output io.Writer) error {
	input := os.NewFile(stdinFD, "stdin")
	program := tea.NewProgram(
		app.NewModel(ctx, client, player.Unavailable{}),
		tea.WithContext(ctx),
		tea.WithInput(input),
		tea.WithOutput(output),
	)
	_, err := program.Run()
	return err
}

func getPassword(
	username string,
	stdinFD uintptr,
	stderr io.Writer,
	getenv func(string) string,
	readPassword passwordReader,
) (string, error) {
	if password := getenv(passwordEnvironmentVariable); password != "" {
		return password, nil
	}
	if username == "" {
		return "", nil
	}

	fmt.Fprint(stderr, "WebDAV password: ")
	password, err := readPassword(stdinFD)
	fmt.Fprintln(stderr)
	if err != nil {
		return "", fmt.Errorf("read password without echo: %w", err)
	}
	return string(password), nil
}
