# Backlog

Stack-ranked within each section.

## Routing

- 2026-08-16: Generate the Tailscale Serve port in the docs from the configured
  `HTTPPort` rather than hardcoding it. `docs/fireup.txt` drifted to `:9280`
  while `DefaultHTTPPort = 80` (`cmd/fireup/constants.go:4`), so the documented
  setup command silently produced a serve config pointing at a dead port. The
  text is corrected now, but nothing prevents the same drift again.

- 2026-08-16: Consider making Tailscale access discoverable from the CLI, e.g.
  a `fireup tailscale` command that prints each app's remote URL and warns when
  no serve config is present. `tailscale serve status` returning "No serve
  config" is currently the only signal, and it lives outside fireup. A related
  nicety: show the remote URL next to each app in `fireup status` when a serve
  config exists.

- 2026-08-16: Decide whether `.ts.net` requests on the plain HTTP listener are
  a supported path or an accident. Requests to `http://<tailnet-ip>:80/` with a
  `.ts.net` Host header already route correctly today because the listener binds
  `*:80` — verified end to end with no `tailscale serve` config at all. That
  works, but it is undocumented and gives up HTTPS. Either bless it as the
  zero-setup option or steer people away from it explicitly.

- 2026-08-16: Consider detecting the "serve config points at a port nothing is
  listening on" case. `tailscale serve` accepts any target port without
  validating it, so a misconfigured setup is indistinguishable from a working
  one in `tailscale serve status` output. fireup is well placed to notice that
  it is not receiving the traffic and say so on the dashboard.
