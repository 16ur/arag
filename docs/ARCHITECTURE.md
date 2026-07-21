# Architecture

The project is divided into four main responsibilities.

## UI

Bubble Tea V2.

Responsibilities:

- rendering;
- navigation;
- keyboard shortcuts;
- loading, confirmation, and error states.

The UI does not know about WebDAV XML.

`View()` only produces a representation of the current state. It does not
start network requests or contain business logic. Bubble Tea commands execute
side effects.

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

Passing authentication to the player must not expose credentials in logs,
error messages, or files. This requires a technical validation before the IINA
integration is finalized.

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
6. After a file is confirmed, the player opens it with the selected player.
