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
- `q`: quit.

The help view must list the shortcuts that are actually implemented. Arrow
keys remain available so that the interface is accessible to users who do not
know Vim.

## Opening media

For a supported video file, `Enter` displays a confirmation before opening it
in IINA. The confirmation clearly identifies the file and allows cancellation
without side effects.

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
