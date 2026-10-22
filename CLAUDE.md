# zebra-sticker-generator

Go CLI that converts an image (raw bytes, base64, or a data URL) into ASCII art and renders it as a ZPL label for Zebra printers. A UI is planned later, so keep logic in `internal/` packages and keep `cmd/zsticker` a thin wrapper.

## Layout
- `cmd/zsticker` — CLI flags, input/output; logic lives in `run()` for testability.
- `internal/imagein` — detect and decode raw, base64, or data-URL image input.
- `internal/ascii` — image → rows of ASCII characters.
- `internal/zpl` — rows of text → ZPL document.

## Commands
- `go vet ./... && go test ./...`
- `go run ./cmd/zsticker -f image.png -ascii -o out.zpl`
- Preview ZPL at https://labelary.com/viewer.html (8 dpmm, 4x6 in).

## Code style
- Use descriptive variable names; avoid single-letter names, including loop indices, method receivers, and short-lived locals (e.g. `label` not `l`, `row` not `y`, `builder` not `b`).
- Group imports into blocks separated by a blank line, in this order: standard library, third-party modules (e.g. `golang.org/x/...`), then project packages (`github.com/mpierini/zebra-sticker-generator/...`).
- Comment non-obvious concepts and library calls (e.g. what an ASCII ramp is, what a resampling filter does).
