# arag

arag is a terminal WebDAV file browser written in Go with Bubble Tea V2. It
lets users browse a remote server and open media in an external player.

The project aims to work with any standard WebDAV server. Seedhost is the first
supported preset because it is the project's initial use case.

## Project status

arag is under development. It can currently connect to a WebDAV server, load
the root directory, and display it in an interactive terminal interface. The
MVP will also provide:

- connection to a WebDAV server;
- directory browsing without loading the entire tree;
- navigation with the arrow keys or `hjkl`;
- video playback in IINA after confirmation.

IINA is the player targeted by the MVP, but the architecture will support
other players, including VLC.

See the [roadmap](docs/ROADMAP.md) for the planned scope.

## Run arag

Run the current interactive root browser with:

```bash
go run ./cmd/arag \
  -url "https://mud.seedhost.eu/USERNAME/webdav" \
  -user "USERNAME"
```

The password is then requested without being displayed. It remains in memory
only while the command is running and is not stored. Once the interface opens,
use the arrow keys or `j`/`k` to move the selection, `Enter` or `l` to open a
directory, `i` to inspect the complete selected entry, the left arrow or `h`
to return to the parent directory, and `q` to quit.

To display all available options:

```bash
go run ./cmd/arag -h
```

For automated environments, the password can be provided through
`ARAG_PASSWORD`. This is less secure than interactive input and should not be
stored permanently in a configuration file or shell history.
