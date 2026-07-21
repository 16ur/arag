# TUI

At startup, arag loads and displays the contents of the configured WebDAV
server. With the Seedhost preset, this is the seedbox WebDAV root.

## Navigation

- up/down arrows or `j`/`k`: move the selection;
- `Enter` or `l`: enter a directory or select a file;
- left arrow, `h`, or `Backspace`: return to the parent directory;
- `i`: show or hide details for the selected entry;
- `Esc`: close the details panel;
- `?`: show or hide help;
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

Pressing `i` opens a details panel for the selected file or directory. The
panel displays:

- the complete name;
- the entry type;
- the file size, when relevant;
- the modification date, when available;
- the ETag, when available;
- the WebDAV path without credentials.

The panel closes with `i` or `Esc`. Navigation keys do not change the
selection while it is open.

## List layout

Directory rows display only their type marker and truncated name. arag does not
calculate directory sizes because that would require recursively loading the
remote tree.

File rows also display their size, aligned to the right of the uniform name
column. Complete names and other metadata remain available in the details
panel.

## Display states

- loading;
- empty directory;
- available contents;
- confirmation prompt;
- entry details;
- recoverable error.

Errors distinguish at least:

- invalid configuration or URL;
- rejected credentials;
- unavailable server or timeout;
- invalid WebDAV response;
- missing player or player launch failure.

Each error suggests an available action: retry, check configuration, enter the
password again, or return to the previous directory.

## Accessibility and layout

- no information depends on color alone;
- the selection also has a textual marker;
- long names are truncated cleanly without breaking the layout;
- small terminals retain essential navigation and messages;
- resizing the terminal does not trigger a network request or lose the
  selection;
- messages remain understandable without unnecessary WebDAV jargon.
