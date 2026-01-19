# Changelog

All notable changes to this project will be documented in this file.

## [0.1.2] - 2026-01-18

### Added
- Client-side period unstuffing for text-based responses (types 0, 1, 7) per RFC 1436
- Client-side response terminator handling (".\r\n" detection and removal)

### Changed
- Client `Transport.RoundTrip` now wraps text responses in `textReader` for automatic unstuffing

## [0.1.1] - 2026-01-17

### Added
- Server-side period stuffing in `ResponseWriter.WriteText()` per RFC 1436
- Proper response termination with ".\r\n" for text and directory responses
- Expanded `ResponseWriter` interface:
  - `WriteText()` - text (period-stuffed and properly terminated)
  - `WriteBinary()` - raw binary (unterminated)

### Fixed
- Lines beginning with "." are now properly escaped as ".." in server text responses
- Text beginning with "." is now properly escaped as ".." in server text responses
- All text responses now properly terminated with ".\r\n"

## [0.1.0] - 2026-01-16

### Added
- Initial release of net-gopher library
- Gopher client implementation with `Get()` and `Client` type
- Gopher server implementation with `ListenAndServe()` and `Server` type
- `ServeMux` for request routing with prefix matching
- `Handler` and `HandlerFunc` interfaces
- `ResponseWriter` interface with methods:
  - `Write()` - raw bytes
  - `WriteItem()` - single menu item
  - `WriteInfo()` - info line
  - `WriteError()` - error message
  - `WriteDirectory()` - multiple menu items
- Support for all standard Gopher item types (0-9, g, I, h, i, etc.)
- `Request` and `Response` types
- URL parsing via `NewRequest()` and `NewRequestWithContext()`
- Client timeout support
- Transport abstraction with `RoundTripper` interface
- Comprehensive test coverage (83.5%)

[0.1.2]: https://codeberg.org/bryzcolson/net-gopher/compare/v0.1.1...v0.1.2
[0.1.1]: https://codeberg.org/bryzcolson/net-gopher/compare/v0.1.0...v0.1.1
[0.1.0]: https://codeberg.org/bryzcolson/net-gopher/src/tag/v0.1.0