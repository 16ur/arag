# Contributing

Always prefer simplicity.

Do not introduce premature abstractions.

Do not add unnecessary dependencies.

Give each package a clear responsibility.

Avoid functions longer than 100 lines.

Prefer explicit names.

Write code, identifiers, comments, user-facing text, documentation, and commit
messages in English.

## Product constraints

- preserve compatibility with generic WebDAV servers;
- implement provider-specific behavior as presets;
- never store or log a secret in plain text;
- never start network requests or perform business logic in `View()`;
- accompany any new WebDAV parsing behavior with XML tests;
- keep arrow-key navigation when adding `hjkl` shortcuts;
- do not communicate information through color alone.
