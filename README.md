# arag

arag is a terminal WebDAV file browser written in Go with Bubble Tea V2. It
lets users browse a remote server and open media in an external player.

The project aims to work with any standard WebDAV server. Seedhost is the first
supported preset because it is the project's initial use case.

## Project status

arag is under development. It can currently connect to a WebDAV server, browse
directories, inspect entries, and open supported videos in IINA through a
temporary authenticated streaming endpoint. Its responsive terminal interface
adapts to light and dark backgrounds and keeps complete metadata available on
both wide and narrow terminals. The MVP provides:

- connection to a WebDAV server;
- directory browsing without loading the entire tree;
- navigation with the arrow keys or `hjkl`;
- video playback in IINA after confirmation.

IINA is the player targeted by the MVP, but the architecture will support
other players, including VLC.

See the [roadmap](docs/ROADMAP.md) for the planned scope.

## Run arag

Launch the interactive connection screen with:

```bash
go run ./cmd/arag
```

Enter the WebDAV URL, username, and password, then select `Connect`. The
password is masked and remains in memory only for the current session. A
successful connection opens the server root in the file browser.

Advanced users and automated environments can still start directly with:

```bash
go run ./cmd/arag \
  -url "https://mud.seedhost.eu/USERNAME/webdav" \
  -user "USERNAME"
```

In direct mode, the password is requested without being displayed. Once the
browser opens, use the arrow keys or `j`/`k` to move the selection, `Enter` or
`l` to open a directory, `i` to inspect the complete selected entry, the left
arrow or `h` to return to the parent directory, and `q` to open the quit
confirmation. `Ctrl+C` remains available for an immediate exit. Pressing
`Enter` or `l` on an MKV or MP4 file opens a confirmation prompt. Confirming
starts a temporary local stream and opens it in IINA. IINA receives neither
the WebDAV password nor the remote media URL. Closing arag also stops the local
stream.

To display all available options:

```bash
go run ./cmd/arag -h
```

For automated environments, the password can be provided through
`ARAG_PASSWORD`. This is less secure than interactive input and should not be
stored permanently in a configuration file or shell history.
