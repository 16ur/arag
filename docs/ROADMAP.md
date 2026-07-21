# MVP

- [x] custom WebDAV URL configuration
- [x] interactive WebDAV connection screen
- [ ] Seedhost preset
- [x] HTTP Basic authentication with hidden input once per session
- [x] WebDAV connection with timeout and cancellation
- [x] parse `PROPFIND` XML responses
- [x] normalize URLs and paths returned by the server
- [x] lazy navigation with `Depth: 1`
- [x] arrow-key and `hjkl` navigation
- [x] return to the parent directory
- [x] loading, empty, and error states
- [ ] built-in help and discoverable shortcuts
- [x] layout suitable for small terminals and long names
- [x] details panel for the selected file or directory
- [x] confirmation before opening a video
- [x] player contract connected to the confirmation flow
- [x] authenticated loopback streaming proxy with byte-range support
- [x] validated and testable IINA loopback URL launcher
- [x] connect confirmed videos to IINA without exposing credentials
- [ ] validate playback and seeking with IINA against a real WebDAV server
- [x] unit tests for the XML parser
- [x] integration tests with a local fake WebDAV server
- [ ] installation and configuration documentation

---

# V1

- [ ] external player selection
- [ ] VLC support
- [ ] optional storage in the operating system's secure credential store
- [ ] additional WebDAV presets
- [ ] prebuilt binaries for supported platforms
- [ ] search within the current directory
- [ ] persistent navigation history

## Quality criteria

- [x] no network requests or business logic in `View()`
- [ ] no secrets in files, logs, or error messages
- [x] no information communicated through color alone
- [ ] distinct errors for invalid URLs, rejected authentication, unavailable
      servers, and invalid WebDAV responses
- [x] navigation usable without knowing Vim shortcuts
