# MVP

- [ ] custom WebDAV URL configuration
- [ ] Seedhost preset
- [ ] HTTP Basic authentication with hidden input once per session
- [ ] WebDAV connection with timeout and cancellation
- [ ] parse `PROPFIND` XML responses
- [ ] normalize URLs and paths returned by the server
- [ ] lazy navigation with `Depth: 1`
- [ ] arrow-key and `hjkl` navigation
- [ ] return to the parent directory
- [ ] loading, empty, and error states
- [ ] built-in help and discoverable shortcuts
- [ ] layout suitable for small terminals and long names
- [ ] confirmation before opening a video
- [ ] open videos in IINA without exposing credentials
- [ ] unit tests for the XML parser
- [ ] integration tests with a local fake WebDAV server
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

- [ ] no network requests or business logic in `View()`
- [ ] no secrets in files, logs, or error messages
- [ ] no information communicated through color alone
- [ ] distinct errors for invalid URLs, rejected authentication, unavailable
      servers, and invalid WebDAV responses
- [ ] navigation usable without knowing Vim shortcuts
