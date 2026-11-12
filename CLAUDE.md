# zebra-sticker-generator

Go CLI that converts an image (raw bytes, base64, or a data URL) into ASCII art and renders it as a ZPL label for Zebra printers. A UI is planned later, so keep logic in `internal/` packages and keep `cmd/zsticker` a thin wrapper.

## Layout
- `cmd/zsticker` — CLI flags, input/output; logic lives in `run()` for testability.
- `internal/imagein` — detect and decode raw, base64, or data-URL image input.
- `internal/ascii` — image → rows of ASCII characters.
- `internal/zpl` — rows of text → ZPL document.

Validation and defaults belong in the library (e.g. `zpl.NewLabel` returns an error for a too-small label, `ascii.Options.MaxCols` clamps columns) so a future UI or HTTP handler gets them too; the CLI only parses flags and does I/O.

## Commands
- `go vet ./... && go test ./...`
- `go run ./cmd/zsticker -f image.png -ascii -o out.zpl`
- Preview ZPL at https://labelary.com/viewer.html (8 dpmm, 4x6 in), or render it to a PNG to inspect:
  `curl -s --data-binary @out.zpl -H "Accept: image/png" http://api.labelary.com/v1/printers/8dpmm/labels/4x6/0/ > out.png`
- `.claude/settings.json` runs `gofmt -w` on Go files after every Write/Edit.

## ZPL and printing gotchas
- Select font A with a bare `^AAN`. Passing its nominal size (`^AA,N,9,5`) renders at 2x and overflows the label. Native size (5-dot glyph + 1-dot gap = 6-dot pitch, 10-dot line height) is confirmed on a real Zebra printer.
- `ascii.DefaultRamp` is ordered by ink coverage measured per font A glyph in Labelary, not by eye. Re-measure before changing it.
- `zpl.Render` is kept to exactly one allocation (`TestRenderAllocatesOnce`): it grows the builder with `estimatedSize` (an upper bound, not an exact count) and avoids `fmt` and passing the builder as an `io.Writer`, both of which allocate.
- `imagein` checks dimensions with `image.DecodeConfig` and rejects images over 50 megapixels before decoding, since input may come from untrusted users later.
- When changing output, compare old and new ZPL for several flag combinations, e.g. by building the previous commit in a `git worktree`.

## Code style
- Use descriptive variable names; avoid single-letter names, including loop indices, method receivers, and short-lived locals (e.g. `label` not `l`, `row` not `y`, `builder` not `b`).
- Group imports into blocks separated by a blank line, in this order: standard library, third-party modules (e.g. `golang.org/x/...`), then project packages (`github.com/mpierini/zebra-sticker-generator/...`).
- Comment non-obvious concepts and library calls (e.g. what an ASCII ramp is, what a resampling filter does).
- Use American English spelling in code, comments and docs (center, gray, color, neighbor).
