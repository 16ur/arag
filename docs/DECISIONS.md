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

---

# ADR 007: Loopback proxy for authenticated playback

## Context

Passing WebDAV credentials in a media URL or player command-line argument can
expose them through process inspection, logs, errors, or player history.
Different external players also support HTTP authentication differently.

## Decision

arag exposes each selected media URL through a temporary HTTP endpoint bound
only to `127.0.0.1`. The endpoint uses a random port and a cryptographically
random token. It adds Basic authentication only when requesting the fixed
remote media URL.

The proxy supports `GET`, `HEAD`, and HTTP byte ranges so that external players
can seek without downloading the complete file. It rejects cross-origin
redirects and never operates as a general-purpose proxy.

External players receive only the temporary loopback URL. They never receive
WebDAV credentials.
