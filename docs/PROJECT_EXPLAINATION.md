# arag

arag is a general-purpose WebDAV browser for exploring remote files from a
terminal and quickly opening videos in an external player.

Seedhost is the first provided preset because it is the initial use case. A
preset supplies provider-appropriate defaults without locking the client to
that provider. Custom WebDAV configuration remains available at all times.

## Goals

- Browse files on any compatible WebDAV server.
- Provide a Seedhost preset, followed by other presets as needed.
- Build a TUI with Bubble Tea V2.
- Open videos in an external player.
- Use IINA for the MVP without making it an architectural requirement.
- Provide fluid keyboard navigation with `hjkl` and the arrow keys.

## Product principles

- The product must remain usable without Vim knowledge: arrow keys and
  built-in help are always available.
- Important information must never be communicated through color alone.
- Errors must explain their cause and, when possible, how to resolve them.
- Directory loading must remain incremental and cancelable.
- Passwords must never be stored in plain text.

## Outside the MVP scope

- search;
- persistent navigation history;
- official support for multiple players and platforms;
- persistent secret storage in operating system credential stores.
