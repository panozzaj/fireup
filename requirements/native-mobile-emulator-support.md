# Native mobile emulator support

## Problem

Hotwire Native dev (and any other native-mobile dev) needs the in-emulator
WebView to reach the Rails app served by fireup. Today this works for the iOS
Simulator (it inherits the Mac's `*.test` resolver) but not for the Android
emulator, which runs its own kernel and DNS resolver.

The Android emulator can reach the host Mac via the magic IP `10.0.2.2`. But
fireup's puma binds to `127.0.0.1:<dynamic-port>` and the proxy on port 80
routes by the `Host:` header. Without a stable, reachable URL that fireup
will route to "the hutch app", the emulator can't easily talk to a
fireup-managed server.

Symptom: launching the native Android shell loads `http://10.0.2.2:3000` and
gets `ERR_CONNECTION_REFUSED`, because nothing is on `:3000`.

## Today's workaround

Run a side-channel `bin/rails server -b 0.0.0.0 -p 3000` alongside fireup.
The Android shell hits `10.0.2.2:3000`; iOS Simulator keeps using
`hutch.test` via fireup. Two puma processes for one app — wasteful and
confusing, and the side-channel server has to be started/stopped manually.

## What would help (rough ideas, not a spec)

In rough order of "fixes the most pain for the least work":

1. **`fireup port <app>`** — print the assigned puma port on stdout. Native
   build scripts could `port=$(fireup port hutch)`, then
   `adb reverse tcp:$port tcp:$port`, and use
   `BASE_URL=http://localhost:$port` in the emulator. Bypasses the proxy,
   uses puma directly. Rails permits `localhost` by default. Solves
   Android — and is also useful for any "what port is X on?" debugging.

2. **`fireup state` JSON dump** — a small JSON snapshot of the running apps:
   `{ apps: { hutch: { web: { port: 51004, status: "running" } } } }`.
   Equivalent to (1) but more discoverable for tools.

3. **Bind the proxy on 0.0.0.0 with an opt-in flag** — a setting like
   `fireup setup --listen 0.0.0.0` so the Android emulator could hit
   `10.0.2.2:80` with `Host: hutch.test`. Combined with a one-time hosts
   entry inside the AVD (`adb root; adb remount; adb push hosts ...`),
   the emulator gets a clean `http://hutch.test` URL too. More setup
   on the user side; less plumbing for fireup.

4. **A `fireup tunnel <app>` subcommand** — allocates a stable port on
   `0.0.0.0` that proxies to the chosen app regardless of Host header.
   E.g. `fireup tunnel hutch --port 3000` → `http://10.0.2.2:3000` from
   the emulator goes straight to hutch's puma. Solves Android without
   any DNS or hosts-file changes. Most user-friendly outcome.

(1) is the thinnest API surface and unblocks the discovery-script pattern
that any tool — native shells, IDE plugins, scripts — could use. The other
three are nicer at the cost of more code in fireup.

## Aside

The same shape of problem exists for any non-localhost-resolving consumer:
Docker containers, VMs, physical phones on the LAN, CI workers running
emulators. A "where is app X listening?" answer + an opt-in 0.0.0.0 bind
covers all of those.
