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
	"github.com/16ur/arag/internal/streaming"
	"github.com/16ur/arag/internal/webdav"
	"github.com/charmbracelet/x/term"
)

const passwordEnvironmentVariable = "ARAG_PASSWORD"

type directoryReader interface {
	ReadDir(context.Context, *url.URL) ([]webdav.Entry, error)
}

type clientFactory func(webdav.Config) (directoryReader, error)

type playerFactory func(string, string) player.Player

type passwordReader func(uintptr) ([]byte, error)

type interfaceRunner func(context.Context, *app.Model, uintptr, io.Writer) error

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
		func(username, password string) player.Player {
			proxy := streaming.NewProxy(streaming.Config{
				Username: username,
				Password: password,
			})
			return player.NewStreaming(proxy, player.NewIINA())
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
	newPlayer playerFactory,
	startInterface interfaceRunner,
) error {
	flags := flag.NewFlagSet("arag", flag.ContinueOnError)
	flags.SetOutput(stderr)

	baseURL := flags.String("url", "", "WebDAV root URL (skips the interactive connection screen)")
	username := flags.String("user", "", "WebDAV username")
	timeout := flags.Duration("timeout", 30*time.Second, "maximum duration of a WebDAV request")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: arag [-url URL] [-user USERNAME] [-timeout DURATION]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Launches an interactive WebDAV connection screen when no URL is provided.")
		fmt.Fprintln(stderr, "With -url, the password is entered without echo and is not stored.")
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
		model := app.NewConnectionModel(
			ctx,
			newSessionFactory(*timeout, newClient, newPlayer),
			app.ConnectionDefaults{Username: *username},
		)
		if err := startInterface(ctx, model, stdinFD, stdout); err != nil {
			return fmt.Errorf("run terminal interface: %w", err)
		}
		return nil
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
	videoPlayer := newPlayer(*username, password)
	if videoPlayer == nil {
		return errors.New("configure video player: player is unavailable")
	}

	model := app.NewModel(ctx, client, videoPlayer)
	if err := startInterface(ctx, model, stdinFD, stdout); err != nil {
		return fmt.Errorf("run terminal interface: %w", err)
	}
	return nil
}

func runInterface(
	ctx context.Context,
	model *app.Model,
	stdinFD uintptr,
	output io.Writer,
) error {
	input := os.NewFile(stdinFD, "stdin")
	program := tea.NewProgram(
		model,
		tea.WithContext(ctx),
		tea.WithInput(input),
		tea.WithOutput(output),
	)
	_, err := program.Run()
	return err
}

func newSessionFactory(
	requestTimeout time.Duration,
	newClient clientFactory,
	newPlayer playerFactory,
) app.SessionFactory {
	return func(ctx context.Context, config app.ConnectionConfig) (app.Session, error) {
		client, err := newClient(webdav.Config{
			BaseURL:        config.BaseURL,
			Username:       config.Username,
			Password:       config.Password,
			RequestTimeout: requestTimeout,
		})
		if err != nil {
			return app.Session{}, fmt.Errorf("invalid WebDAV configuration: %w", err)
		}
		entries, err := client.ReadDir(ctx, nil)
		if err != nil {
			return app.Session{}, err
		}
		videoPlayer := newPlayer(config.Username, config.Password)
		if videoPlayer == nil {
			return app.Session{}, errors.New("configure video player: player is unavailable")
		}
		return app.Session{
			Client:  client,
			Player:  videoPlayer,
			Entries: entries,
		}, nil
	}
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
