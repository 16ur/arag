# TUI

At startup, arag loads and displays the contents of the configured WebDAV
server. With the Seedhost preset, this is the seedbox WebDAV root.

## Navigation

- up/down arrows or `j`/`k`: move the selection;
- `Enter` or `l`: enter a directory or select a file;
- left arrow, `h`, or `Backspace`: return to the parent directory;
- `?`: show or hide help;
- `q`: quit.

The help view must list the shortcuts that are actually implemented. Arrow
keys remain available so that the interface is accessible to users who do not
know Vim.

## Opening media

For a supported video file, `Enter` displays a confirmation before opening it
in IINA. The confirmation clearly identifies the file and allows cancellation
without side effects.

## Display states

- loading;
- empty directory;
- available contents;
- confirmation prompt;
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
