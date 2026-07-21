# Project

arag is a WebDAV browser.

## Stack

- Go
- Bubble Tea V2

## Goal

Build a remote file browser.

## Product scope

- Compatible with standard WebDAV servers.
- Seedhost is the first preset, not a client dependency.
- IINA is the MVP player, but the player must remain interchangeable.
- Search and persistent history are outside the MVP scope.

## Architecture

UI -> WebDAV

UI -> Player -> IINA for the MVP

## Constraints

Never parse HTML.

Always use WebDAV.

Never start a network request in `View()`.

Never perform business logic in `View()`.

Prefer small functions.

Always document public packages.

Avoid dependencies when the standard library is sufficient.

Never store or log a secret in plain text.

Important information must never depend on color alone.

Keep arrow-key navigation in addition to `hjkl` shortcuts.

Write code, identifiers, comments, user-facing text, documentation, and commit
messages in English.
