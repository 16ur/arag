package app

import (
	"context"
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
	if got := model.View().Content; !strings.Contains(got, "Loading WebDAV root") {
		t.Fatalf("loading View() = %q", got)
	}
	model.Update(entriesLoadedMsg{})
	if got := model.View().Content; !strings.Contains(got, "Empty directory") {
		t.Fatalf("empty View() = %q", got)
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
		"up":   tea.KeyUp,
		"down": tea.KeyDown,
	}
	if code, ok := keyCodes[value]; ok {
		return tea.KeyPressMsg(tea.Key{Code: code})
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(value[0]), Text: value})
}
