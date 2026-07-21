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
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/16ur/arag/internal/webdav"
	"github.com/charmbracelet/x/term"
)

const passwordEnvironmentVariable = "ARAG_PASSWORD"

type directoryReader interface {
	ReadDir(context.Context, *url.URL) ([]webdav.Entry, error)
}

type clientFactory func(webdav.Config) (directoryReader, error)

type passwordReader func(uintptr) ([]byte, error)

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
	)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", friendlyError(err))
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
) error {
	flags := flag.NewFlagSet("arag", flag.ContinueOnError)
	flags.SetOutput(stderr)

	baseURL := flags.String("url", "", "WebDAV root URL (required)")
	username := flags.String("user", "", "WebDAV username")
	timeout := flags.Duration("timeout", 30*time.Second, "maximum duration of a WebDAV request")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: arag -url URL [-user USERNAME] [-timeout DURATION]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Lists the contents of a WebDAV server root.")
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

	entries, err := client.ReadDir(ctx, nil)
	if err != nil {
		return fmt.Errorf("read WebDAV root: %w", err)
	}
	printEntries(stdout, entries)
	return nil
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

func printEntries(writer io.Writer, entries []webdav.Entry) {
	if len(entries) == 0 {
		fmt.Fprintln(writer, "Empty directory.")
		return
	}

	sort.Slice(entries, func(left, right int) bool {
		if entries[left].IsCollection != entries[right].IsCollection {
			return entries[left].IsCollection
		}
		return strings.ToLower(entries[left].Name) < strings.ToLower(entries[right].Name)
	})

	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	for _, entry := range entries {
		kind := "FILE"
		size := formatSize(entry.Size)
		if entry.IsCollection {
			kind = "DIRECTORY"
			size = "-"
		}
		fmt.Fprintf(table, "%s\t%s\t%s\n", kind, size, entry.Name)
	}
	_ = table.Flush()
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	unit := "B"
	for _, candidate := range units {
		value /= 1024
		unit = candidate
		if value < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

func friendlyError(err error) string {
	switch {
	case errors.Is(err, webdav.ErrAuthentication):
		return "the server rejected the credentials"
	case errors.Is(err, webdav.ErrUnexpectedStatus):
		return "the server did not return a valid WebDAV response"
	case errors.Is(err, webdav.ErrInvalidResponse):
		return "the WebDAV XML response is invalid"
	case errors.Is(err, context.DeadlineExceeded):
		return "the server took too long to respond"
	case errors.Is(err, context.Canceled):
		return "operation canceled"
	default:
		return err.Error()
	}
}
