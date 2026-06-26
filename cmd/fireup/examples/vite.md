# Vite

```yaml
description: My Vite App
root: ~/projects/my-vite-app
cmd: npx vite --port $PORT
```

Vite 6.4+ blocks non-localhost Host headers by default.
Add to vite.config.js:

    server: { allowedHosts: ['my-vite-app.test'] }

Or use `allowedHosts: true` to allow all hosts.

If using npm run dev, pass the port through:

    cmd: npm run dev -- --port $PORT
