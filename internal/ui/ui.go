package ui

import (
	"fmt"
	"net/http"
)

// ServeIndex serves the main dashboard HTML
func ServeIndex(w http.ResponseWriter, r *http.Request, tld string, port int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, indexHTML, tld, port, tld)
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>roost-dev</title>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: #1a1a2e;
            color: #eee;
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
        }
        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 1px solid #333;
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
            color: #fff;
        }
        .actions button {
            background: #333;
            color: #fff;
            border: none;
            padding: 8px 16px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
            margin-left: 8px;
        }
        .actions button:hover {
            background: #444;
        }
        .app {
            background: #16213e;
            border-radius: 8px;
            margin-bottom: 12px;
            overflow: hidden;
        }
        .app-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 20px;
            cursor: pointer;
        }
        .app-header:hover {
            background: #1a2744;
        }
        .app-info {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .status-dot {
            width: 10px;
            height: 10px;
            border-radius: 50%%;
            background: #6b7280;
            cursor: pointer;
            transition: transform 0.1s;
        }
        .status-dot:hover {
            transform: scale(1.3);
        }
        .status-dot.running {
            background: #22c55e;
        }
        .status-dot.failed {
            background: #ef4444;
        }
        .status-dot.idle {
            background: #6b7280;
        }
        .status-dot.starting {
            background: #f59e0b;
            animation: pulse 1s ease-in-out infinite;
        }
        @keyframes pulse {
            0%%, 100%% { opacity: 1; transform: scale(1); }
            50%% { opacity: 0.5; transform: scale(1.2); }
        }
        .app-description {
            font-size: 13px;
            color: #9ca3af;
            margin-left: 4px;
        }
        .external-link {
            display: inline-flex;
            align-items: center;
            gap: 4px;
        }
        .external-link svg {
            width: 12px;
            height: 12px;
            opacity: 0.7;
        }
        .external-link:hover svg {
            opacity: 1;
        }
        .app-name {
            font-weight: 600;
            font-size: 16px;
        }
        .app-type {
            font-size: 12px;
            color: #d1d5db;
            background: #374151;
            padding: 2px 8px;
            border-radius: 4px;
        }
        .app-url {
            color: #60a5fa;
            text-decoration: none;
            font-size: 14px;
        }
        .app-url:hover {
            text-decoration: underline;
            color: #93c5fd;
        }
        .app-meta {
            display: flex;
            align-items: center;
            gap: 16px;
        }
        .app-port {
            font-size: 14px;
            color: #9ca3af;
        }
        .app-actions {
            display: flex;
            gap: 8px;
        }
        .app-actions button {
            background: transparent;
            border: 1px solid #4b5563;
            color: #d1d5db;
            padding: 4px 12px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
        }
        .app-actions button:hover {
            background: #374151;
            color: #fff;
        }
        .app-actions button.stop {
            border-color: #ef4444;
            color: #ef4444;
        }
        .app-actions button.stop:hover {
            background: #ef4444;
            color: #fff;
        }
        .services {
            padding: 0 20px 16px 42px;
        }
        .service {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px 12px;
            background: #1a2744;
            border-radius: 4px;
            margin-top: 8px;
        }
        .service-info {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .service-name {
            font-size: 14px;
            color: #e5e7eb;
        }
        .app-error {
            font-size: 12px;
            color: #f87171;
            display: block;
            margin-top: 4px;
            padding: 4px 8px;
            background: rgba(239, 68, 68, 0.1);
            border-radius: 4px;
            max-width: 500px;
        }
        .logs-panel {
            background: #0f0f1a;
            border-top: 1px solid #333;
            padding: 16px 20px;
            display: none;
        }
        .logs-panel.visible {
            display: block;
        }
        .logs-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 12px;
        }
        .logs-title {
            font-size: 14px;
            color: #9ca3af;
        }
        .logs-content {
            font-family: "SF Mono", Monaco, "Cascadia Code", monospace;
            font-size: 12px;
            line-height: 1.6;
            max-height: 300px;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-all;
            color: #d1d5db;
        }
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #9ca3af;
        }
        .empty-state h2 {
            font-size: 18px;
            margin-bottom: 12px;
            color: #d1d5db;
        }
        .empty-state code {
            display: block;
            background: #16213e;
            padding: 16px;
            border-radius: 6px;
            margin-top: 16px;
            font-family: "SF Mono", Monaco, monospace;
            font-size: 13px;
            color: #7c3aed;
            text-align: left;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>roost-dev</h1>
            <div class="actions">
                <button onclick="reload()">Reload Config</button>
            </div>
        </header>
        <main id="apps"></main>
    </div>

    <script>
        const TLD = '%s';
        const PORT = %d;
        const portSuffix = PORT === 80 ? '' : ':' + PORT;
        let selectedApp = null;

        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                const apps = await res.json();
                renderApps(apps || []);
            } catch (e) {
                console.error('Failed to fetch status:', e);
            }
        }

        function renderApps(apps) {
            const container = document.getElementById('apps');

            if (!apps.length) {
                container.innerHTML = ` + "`" + `
                    <div class="empty-state">
                        <h2>No apps configured</h2>
                        <p>Add config files to ~/.config/roost-dev/</p>
                        <code># Simple port proxy
echo "3000" > ~/.config/roost-dev/myapp

# Command (auto-starts with PORT env)
echo "npm run dev" > ~/.config/roost-dev/myapp

# Then visit http://myapp.%s</code>
                    </div>
                ` + "`" + `;
                return;
            }

            const externalLinkIcon = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.25 5.5a.75.75 0 00-.75.75v8.5c0 .414.336.75.75.75h8.5a.75.75 0 00.75-.75v-4a.75.75 0 011.5 0v4A2.25 2.25 0 0112.75 17h-8.5A2.25 2.25 0 012 14.75v-8.5A2.25 2.25 0 014.25 4h5a.75.75 0 010 1.5h-5z" clip-rule="evenodd" /><path fill-rule="evenodd" d="M6.194 12.753a.75.75 0 001.06.053L16.5 4.44v2.81a.75.75 0 001.5 0v-4.5a.75.75 0 00-.75-.75h-4.5a.75.75 0 000 1.5h2.553l-9.056 8.194a.75.75 0 00-.053 1.06z" clip-rule="evenodd" /></svg>';

            container.innerHTML = apps.map(app => {
                const isRunning = app.running || (app.services && app.services.some(s => s.running));
                const hasFailed = app.failed || (app.services && app.services.some(s => s.failed));
                const statusClass = hasFailed ? 'failed' : (isRunning ? 'running' : 'idle');
                const displayName = app.description || app.name;

                const getServiceStatus = (svc) => svc.failed ? 'failed' : (svc.running ? 'running' : 'idle');

                let servicesHTML = '';
                if (app.services && app.services.length > 0) {
                    servicesHTML = ` + "`" + `
                        <div class="services">
                            ${app.services.map(svc => {
                                const svcStatus = getServiceStatus(svc);
                                const svcName = svc.name + '-' + app.name;
                                return ` + "`" + `
                                <div class="service">
                                    <div class="service-info">
                                        <div class="status-dot ${svcStatus}" onclick="event.stopPropagation(); toggleProcess('${svcName}', '${svcStatus}', event)" title="${svcStatus === 'running' ? 'Click to stop' : svcStatus === 'failed' ? 'Click to restart' : 'Click to start'}"></div>
                                        <span class="service-name">${svc.name}</span>
                                        ${svc.port ? ` + "`" + `<span class="app-port">:${svc.port}</span>` + "`" + ` : ''}
                                        ${svc.error ? ` + "`" + `<span class="app-error">${svc.error}</span>` + "`" + ` : ''}
                                    </div>
                                    <a class="app-url external-link" href="http://${svc.name}-${app.name}.${TLD}${portSuffix}" target="_blank">
                                        ${svc.name}-${app.name}.${TLD} ${externalLinkIcon}
                                    </a>
                                </div>
                            ` + "`" + `}).join('')}
                        </div>
                    ` + "`" + `;
                }

                return ` + "`" + `
                    <div class="app" data-name="${app.name}">
                        <div class="app-header" onclick="toggleLogs('${app.name}')">
                            <div class="app-info">
                                <div class="status-dot ${statusClass}" onclick="event.stopPropagation(); toggleProcess('${app.name}', '${statusClass}', event)" title="${statusClass === 'running' ? 'Click to stop' : statusClass === 'failed' ? 'Click to restart' : 'Click to start'}"></div>
                                <span class="app-name">${displayName}</span>
                                ${app.description ? ` + "`" + `<span class="app-description">(${app.name})</span>` + "`" + ` : ''}
                                <span class="app-type">${app.type}</span>
                            </div>
                            <div class="app-meta">
                                ${app.port ? ` + "`" + `<span class="app-port">:${app.port}</span>` + "`" + ` : ''}
                                ${app.uptime ? ` + "`" + `<span class="app-port">${app.uptime}</span>` + "`" + ` : ''}
                                <a class="app-url external-link" href="${app.url}" target="_blank" onclick="event.stopPropagation()">
                                    ${app.name}.${TLD} ${externalLinkIcon}
                                </a>
                                <div class="app-actions">
                                    ${app.running ? ` + "`" + `
                                        <button onclick="event.stopPropagation(); restart('${app.name}')">restart</button>
                                        <button class="stop" onclick="event.stopPropagation(); stop('${app.name}')">stop</button>
                                    ` + "`" + ` : ''}
                                </div>
                            </div>
                        </div>
                        ${servicesHTML}
                        <div class="logs-panel" id="logs-${app.name}">
                            <div class="logs-header">
                                <span class="logs-title">Logs</span>
                                <button onclick="clearLogs('${app.name}')">Clear</button>
                            </div>
                            <div class="logs-content" id="logs-content-${app.name}"></div>
                        </div>
                    </div>
                ` + "`" + `;
            }).join('');
        }

        async function toggleLogs(name) {
            const panel = document.getElementById('logs-' + name);
            const isVisible = panel.classList.contains('visible');

            // Hide all panels
            document.querySelectorAll('.logs-panel').forEach(p => p.classList.remove('visible'));

            if (!isVisible) {
                panel.classList.add('visible');
                await fetchLogs(name);
            }
        }

        async function fetchLogs(name) {
            try {
                const res = await fetch('/api/logs?name=' + encodeURIComponent(name));
                const lines = await res.json();
                const content = document.getElementById('logs-content-' + name);
                content.textContent = (lines || []).join('\n');
                content.scrollTop = content.scrollHeight;
            } catch (e) {
                console.error('Failed to fetch logs:', e);
            }
        }

        function clearLogs(name) {
            const content = document.getElementById('logs-content-' + name);
            content.textContent = '';
        }

        async function stop(name) {
            await fetch('/api/stop?name=' + encodeURIComponent(name));
            fetchStatus();
        }

        async function restart(name) {
            await fetch('/api/restart?name=' + encodeURIComponent(name));
            fetchStatus();
        }

        async function start(name) {
            await fetch('/api/start?name=' + encodeURIComponent(name));
            fetchStatus();
        }

        async function toggleProcess(name, status, event) {
            // Find the clicked dot and show starting animation
            const dot = event ? event.target : null;
            if (dot && (status === 'idle' || status === 'failed')) {
                dot.className = 'status-dot starting';
            }

            if (status === 'running') {
                await stop(name);
            } else if (status === 'failed') {
                await restart(name);
            } else {
                await start(name);
            }
        }

        async function reload() {
            await fetch('/api/reload');
            fetchStatus();
        }

        // Initial load
        fetchStatus();

        // Refresh every 5 seconds, preserving expanded logs
        setInterval(() => {
            const expanded = document.querySelector('.logs-panel.visible');
            const expandedApp = expanded ? expanded.id.replace('logs-', '') : null;
            fetchStatus().then(() => {
                if (expandedApp) {
                    const panel = document.getElementById('logs-' + expandedApp);
                    if (panel) {
                        panel.classList.add('visible');
                        fetchLogs(expandedApp);
                    }
                }
            });
        }, 5000);
    </script>
</body>
</html>
`
