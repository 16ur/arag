package app

import (
	"context"
	"errors"
	"image/color"
	"net/url"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/16ur/arag/internal/player"
	"github.com/16ur/arag/internal/webdav"
	"github.com/charmbracelet/x/ansi"
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

type fakeVideoPlayer struct {
	openedURL *url.URL
	err       error
	calls     int
}

func (videoPlayer *fakeVideoPlayer) Open(_ context.Context, mediaURL *url.URL) error {
	videoPlayer.calls++
	videoPlayer.openedURL = cloneURL(mediaURL)
	return videoPlayer.err
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
	model := NewModel(context.Background(), reader, player.Unavailable{})

	runInit(t, model)
	result := model
	if reader.calls != 1 {
		t.Fatalf("ReadDir() calls = %d, want 1", reader.calls)
	}
	if len(result.entries) != 2 || result.entries[0].Name != "Movies" {
		t.Fatalf("entries = %+v", result.entries)
	}

	view := viewText(result)
	if !strings.Contains(view, "> Movies/") || !strings.Contains(view, "2.0 KiB") {
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
	model := NewModel(context.Background(), reader, player.Unavailable{})
	runInit(t, model)
	if got := viewText(model); !strings.Contains(got, "server rejected the credentials") {
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

	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	if got := viewText(model); !strings.Contains(got, "Loading directory") {
		t.Fatalf("loading View() = %q", got)
	}
	model.Update(entriesLoadedMsg{})
	if got := viewText(model); !strings.Contains(got, "Empty directory") {
		t.Fatalf("empty View() = %q", got)
	}
}

func TestModelViewDoesNotPerformIO(t *testing.T) {
	t.Parallel()

	reader := &fakeDirectoryReader{}
	model := NewModel(context.Background(), reader, player.Unavailable{})
	_ = model.View()
	_ = model.View()
	if reader.calls != 0 {
		t.Fatalf("View() performed %d directory requests", reader.calls)
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
	model := NewModel(context.Background(), reader, player.Unavailable{})
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
	if !strings.Contains(viewText(model), "/webdav/Movies/") {
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

func TestModelAsksForConfirmationBeforeOpeningVideo(t *testing.T) {
	t.Parallel()

	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{{
		Name: "A Movie.MKV",
		Size: 2 << 30,
	}}})

	_, command := model.Update(key("enter"))
	if command != nil || model.pendingOpen == nil {
		t.Fatalf("confirmation state = pending %v, command %v", model.pendingOpen, command)
	}
	view := viewText(model)
	if !strings.Contains(view, "Open video?") ||
		!strings.Contains(view, "A Movie.MKV") ||
		!strings.Contains(view, "2.0 GiB") {
		t.Fatalf("View() = %q", view)
	}

	model.Update(key("down"))
	if model.selected != 0 {
		t.Fatal("selection moved while confirmation was open")
	}
	model.Update(key("esc"))
	if model.pendingOpen != nil || strings.Contains(viewText(model), "Open video?") {
		t.Fatal("Escape did not cancel confirmation")
	}
}

func TestModelConfirmsVideoThroughPlayer(t *testing.T) {
	t.Parallel()

	mediaURL, err := url.Parse("http://127.0.0.1/video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	videoPlayer := &fakeVideoPlayer{}
	model := NewModel(context.Background(), &fakeDirectoryReader{}, videoPlayer)
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{{Name: "video.mp4", URL: mediaURL}}})
	model.Update(key("enter"))
	_, command := model.Update(key("enter"))
	if command == nil || model.pendingOpen != nil || !model.opening {
		t.Fatalf("confirmation state = pending %v, opening %v, command %v", model.pendingOpen, model.opening, command)
	}
	if !strings.Contains(viewText(model), "Opening video") {
		t.Fatalf("View() = %q", model.View().Content)
	}
	model.Update(command())
	if videoPlayer.calls != 1 || videoPlayer.openedURL.String() != mediaURL.String() {
		t.Fatalf("player calls = %d, URL = %v", videoPlayer.calls, videoPlayer.openedURL)
	}
	if model.opening || !strings.Contains(viewText(model), "✓ Video sent to the player") {
		t.Fatalf("View() = %q", model.View().Content)
	}
}

func TestModelDisplaysPlayerFailure(t *testing.T) {
	t.Parallel()

	mediaURL, err := url.Parse("http://127.0.0.1/video.mkv")
	if err != nil {
		t.Fatal(err)
	}
	videoPlayer := &fakeVideoPlayer{err: errors.New("launch failed")}
	model := NewModel(context.Background(), &fakeDirectoryReader{}, videoPlayer)
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{{Name: "video.mkv", URL: mediaURL}}})
	model.Update(key("enter"))
	_, command := model.Update(key("enter"))
	model.Update(command())
	if model.opening || !strings.Contains(viewText(model), "! Could not open video: launch failed") {
		t.Fatalf("View() = %q", model.View().Content)
	}
}

func TestModelDisplaysUnavailablePlayer(t *testing.T) {
	t.Parallel()

	mediaURL, err := url.Parse("http://127.0.0.1/video.mp4")
	if err != nil {
		t.Fatal(err)
	}
	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{{Name: "video.mp4", URL: mediaURL}}})
	model.Update(key("enter"))
	_, command := model.Update(key("enter"))
	model.Update(command())
	if !strings.Contains(viewText(model), "! Player integration is not available yet") {
		t.Fatalf("View() = %q", model.View().Content)
	}
}

func TestModelRejectsUnsupportedFile(t *testing.T) {
	t.Parallel()

	model := loadedModel("subtitle.srt")
	_, command := model.Update(key("enter"))
	if command != nil || model.pendingOpen != nil || model.opening {
		t.Fatalf("unsupported file changed open state")
	}
	if !strings.Contains(viewText(model), "! Unsupported file type. Only MKV and MP4 videos") {
		t.Fatalf("View() = %q", model.View().Content)
	}
}

func TestIsVideoFile(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"movie.mkv":      true,
		"movie.MP4":      true,
		"movie.mkv.part": false,
		"subtitle.srt":   false,
		"no-extension":   false,
	}
	for name, want := range tests {
		if got := isVideoFile(name); got != want {
			t.Errorf("isVideoFile(%q) = %v, want %v", name, got, want)
		}
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
	model := NewModel(context.Background(), reader, player.Unavailable{})
	staleCommand := model.startLoad(nil, 0)
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
	view := viewText(model)
	if strings.Contains(view, "a-very-long-file-name.mkv") {
		t.Fatalf("long name was not truncated: %q", view)
	}
	if !strings.Contains(view, "> second…") {
		t.Fatalf("selection marker missing: %q", view)
	}
	if strings.Contains(view, "SELECTED") || strings.Contains(view, "│") {
		t.Fatalf("small layout unexpectedly contains the details pane: %q", view)
	}
}

func TestModelUsesSplitLayoutOnWideTerminal(t *testing.T) {
	t.Parallel()

	model := loadedModel("video.mkv")
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	view := viewText(model)
	if !strings.Contains(view, "SELECTED") || !strings.Contains(view, "│") {
		t.Fatalf("wide layout does not contain the details pane: %q", view)
	}
}

func TestModelKeepsRenderedDimensionsStable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		width  int
		height int
		setup  func(*Model)
	}{
		{name: "wide browser", width: 120, height: 20},
		{name: "compact browser", width: 24, height: 8},
		{name: "notification", width: 96, height: 20, setup: func(model *Model) {
			model.notice = "Unsupported file type."
			model.noticeIsError = true
		}},
		{name: "dialog", width: 96, height: 20, setup: func(model *Model) {
			model.confirmQuit = true
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			model := loadedModel("video.mkv")
			model.Update(tea.WindowSizeMsg{Width: test.width, Height: test.height})
			if test.setup != nil {
				test.setup(model)
			}
			lines := strings.Split(model.View().Content, "\n")
			if len(lines) != test.height {
				t.Fatalf("View() height = %d, want %d", len(lines), test.height)
			}
			for index, line := range lines {
				if width := ansi.StringWidth(line); width != test.width {
					t.Fatalf("line %d width = %d, want %d", index, width, test.width)
				}
			}
		})
	}
}

func TestModelUsesRequestedAccentForSelection(t *testing.T) {
	t.Parallel()

	model := loadedModel("video.mkv")
	if !strings.Contains(model.View().Content, "48;2;18;129;130") {
		t.Fatalf("selection does not use accent %s", accentColor)
	}
}

func TestModelAdaptsThemeToTerminalBackground(t *testing.T) {
	t.Parallel()

	model := loadedModel("video.mkv")
	model.Update(tea.BackgroundColorMsg{Color: color.White})
	if model.darkBackground {
		t.Fatal("light terminal background was detected as dark")
	}
	model.Update(tea.BackgroundColorMsg{Color: color.Black})
	if !model.darkBackground {
		t.Fatal("dark terminal background was detected as light")
	}
}

func TestModelTruncatesNamesToUniformMaximumWidth(t *testing.T) {
	t.Parallel()

	longName := strings.Repeat("a", maximumNameWidth+10) + ".mkv"
	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{
		{Name: longName},
		{Name: "short.mkv"},
	}})
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 20})

	view := viewText(model)
	if strings.Contains(view, longName) || !strings.Contains(view, "…") {
		t.Fatalf("name was not truncated: %q", view)
	}
	if model.nameWidth() != maximumNameWidth {
		t.Fatalf("name width = %d, want %d", model.nameWidth(), maximumNameWidth)
	}
}

func TestModelDisplaysSizesForFilesOnly(t *testing.T) {
	t.Parallel()

	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{
		{Name: "Movies", IsCollection: true},
		{Name: "video.mkv", Size: 2048},
	}})
	lines := strings.Split(viewText(model), "\n")
	var directoryLine, fileLine string
	for _, line := range lines {
		if strings.Contains(line, "> Movies/") {
			directoryLine = line
		}
		if strings.Contains(line, "  video.mkv") {
			fileLine = line
		}
	}
	if !strings.Contains(directoryLine, "> Movies/") || strings.Contains(directoryLine, "0 B") {
		t.Fatalf("directory line = %q", directoryLine)
	}
	if !strings.Contains(fileLine, "2.0 KiB") {
		t.Fatalf("file line = %q", fileLine)
	}
}

func TestModelShowsCompleteEntryDetails(t *testing.T) {
	t.Parallel()

	entryURL, err := url.Parse("https://example.com/webdav/Movies/a-very-long-video-name.mkv")
	if err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2026, time.July, 21, 14, 30, 0, 0, time.UTC)
	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{{
		Name:    "a-very-long-video-name.mkv",
		URL:     entryURL,
		Size:    2048,
		ModTime: modified,
		ETag:    "video-tag",
	}}})

	model.Update(key("i"))
	view := viewText(model)
	wanted := []string{
		"Details",
		"Name      a-very-long-video-name.mkv",
		"Type      File",
		"Size      2.0 KiB",
		"Modified  2026-07-21T14:30:00Z",
		"ETag      video-tag",
		"Path      /webdav/Movies/a-very-long-video-name.mkv",
	}
	for _, value := range wanted {
		if !strings.Contains(view, value) {
			t.Errorf("View() does not contain %q:\n%s", value, view)
		}
	}

	model.Update(key("down"))
	if model.selected != 0 {
		t.Fatal("selection moved while details were open")
	}
	model.Update(key("esc"))
	if model.showDetails || strings.Contains(viewText(model), "Details") {
		t.Fatal("Escape did not close details")
	}
}

func TestModelShowsRelevantDirectoryDetails(t *testing.T) {
	t.Parallel()

	directoryURL, err := url.Parse("https://example.com/webdav/Movies/")
	if err != nil {
		t.Fatal(err)
	}
	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	model.Update(entriesLoadedMsg{entries: []webdav.Entry{{
		Name:         "Movies",
		URL:          directoryURL,
		IsCollection: true,
	}}})
	model.Update(key("i"))
	view := viewText(model)
	if !strings.Contains(view, "Type      Directory") ||
		!strings.Contains(view, "Size      Not applicable") ||
		!strings.Contains(view, "Modified  Not available") {
		t.Fatalf("View() = %q", view)
	}
}

func TestModelConfirmsBeforeQuitting(t *testing.T) {
	t.Parallel()

	model := loadedModel("file")
	_, command := model.Update(key("q"))
	if command != nil {
		t.Fatal("q unexpectedly returned a command")
	}
	if !model.confirmQuit || !strings.Contains(viewText(model), "Quit arag?") {
		t.Fatal("q did not open the quit confirmation")
	}
	select {
	case <-model.ctx.Done():
		t.Fatal("q canceled the model context before confirmation")
	default:
	}

	_, command = model.Update(key("enter"))
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

func TestModelCancelsQuitConfirmation(t *testing.T) {
	t.Parallel()

	model := loadedModel("file")
	model.showDetails = true
	model.Update(key("q"))
	_, command := model.Update(key("esc"))
	if command != nil || model.confirmQuit {
		t.Fatal("Escape did not cancel the quit confirmation")
	}
	if !model.showDetails || !strings.Contains(viewText(model), "Details") {
		t.Fatal("canceling quit did not restore the previous view")
	}
	select {
	case <-model.ctx.Done():
		t.Fatal("canceling quit canceled the model context")
	default:
	}
}

func TestModelControlCQuitsImmediately(t *testing.T) {
	t.Parallel()

	model := loadedModel("file")
	_, command := model.Update(key("ctrl+c"))
	if command == nil {
		t.Fatal("Ctrl+C quit command is nil")
	}
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatalf("Ctrl+C command returned %T", command())
	}
	select {
	case <-model.ctx.Done():
	default:
		t.Fatal("Ctrl+C did not cancel the model context")
	}
}

func loadedModel(names ...string) *Model {
	entries := make([]webdav.Entry, len(names))
	for index, name := range names {
		entries[index] = webdav.Entry{Name: name}
	}
	model := NewModel(context.Background(), &fakeDirectoryReader{}, player.Unavailable{})
	model.Update(entriesLoadedMsg{entries: entries})
	return model
}

func runInit(t *testing.T, model *Model) {
	t.Helper()
	message := model.Init()()
	batch, ok := message.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Init() returned %T, want tea.BatchMsg", message)
	}
	for _, command := range batch {
		message := command()
		switch message.(type) {
		case entriesLoadedMsg, loadFailedMsg:
			model.Update(message)
		}
	}
}

func viewText(model *Model) string {
	return ansi.Strip(model.View().Content)
}

func key(value string) tea.KeyPressMsg {
	if value == "ctrl+c" {
		return tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})
	}
	if value == "shift+tab" {
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift})
	}
	keyCodes := map[string]rune{
		"up":        tea.KeyUp,
		"down":      tea.KeyDown,
		"left":      tea.KeyLeft,
		"enter":     tea.KeyEnter,
		"backspace": tea.KeyBackspace,
		"esc":       tea.KeyEscape,
		"tab":       tea.KeyTab,
	}
	if code, ok := keyCodes[value]; ok {
		return tea.KeyPressMsg(tea.Key{Code: code})
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(value[0]), Text: value})
}
