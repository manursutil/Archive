# Architecture

Archive is two programs that talk over HTTP on the loopback interface:

- **`archive`** (Go): the CLI and HTTP API. It owns the SQLite database and also serves the UI, which is embedded in the binary.
- **`archive-desktop`** (Deno): a thin desktop shell. It starts `archive serve`, opens a native webview window pointed at it, and stops the server when the window closes.

```
 ┌──────────────── archive-desktop (Deno) ────────────────┐
 │                                                         │
 │  1. spawns child process:  archive serve                │
 │  2. waits until http://localhost:8080 responds          │
 │  3. opens a native window pointing at that URL          │
 │  4. when the window closes → kills the Go process       │
 │                                                         │
 │   ┌─────────────── native window (webview) ──────────┐  │
 │   │  ui/dist (Svelte build)                          │  │
 │   │     fetch("/items")  fetch("/search?q=...")      │  │
 │   └─────────────────────────┬────────────────────────┘  │
 └─────────────────────────────┼───────────────────────────┘
                               │ HTTP over loopback (127.0.0.1)
                               │ (never leaves the machine)
 ┌─────────────────────────────▼───────────────────────────┐
 │  archive serve (Go)                                     │
 │     GET  /            → ui/ (embedded in binary)        │
 │     GET  /items ...   → JSON API                        │
 │                  │                                      │
 │     <UserConfigDir>/archive/archive.db (SQLite + FTS5)  │
 └─────────────────────────────────────────────────────────┘
```

## Components

| Path              | Role                                                                  |
| ----------------- | --------------------------------------------------------------------- |
| `main.go`         | Entry point, DB path resolution                                        |
| `cli.go`          | Command dispatch (`add`, `list`, `del`, `search`, `serve`)             |
| `item.go`         | Fetches URLs and reads files, then extracts plain text                 |
| `db.go`           | SQLite schema, FTS5 index, queries                                     |
| `server.go`       | HTTP API, embedded UI, localhost-only middleware                       |
| `ui/`             | Svelte + Vite frontend; `go generate` builds `ui/dist`, embedded with `go:embed` |
| `desktop/main.ts` | Deno shell: spawns the server and opens the webview                    |
| `desktop/menu.ts` | macOS menu bar (Edit + Quit) so standard shortcuts work                |

## HTTP API

Served on `localhost:8080` by `archive serve`.

| Method | Path          | Description                                  |
| ------ | ------------- | -------------------------------------------- |
| GET    | `/`           | The UI                                       |
| GET    | `/items`      | List saved items (JSON)                      |
| POST   | `/items`      | Add an item, body `{"source": "<url\|path>"}` |
| DELETE | `/items/{id}` | Delete an item                               |
| GET    | `/items/{id}/html` | Saved copy of a web page (`404` for files) |
| GET    | `/items/{id}/text` | Extracted plain text (`text/plain`)      |
| GET    | `/search?q=`  | Full-text search (JSON)                      |

## Database location

The database lives in the OS user config directory, so the CLI and the desktop app share one database no matter where they're run from:

- macOS: `~/Library/Application Support/archive/archive.db`
- Linux: `~/.config/archive/archive.db`
- Windows: `%AppData%\archive\archive.db`

Set `ARCHIVE_DB=/some/path.db` to override it. The tests use this.

## Why Go serves the UI

If Deno served the page on a different port, the page and the API would be different origins. The API would then need CORS headers, and `Access-Control-Allow-Origin: *` would let any website in your browser read your archive. When Go serves both, they share one origin and no CORS headers are needed.

## Localhost is not automatically safe

Any page open in your regular browser can send requests to `localhost:8080`. The `localOnly` middleware in `server.go` blocks two attacks:

1. **Cross-site POST.** A malicious page could submit a form that makes the server ingest a local file. POSTs must have `Content-Type: application/json`. Browsers can't send that cross-origin without a CORS preflight, which the server never approves. Other POSTs get `415`.
2. **DNS rebinding.** An attacker's domain could resolve to `127.0.0.1`. Requests whose `Host` header isn't `localhost:…` or `127.0.0.1:…` get `403`.

## Viewing saved pages

A saved page is untrusted HTML. Served as-is from `localhost:8080`, its scripts would share the UI's origin and could call the API. `GET /items/{id}/html` sends `Content-Security-Policy: sandbox`, which runs the page in an opaque origin with scripts and forms disabled. This holds however the page is opened, so the UI's preview pane points a sandboxed iframe at it. Files have no HTML, so the pane shows `/items/{id}/text` instead.

When a page is saved, its images and stylesheets (including `@import`s and CSS `url()`s such as fonts and backgrounds) are downloaded and embedded as `data:` URIs, so the copy renders offline. `srcset` is renamed to `data-srcset` so the browser uses the embedded `src`. Assets that fail to download keep their absolute URL. Scripts aren't saved: the sandbox would block them anyway.

## Webview

The desktop window uses [`@webview/webview`](https://jsr.io/@webview/webview), which binds the C [webview](https://github.com/webview/webview) library through Deno FFI. It uses the browser engine the OS already ships (WebKit on macOS, WebView2 on Windows, WebKitGTK on Linux), so no Chromium is bundled. On first run it downloads its native library (`libwebview.*`) and caches it. After that it works offline.

The UI's "open" button calls `window.openExternal(url)`, which `archive-desktop` binds to the system browser (`open`, `xdg-open` or `explorer`). It accepts only `http(s)` URLs. In a plain browser the UI falls back to `window.open`.

The add page's "choose file" button calls `window.pickFile()`, bound to a native open dialog (`osascript` on macOS, `zenity` on Linux) that returns an absolute path. Browsers never expose a file's path, so the button only appears in `archive-desktop`. The dialog runs synchronously: `webview.run()` blocks Deno's event loop, so an async binding would never resolve.

On macOS, ⌘C/⌘V/⌘Z/⌘Q only work through the app's menu bar, and webview creates none. `desktop/menu.ts` builds an app menu and an Edit menu through the Objective-C runtime (Deno FFI) before the window runs. Quit closes the window instead of calling `terminate:`, so `main.ts` still gets to stop the server.
