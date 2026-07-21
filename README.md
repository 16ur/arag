# arag

arag is a terminal WebDAV file browser written in Go with Bubble Tea V2. It
lets users browse a remote server and open media in an external player.

The project aims to work with any standard WebDAV server. Seedhost is the first
supported preset because it is the project's initial use case.

## Project status

arag is under development. The MVP will provide:

- connection to a WebDAV server;
- directory browsing without loading the entire tree;
- navigation with the arrow keys or `hjkl`;
- video playback in IINA after confirmation.

IINA is the player targeted by the MVP, but the architecture will support
other players, including VLC.

See the [roadmap](docs/ROADMAP.md) for the planned scope.

## Test a WebDAV connection

The first functional command lists the contents of a WebDAV root:

```bash
go run ./cmd/arag \
  -url "https://mud.seedhost.eu/USERNAME/webdav" \
  -user "USERNAME"
```

The password is then requested without being displayed. It remains in memory
only while the command is running and is not stored.

To display all available options:

```bash
go run ./cmd/arag -h
```

For automated environments, the password can be provided through
`ARAG_PASSWORD`. This is less secure than interactive input and should not be
stored permanently in a configuration file or shell history.
