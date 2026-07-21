# Architecture

The project is divided into five main responsibilities.

## UI

Bubble Tea V2.

Responsibilities:

- rendering;
- adaptive terminal styling;
- navigation;
- keyboard shortcuts;
- loading, confirmation, and error states.

The UI does not know about WebDAV XML.

`View()` only produces a representation of the current state. It does not
start network requests or contain business logic. Bubble Tea commands execute
side effects.

The UI requests the terminal background color during initialization. `Update()`
stores whether that background is light or dark, and `View()` selects the
corresponding presentation styles without performing terminal I/O. Lip Gloss
handles ANSI styling, Unicode-aware widths, and responsive panel composition.

---

## WebDAV

Responsibilities:

- authentication;
- `PROPFIND` requests;
- XML parsing;
- URL and path normalization;
- timeout enforcement.

The client returns Go values.

Navigation uses `Depth: 1`. The client never loads the entire remote tree.

---

## Player

Responsibilities:

- open a URL in an external player;
- adapt the invocation to the selected player.

The player does not know about Bubble Tea.

IINA is the first MVP implementation. The package contract must not depend on
IINA, allowing VLC or another player to be added later.

The IINA launcher accepts only tokenized HTTP URLs using an IP loopback
address and an explicit port. It invokes `/usr/bin/open` directly with
separate arguments and never through a shell. Remote URLs, credentials,
queries, and fragments are rejected before process execution.

The IINA launcher is composed with the streaming proxy by the `Streaming`
player. Authentication remains inside the proxy and is never included in the
URL or arguments passed to IINA. A failed player launch closes the newly
created stream. A successful launch replaces and closes the previous stream.

---

## Streaming proxy

Responsibilities:

- expose one fixed remote media URL through a temporary loopback URL;
- add WebDAV authentication only to upstream requests;
- forward byte-range requests and media response headers;
- prevent credentials from reaching the external player;
- stop the local endpoint when its context is canceled.

The proxy listens only on `127.0.0.1`, uses a random port and a cryptographically
random path token, and accepts only `GET` and `HEAD`. It never acts as a general
forward proxy. Cross-origin upstream redirects are rejected.

The proxy streams response bodies directly and does not load complete media
files into memory. Navigation timeouts do not apply to media bodies.

---

## Configuration

Responsibilities:

- load the URL, preset, username, and player;
- apply default values;
- validate configuration without performing a network request.

Non-sensitive configuration may be stored in a file. Secrets are never written
to it in plain text.

## Main flow

1. Configuration prepares a WebDAV client.
2. A Bubble Tea command requests the contents of the current directory.
3. The client runs `PROPFIND` and returns Go values.
4. `Update()` incorporates the result into the UI state.
5. `View()` renders that state.
6. After a file is confirmed, the streaming proxy creates a temporary local
   URL for the authenticated remote media.
7. The player opens the temporary local URL without receiving WebDAV
   credentials.
8. The temporary endpoint remains available while arag is running and closes
   when it is replaced or the application exits.
