package app

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/16ur/arag/internal/webdav"
)

type fakeDirectoryReader struct {
	entries []webdav.Entry
	err     error
	calls   int
}

type navigationReader struct {
	entriesByPath map[string][]webdav.Entry
	requests      []string
	contexts      []context.Context
}

func (reader *navigationReader) ReadDir(ctx context.Context, directory *url.URL) ([]webdav.Entry, error) {
	path := "/"
	if directory != nil {
		path = directory.Path
	}
	reader.requests = append(reader.requests, path)
	reader.contexts = append(reader.contexts, ctx)
	return reader.entriesByPath[path], nil
}

func (reader *fakeDirectoryReader) ReadDir(context.Context, *url.URL) ([]webdav.Entry, error) {
	reader.calls++
	return reader.entries, reader.err
}

func TestModelLoadsAndSortsRootEntries(t *testing.T) {
	t.Parallel()

	reader := &fakeDirectoryReader{entries: []webdav.Entry{
		{Name: "video.mkv", Size: 2048},
		{Name: "Movies", IsCollection: true},
	}}
	model := NewModel(context.Background(), reader)

	msg := model.Init()()
	updated, command := model.Update(msg)
	if command != nil {
		t.Fatal("Update() command is not nil")
	}
	result := updated.(*Model)
	if reader.calls != 1 {
		t.Fatalf("ReadDir() calls = %d, want 1", reader.calls)
	}
	if len(result.entries) != 2 || result.entries[0].Name != "Movies" {
		t.Fatalf("entries = %+v", result.entries)
	}

	view := result.View().Content
	if !strings.Contains(view, "> [D]") || !strings.Contains(view, "2.0 KiB") {
		t.Errorf("View() = %q", view)
	}
}

func TestModelMovesSelectionWithArrowsAndVimKeys(t *testing.T) {
	t.Parallel()

	model := loadedModel("one", "two", "three")
	model.Update(key("down"))
	model.Update(key("j"))
	model.Update(key("down"))
	if model.selected != 2 {
		t.Fatalf("selected = %d, want 2", model.selected)
	}
	model.Update(key("up"))
	model.Update(key("k"))
	model.Update(key("up"))
	if model.selected != 0 {
		t.Fatalf("selected = %d, want 0", model.selected)
	}
}

func TestModelDisplaysAndRetriesError(t *testing.T) {
	t.Parallel()

	reader := &fakeDirectoryReader{err: webdav.ErrAuthentication}
	model := NewModel(context.Background(), reader)
	model.Update(model.Init()())
	if got := model.View().Content; !strings.Contains(got, "server rejected the credentials") {
		t.Fatalf("View() = %q", got)
	}

	reader.err = nil
	reader.entries = []webdav.Entry{{Name: "Movies", IsCollection: true}}
	_, command := model.Update(key("r"))
	if command == nil || !model.loading {
		t.Fatal("retry did not start loading")
	}
	model.Update(command())
	if model.loading || len(model.entries) != 1 || reader.calls != 2 {
		t.Fatalf("model after retry = %+v, calls = %d", model, reader.calls)
	}
}

func TestModelDisplaysLoadingAndEmptyStates(t *testing.T) {
	t.Parallel()

	model := NewModel(context.Background(), &fakeDirectoryReader{})
	if got := model.View().Content; !strings.Contains(got, "Loading directory") {
		t.Fatalf("loading View() = %q", got)
	}
	model.Update(entriesLoadedMsg{})
	if got := model.View().Content; !strings.Contains(got, "Empty directory") {
		t.Fatalf("empty View() = %q", got)
	}
}

func TestModelNavigatesIntoDirectoryAndBack(t *testing.T) {
	t.Parallel()

	moviesURL, err := url.Parse("https://example.com/webdav/Movies/")
	if err != nil {
		t.Fatal(err)
	}
	reader := &navigationReader{entriesByPath: map[string][]webdav.Entry{
		"/webdav/Movies/": {{Name: "video.mkv", Size: 1024}},
		"/": {
			{Name: "Archive", IsCollection: true},
			{Name: "Movies", URL: moviesURL, IsCollection: true},
		},
	}}
	model := NewModel(context.Background(), reader)
	model.Update(entriesLoadedMsg{entries: reader.entriesByPath["/"]})
	model.Update(key("down"))

	_, command := model.Update(key("enter"))
	if command == nil || !model.loading {
		t.Fatal("enter did not start directory loading")
	}
	model.Update(command())
	if model.currentDirectory.String() != moviesURL.String() {
		t.Fatalf("current directory = %v", model.currentDirectory)
	}
	if len(model.entries) != 1 || model.entries[0].Name != "video.mkv" {
		t.Fatalf("child entries = %+v", model.entries)
	}
	if !strings.Contains(model.View().Content, "Location: /webdav/Movies/") {
		t.Fatalf("View() = %q", model.View().Content)
	}

	_, command = model.Update(key("left"))
	if command == nil {
		t.Fatal("left did not start parent loading")
	}
	model.Update(command())
	if model.currentDirectory != nil {
		t.Fatalf("current directory = %v, want root", model.currentDirectory)
	}
	if model.selected != 1 {
		t.Fatalf("selected = %d, want restored selection 1", model.selected)
	}
	if got := strings.Join(reader.requests, ","); got != "/webdav/Movies/,/" {
		t.Fatalf("requests = %q", got)
	}
}

func TestModelDoesNotOpenFile(t *testing.T) {
	t.Parallel()

	model := loadedModel("video.mkv")
	_, command := model.Update(key("enter"))
	if command != nil || len(model.history) != 0 {
		t.Fatalf("file opened: command = %v, history = %+v", command, model.history)
	}
}

func TestModelIgnoresStaleResponseAndCancelsPreviousRequest(t *testing.T) {
	t.Parallel()

	directory, err := url.Parse("https://example.com/webdav/Movies/")
	if err != nil {
		t.Fatal(err)
	}
	reader := &navigationReader{entriesByPath: map[string][]webdav.Entry{
		"/":               {{Name: "stale.mkv"}},
		"/webdav/Movies/": {{Name: "current.mkv"}},
	}}
	model := NewModel(context.Background(), reader)
	staleCommand := model.Init()
	currentCommand := model.startLoad(directory, 0)

	staleMsg := staleCommand()
	if !errors.Is(reader.contexts[0].Err(), context.Canceled) {
		t.Fatal("previous request context was not canceled")
	}
	model.Update(staleMsg)
	if len(model.entries) != 0 || !model.loading {
		t.Fatalf("stale response changed model: %+v", model.entries)
	}

	model.Update(currentCommand())
	if len(model.entries) != 1 || model.entries[0].Name != "current.mkv" {
		t.Fatalf("entries = %+v", model.entries)
	}
}

func TestModelAdaptsToSmallTerminal(t *testing.T) {
	t.Parallel()

	model := loadedModel("a-very-long-file-name.mkv", "second.mkv")
	model.Update(tea.WindowSizeMsg{Width: 20, Height: 6})
	model.Update(key("down"))
	view := model.View().Content
	if strings.Contains(view, "a-very-long-file-name.mkv") {
		t.Fatalf("long name was not truncated: %q", view)
	}
	if !strings.Contains(view, "> [F]") {
		t.Fatalf("selection marker missing: %q", view)
	}
}

func TestModelQuits(t *testing.T) {
	t.Parallel()

	model := loadedModel("file")
	_, command := model.Update(key("q"))
	if command == nil {
		t.Fatal("quit command is nil")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatalf("quit command returned %T", command())
	}
	select {
	case <-model.ctx.Done():
	default:
		t.Fatal("quit did not cancel the model context")
	}
}

func loadedModel(names ...string) *Model {
	entries := make([]webdav.Entry, len(names))
	for index, name := range names {
		entries[index] = webdav.Entry{Name: name}
	}
	model := NewModel(context.Background(), &fakeDirectoryReader{})
	model.Update(entriesLoadedMsg{entries: entries})
	return model
}

func key(value string) tea.KeyPressMsg {
	keyCodes := map[string]rune{
		"up":        tea.KeyUp,
		"down":      tea.KeyDown,
		"left":      tea.KeyLeft,
		"enter":     tea.KeyEnter,
		"backspace": tea.KeyBackspace,
	}
	if code, ok := keyCodes[value]; ok {
		return tea.KeyPressMsg(tea.Key{Code: code})
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(value[0]), Text: value})
}
