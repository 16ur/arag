# ADR 001: Lazy navigation

## Context

A seedbox may contain several thousand files.

## Decision

The TUI never loads the entire directory tree. It only requests the current
directory and its direct children.

---

# ADR 002: WebDAV instead of SSH

## Context

The application needs one reliable protocol for browsing remote files.

## Decision

Use WebDAV exclusively. Never parse HTML directory listings.

---

# ADR 003: Generic WebDAV client with presets

## Context

The core product must work with any standard WebDAV server. A preset such as
Seedhost can simplify provider configuration without adding provider-specific
behavior to the WebDAV client.

## Decision

Keep the WebDAV URL configurable. A preset only supplies default values that
user configuration can override.

---

# ADR 004: Interchangeable external player

## Context

IINA addresses the initial macOS use case but must not limit arag to one
application or platform.

## Decision

Implement IINA for the MVP behind a minimal player contract. VLC and other
players may be added after the MVP.

---

# ADR 005: No plain-text secret storage

## Context

Requesting the password before every playback would significantly harm the
experience, but storing it in the configuration file is unacceptable.

## Decision

For the MVP, request the password once without echo at startup and keep it in
memory for the session only. Automated environments may use an environment
variable. A later version may use the operating system's secure credential
store when the user explicitly enables persistence.

The password must never appear in logs, error messages, or shell history.

---

# ADR 006: Short, configurable network timeouts

## Context

A TUI must report an unavailable server promptly without giving up too early
on a slow seedbox.

## Decision

Use these initial values:

- 10 seconds to establish a connection and receive response headers;
- 30 seconds at most for a navigation `PROPFIND` request;
- immediate cancellation when the user exits or starts a navigation action
  that supersedes the current request.

These values are configurable. Media playback does not use the navigation
timeout because its duration is managed by the external player.
