# TUI

Without command-line connection options, arag starts on a WebDAV connection
screen. A successful connection loads and displays the server root.

## Connection

The connection screen contains three fields:

- WebDAV server URL;
- username;
- masked password.

`Tab`, `Shift+Tab`, or the up/down arrows move between controls. `Enter` moves
to the next field and submits only when the `Connect` button is selected.
`Esc` opens the quit confirmation, while `Ctrl+C` exits immediately. The
letter `q` remains available while editing because it may be part of a URL,
username, or password.

Submitting starts authentication and the initial root `PROPFIND` in a Bubble
Tea command. `Esc` cancels an in-progress attempt. Authentication and network
errors remain on the form so the user can correct a value and retry.

After a successful connection, the password value is removed from the form.
The authenticated WebDAV client and streaming player retain the credentials in
memory only for the lifetime of the application. Command-line options remain
available to bypass the form.

## Navigation

- up/down arrows or `j`/`k`: move the selection;
- `Enter` or `l`: enter a directory or select a file;
- left arrow, `h`, or `Backspace`: return to the parent directory;
- `i`: show or hide details for the selected entry;
- `Esc`: close the details dialog;
- `q`: open the quit confirmation;
- `Ctrl+C`: interrupt and quit immediately.

The quit confirmation requires `Enter` to close arag. `Esc` cancels it and
restores the previous view without changing the current navigation state.

The help view must list the shortcuts that are actually implemented. Arrow
keys remain available so that the interface is accessible to users who do not
know Vim.

## Opening media

MKV and MP4 files are recognized as supported videos, regardless of extension
case. For a supported video file, `Enter` or `l` displays a confirmation before
opening it in IINA. The confirmation clearly identifies the complete file name
and size. `Enter` confirms and `Esc` cancels without side effects.

Confirmation dispatches the selected URL through the player contract in a
Bubble Tea command. The production player starts a temporary authenticated
loopback stream and opens its local URL in IINA. The UI displays the player
result when the command completes. Unsupported file types do not open the
confirmation prompt and produce an explanatory message instead.

IINA receives only a random temporary URL bound to `127.0.0.1`. It never
receives the remote WebDAV URL or credentials. The launcher rejects direct
WebDAV URLs and local URLs containing credentials. The active stream closes
when another video replaces it or when arag exits.

## Entry details

On terminals at least 80 columns wide, a right-hand panel displays the selected
entry's essential metadata: complete name, type, file size when relevant, and
modification date. The file list uses 65% of the available width and the
metadata panel uses the remaining 35%.

Pressing `i` opens a centered details dialog for the selected file or
directory. The dialog displays:

- the complete name;
- the entry type;
- the file size, when relevant;
- the modification date, when available;
- the ETag, when available;
- the WebDAV path without credentials.

The dialog closes with `i` or `Esc`. Navigation keys do not change the
selection while it is open.

## List layout

Directory names end with `/` and never display a size. arag does not calculate
directory sizes because that would require recursively loading the remote
tree.

File rows also display their size, aligned to the right of the uniform name
column. The selected row uses a full-width accent background and retains the
`>` marker, so selection never depends on color alone.

Below 80 columns, the metadata panel is hidden automatically and the file list
uses the complete width. Complete metadata remains available with `i`.

## Display states

- connection form;
- connecting;
- loading;
- empty directory;
- available contents;
- confirmation prompt;
- entry details;
- recoverable error.

The header always displays a breadcrumb and a textual connection state.
Successful actions and recoverable action errors appear in a one-line
notification above the shortcut bar. Confirmations use centered dialogs.

Errors distinguish at least:

- invalid configuration or URL;
- rejected credentials;
- unavailable server or timeout;
- invalid WebDAV response;
- missing player or player launch failure.

Each error suggests an available action: retry, check configuration, enter the
password again, or return to the previous directory.

## Accessibility and layout

- the interface keeps the terminal's own background and detects whether it is
  light or dark before choosing secondary text and separator colors;
- `#128182` is the only accent color;
- the interface uses standard characters and does not require a Nerd Font;
- no information depends on color alone;
- the selection also has a textual marker;
- long names are truncated cleanly without breaking the layout;
- small terminals retain essential navigation and messages;
- resizing the terminal does not trigger a network request or lose the
  selection;
- messages remain understandable without unnecessary WebDAV jargon.
