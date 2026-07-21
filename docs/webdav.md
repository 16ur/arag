# WebDAV

## URL

The URL is configurable to support any standard WebDAV server.

Seedhost preset:

`https://mud.seedhost.eu/<user>/webdav`

The preset supplies a URL template but does not change the protocol or parsing
behavior.

## Authentication

HTTP Basic.

For the MVP, the password is requested without echo at startup and kept in
memory for the session only. It does not need to be entered again for every
directory or video.

The `ARAG_PASSWORD` environment variable can provide the secret in automated
environments. It must not be displayed, logged, or stored permanently in a
configuration file.

The configuration file never contains the password in plain text. Storage in
the operating system's secure credential store is planned after the MVP.

## Navigation

Method: `PROPFIND`.

Header: `Depth: 1`.

Expected response: `207 Multi-Status`.

Directories are identified by `<D:collection/>`.

Files may provide:

- `getcontentlength`;
- `creationdate`;
- `getlastmodified`, when available;
- `getetag`.

The parser uses XML namespaces and does not depend on the `D` prefix.
Properties may be missing. The client ignores the entry representing the
requested directory itself and returns Go values to the UI.

The client never parses HTML or falls back to an HTML directory listing.

## Network behavior

Proposed default values:

- 10 seconds for connection establishment and response headers;
- 30 seconds to complete a navigation `PROPFIND` request.

A request is canceled when the user exits or when a newer navigation action
makes its result obsolete. Timeout errors are presented separately from
authentication failures and invalid XML responses.

These limits are configurable. They apply to WebDAV navigation, not to media
playback duration in the external player.
