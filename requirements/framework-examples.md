# `fireup examples <framework>` Command

## Problem

Common frameworks have gotchas when setting up with fireup — particularly
around allowed hosts. fireup's readiness check probes both IPv4 (`127.0.0.1`)
and IPv6 (`::1`), so frameworks that bind to either work out of the box.
The remaining gotcha is that some frameworks block requests from non-localhost
`Host` headers (e.g. `myapp.test`), which fireup uses for routing.

## Proposed Solution

A `fireup examples <framework>` command that prints a working config snippet + notes for a given framework.

```
$ fireup examples vite

  # ~/.config/fireup/my-vite-app.yml
  description: My Vite App
  root: /path/to/project
  cmd: npx vite --port $PORT

  Notes:
  - Vite 6.4+ blocks non-localhost Host headers by default.
    Add to vite.config.js: server: { allowedHosts: ['my-vite-app.test'] }
```

## Frameworks to Cover

| Framework | Gotcha | Fix |
|---|---|---|
| **Vite** | Blocks non-localhost `Host` headers (v6.4+) | `server: { allowedHosts: ['<app>.test'] }` in vite.config |
| **Next.js** | None | Just works: `next dev --port $PORT` |
| **Rails** | None | Just works: `rails server -p $PORT` |
| **Django** | Blocks non-localhost `Host` headers | Add `<app>.test` to `ALLOWED_HOSTS` in settings |
| **Docsify** | None | Just works: `npx docsify-cli serve . -p $PORT` |
| **Python http.server** | None | Just works: `python -m http.server $PORT` |
| **Astro** | None | Just works: `npx astro dev --port $PORT` |
| **Remix** | Varies by adapter | Check adapter docs |
| **Webpack Dev Server** | Blocks non-localhost `Host` headers | `devServer: { allowedHosts: ['<app>.test'] }` |
| **Esbuild serve** | None | Just works: `esbuild --serve=$PORT` |
| **uvicorn** | None | Just works: `uvicorn app:app --port $PORT` |

## Behavior

- `fireup examples` — list all available frameworks
- `fireup examples vite` — show config + notes for Vite
- `fireup examples rails` — show config + notes for Rails
- Unknown framework — suggest checking `fireup examples` for the full list

## Implementation Notes

- Could be a static lookup table in Go, no external deps needed
- Alternatively, could be markdown files in a `examples/` directory that the binary embeds
- Should include the full yml block so users can copy-paste
