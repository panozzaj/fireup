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
        :root {
            --bg-primary: #1a1a2e;
            --bg-secondary: #16213e;
            --bg-tertiary: #1a2744;
            --bg-logs: #0f0f1a;
            --text-primary: #eee;
            --text-secondary: #d1d5db;
            --text-muted: #9ca3af;
            --border-color: #333;
            --accent-blue: #60a5fa;
            --accent-blue-hover: #93c5fd;
            --btn-bg: #374151;
            --btn-hover: #4b5563;
            --tag-bg: #374151;
            --success: #22c55e;
            --warning: #f59e0b;
            --error: #ef4444;
            --error-bg: rgba(239, 68, 68, 0.1);
        }

        @media (prefers-color-scheme: light) {
            :root:not([data-theme="dark"]) {
                --bg-primary: #f5f5f5;
                --bg-secondary: #ffffff;
                --bg-tertiary: #f0f0f0;
                --bg-logs: #fafafa;
                --text-primary: #1a1a1a;
                --text-secondary: #374151;
                --text-muted: #6b7280;
                --border-color: #e5e7eb;
                --btn-bg: #e5e7eb;
                --btn-hover: #d1d5db;
                --tag-bg: #e5e7eb;
            }
        }

        [data-theme="light"] {
            --bg-primary: #f5f5f5;
            --bg-secondary: #ffffff;
            --bg-tertiary: #f0f0f0;
            --bg-logs: #fafafa;
            --text-primary: #1a1a1a;
            --text-secondary: #374151;
            --text-muted: #6b7280;
            --border-color: #e5e7eb;
            --btn-bg: #e5e7eb;
            --btn-hover: #d1d5db;
            --tag-bg: #e5e7eb;
        }

        [data-theme="dark"] {
            --bg-primary: #1a1a2e;
            --bg-secondary: #16213e;
            --bg-tertiary: #1a2744;
            --bg-logs: #0f0f1a;
            --text-primary: #eee;
            --text-secondary: #d1d5db;
            --text-muted: #9ca3af;
            --border-color: #333;
            --btn-bg: #374151;
            --btn-hover: #4b5563;
            --tag-bg: #374151;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            padding: 20px;
            transition: background 0.2s, color 0.2s;
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
            border-bottom: 1px solid var(--border-color);
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
        }
        .header-actions {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .theme-toggle {
            background: var(--btn-bg);
            color: var(--text-secondary);
            border: none;
            padding: 8px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 16px;
            line-height: 1;
        }
        .theme-toggle:hover {
            background: var(--btn-hover);
        }
        .connection-status {
            font-size: 12px;
            color: var(--text-muted);
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .connection-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%%;
            background: var(--error);
        }
        .connection-dot.connected {
            background: var(--success);
        }
        .app {
            background: var(--bg-secondary);
            border-radius: 8px;
            margin-bottom: 12px;
            transition: background 0.2s;
        }
        .app.highlight {
            animation: highlightPulse 1s ease-out;
        }
        @keyframes highlightPulse {
            0%% { box-shadow: 0 0 0 0 rgba(96, 165, 250, 0.7); }
            70%% { box-shadow: 0 0 0 10px rgba(96, 165, 250, 0); }
            100%% { box-shadow: 0 0 0 0 rgba(96, 165, 250, 0); }
        }
        .app-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 16px 20px;
            cursor: pointer;
        }
        .app-header:hover {
            background: var(--bg-tertiary);
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
            background: var(--text-muted);
            cursor: pointer;
            transition: transform 0.1s;
        }
        .status-dot:hover {
            transform: scale(1.3);
        }
        .status-dot-wrapper {
            position: relative;
            display: inline-block;
        }
        .status-menu {
            position: absolute;
            top: 100%%;
            left: 50%%;
            transform: translateX(-50%%);
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            padding: 4px 0;
            min-width: 80px;
            z-index: 100;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            display: none;
        }
        .status-menu.visible {
            display: block;
        }
        .status-menu button {
            display: block;
            width: 100%%;
            padding: 6px 12px;
            background: none;
            border: none;
            color: var(--text-secondary);
            font-size: 12px;
            text-align: left;
            cursor: pointer;
        }
        .status-menu button:hover {
            background: var(--btn-bg);
            color: var(--text-primary);
        }
        .status-menu button.danger {
            color: var(--error);
        }
        .status-menu button.danger:hover {
            background: var(--error);
            color: #fff;
        }
        .status-dot.running {
            background: var(--success);
        }
        .status-dot.failed {
            background: var(--error);
        }
        .status-dot.idle {
            background: var(--text-muted);
        }
        .status-dot.starting {
            background: var(--warning);
            animation: pulse 1s ease-in-out infinite;
        }
        @keyframes pulse {
            0%%, 100%% { opacity: 1; transform: scale(1); }
            50%% { opacity: 0.5; transform: scale(1.2); }
        }
        .app-description {
            font-size: 13px;
            color: var(--text-muted);
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
            color: var(--text-secondary);
            background: var(--tag-bg);
            padding: 2px 8px;
            border-radius: 4px;
        }
        .app-aliases {
            font-size: 12px;
            color: var(--text-muted);
            font-style: italic;
        }
        .app-url {
            color: var(--accent-blue);
            text-decoration: none;
            font-size: 14px;
        }
        .app-url:hover {
            text-decoration: underline;
            color: var(--accent-blue-hover);
        }
        .app-meta {
            display: flex;
            align-items: center;
            gap: 16px;
        }
        .app-port {
            font-size: 14px;
            color: var(--text-muted);
        }
        .services {
            padding: 0 20px 16px 42px;
        }
        .service {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px 12px;
            background: var(--bg-tertiary);
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
            color: var(--text-secondary);
        }
        .app-error {
            font-size: 12px;
            color: var(--error);
            display: block;
            margin-top: 4px;
            padding: 4px 8px;
            background: var(--error-bg);
            border-radius: 4px;
            max-width: 500px;
        }
        .logs-panel {
            background: var(--bg-logs);
            border-top: 1px solid var(--border-color);
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
            color: var(--text-muted);
        }
        .logs-actions {
            display: flex;
            gap: 8px;
        }
        .logs-actions button {
            background: var(--btn-bg);
            color: var(--text-secondary);
            border: none;
            padding: 6px 12px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
            transition: background 0.15s;
        }
        .logs-actions button:hover {
            background: var(--btn-hover);
            color: var(--text-primary);
        }
        .logs-content {
            font-family: "SF Mono", Monaco, "Cascadia Code", monospace;
            font-size: 12px;
            line-height: 1.6;
            max-height: 300px;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-all;
            color: var(--text-secondary);
        }
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: var(--text-muted);
        }
        .empty-state h2 {
            font-size: 18px;
            margin-bottom: 12px;
            color: var(--text-secondary);
        }
        .empty-state code {
            display: block;
            background: var(--bg-secondary);
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
            <div class="header-actions">
                <span class="connection-status">
                    <span class="connection-dot" id="connection-dot"></span>
                    <span id="connection-text">Connecting...</span>
                </span>
                <button class="theme-toggle" onclick="toggleTheme()" title="Toggle theme">
                    <span id="theme-icon">&#9790;</span>
                </button>
            </div>
        </header>
        <main id="apps"></main>
    </div>

    <script>
        const TLD = '%s';
        const PORT = %d;
        const portSuffix = PORT === 80 ? '' : ':' + PORT;
        let currentApps = [];
        let expandedLogs = null;
        let eventSource = null;

        const externalLinkIcon = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.25 5.5a.75.75 0 00-.75.75v8.5c0 .414.336.75.75.75h8.5a.75.75 0 00.75-.75v-4a.75.75 0 011.5 0v4A2.25 2.25 0 0112.75 17h-8.5A2.25 2.25 0 012 14.75v-8.5A2.25 2.25 0 014.25 4h5a.75.75 0 010 1.5h-5z" clip-rule="evenodd" /><path fill-rule="evenodd" d="M6.194 12.753a.75.75 0 001.06.053L16.5 4.44v2.81a.75.75 0 001.5 0v-4.5a.75.75 0 00-.75-.75h-4.5a.75.75 0 000 1.5h2.553l-9.056 8.194a.75.75 0 00-.053 1.06z" clip-rule="evenodd" /></svg>';

        // Theme management
        function getTheme() {
            return localStorage.getItem('roost-theme') || 'system';
        }

        function setTheme(theme) {
            localStorage.setItem('roost-theme', theme);
            if (theme === 'system') {
                document.documentElement.removeAttribute('data-theme');
            } else {
                document.documentElement.setAttribute('data-theme', theme);
            }
            updateThemeIcon();
        }

        function toggleTheme() {
            const current = getTheme();
            const themes = ['system', 'light', 'dark'];
            const next = themes[(themes.indexOf(current) + 1) %% themes.length];
            setTheme(next);
        }

        function updateThemeIcon() {
            const theme = getTheme();
            const icon = document.getElementById('theme-icon');
            if (theme === 'light') icon.textContent = '☀';
            else if (theme === 'dark') icon.textContent = '☾';
            else icon.textContent = '◐';
        }

        // Initialize theme
        setTheme(getTheme());

        // SSE Connection
        function connectSSE() {
            if (eventSource) {
                eventSource.close();
            }

            eventSource = new EventSource('/api/events');

            eventSource.onopen = () => {
                document.getElementById('connection-dot').classList.add('connected');
                document.getElementById('connection-text').textContent = 'Live';
            };

            eventSource.onmessage = (event) => {
                try {
                    const apps = JSON.parse(event.data);
                    updateApps(apps || []);
                } catch (e) {
                    console.error('Failed to parse SSE data:', e);
                }
            };

            eventSource.onerror = () => {
                document.getElementById('connection-dot').classList.remove('connected');
                document.getElementById('connection-text').textContent = 'Reconnecting...';
                // EventSource will auto-reconnect
            };
        }

        // Incremental DOM update - only updates changed elements
        function updateApps(newApps) {
            const container = document.getElementById('apps');
            const oldAppsMap = new Map(currentApps.map(a => [a.name, a]));
            const newAppsMap = new Map(newApps.map(a => [a.name, a]));

            // Handle empty state
            if (!newApps.length) {
                currentApps = [];
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

            // If no apps rendered yet, do full render
            if (!currentApps.length) {
                currentApps = newApps;
                container.innerHTML = newApps.map(app => renderApp(app)).join('');
                return;
            }

            // Detect changes
            const added = newApps.filter(a => !oldAppsMap.has(a.name));
            const removed = currentApps.filter(a => !newAppsMap.has(a.name));

            // Remove deleted apps
            for (const app of removed) {
                const el = container.querySelector(` + "`" + `[data-name="${app.name}"]` + "`" + `);
                if (el) el.remove();
            }

            // Add new apps with highlight
            for (const app of added) {
                const html = renderApp(app);
                container.insertAdjacentHTML('beforeend', html);
                const el = container.querySelector(` + "`" + `[data-name="${app.name}"]` + "`" + `);
                if (el) {
                    el.classList.add('highlight');
                    setTimeout(() => el.classList.remove('highlight'), 1000);
                }
            }

            // Update existing apps (in-place)
            for (const app of newApps) {
                const oldApp = oldAppsMap.get(app.name);
                if (oldApp && JSON.stringify(oldApp) !== JSON.stringify(app)) {
                    updateAppInPlace(app);
                }
            }

            currentApps = newApps;

            // Restore expanded logs panel
            if (expandedLogs) {
                const panel = document.getElementById('logs-' + expandedLogs);
                if (panel) panel.classList.add('visible');
            }
        }

        function updateAppInPlace(app) {
            const el = document.querySelector(` + "`" + `[data-name="${app.name}"]` + "`" + `);
            if (!el) return;

            // Update app status dot
            const isRunning = app.running || (app.services && app.services.some(s => s.running));
            const isStarting = app.starting || (app.services && app.services.some(s => s.starting));
            const hasFailed = app.failed || (app.services && app.services.some(s => s.failed));
            const statusClass = hasFailed ? 'failed' : (isRunning ? 'running' : (isStarting ? 'starting' : 'idle'));

            const appDot = el.querySelector(':scope > .app-header .status-dot');
            if (appDot) {
                appDot.className = 'status-dot ' + statusClass;
            }

            // Update uptime
            const uptimeEl = el.querySelector('.app-port');
            if (uptimeEl && app.uptime) {
                uptimeEl.textContent = app.uptime;
            }

            // Update services
            if (app.services) {
                for (const svc of app.services) {
                    const svcStatus = svc.failed ? 'failed' : (svc.running ? 'running' : (svc.starting ? 'starting' : 'idle'));
                    const svcDot = el.querySelector(` + "`" + `.service-info .status-dot[onclick*="${svc.name}-${app.name}"]` + "`" + `);
                    if (svcDot) {
                        svcDot.className = 'status-dot ' + svcStatus;
                    }
                }
            }
        }

        function renderApp(app) {
            const isRunning = app.running || (app.services && app.services.some(s => s.running));
            const isStarting = app.starting || (app.services && app.services.some(s => s.starting));
            const hasFailed = app.failed || (app.services && app.services.some(s => s.failed));
            const statusClass = hasFailed ? 'failed' : (isRunning ? 'running' : (isStarting ? 'starting' : 'idle'));
            const displayName = app.description || app.name;

            const getServiceStatus = (svc) => svc.failed ? 'failed' : (svc.running ? 'running' : (svc.starting ? 'starting' : 'idle'));

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
                                    <div class="status-dot-wrapper">
                                        <div class="status-dot ${svcStatus}" onclick="event.stopPropagation(); handleDotClick('${svcName}', '${svcStatus}', event)"></div>
                                        ${svcStatus === 'running' ? ` + "`" + `<div class="status-menu" id="menu-${svcName}">
                                            <button onclick="event.stopPropagation(); doRestart('${svcName}', event)">Restart</button>
                                            <button class="danger" onclick="event.stopPropagation(); doStop('${svcName}')">Stop</button>
                                        </div>` + "`" + ` : ''}
                                    </div>
                                    <span class="service-name">${svc.name}</span>
                                    ${svc.port ? ` + "`" + `<span class="app-port">:${svc.port}</span>` + "`" + ` : ''}
                                    ${svc.error ? ` + "`" + `<span class="app-error">${svc.error}</span>` + "`" + ` : ''}
                                </div>
                                <a class="app-url external-link" href="http://${svc.name}-${app.name}.${TLD}${portSuffix}" target="_blank" rel="noopener">
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
                            <div class="status-dot-wrapper">
                                <div class="status-dot ${statusClass}" onclick="event.stopPropagation(); handleDotClick('${app.name}', '${statusClass}', event)"></div>
                                ${statusClass === 'running' ? ` + "`" + `<div class="status-menu" id="menu-${app.name}">
                                    <button onclick="event.stopPropagation(); doRestart('${app.name}', event)">Restart</button>
                                    <button class="danger" onclick="event.stopPropagation(); doStop('${app.name}')">Stop</button>
                                </div>` + "`" + ` : ''}
                            </div>
                            <span class="app-name">${displayName}</span>
                            ${app.description ? ` + "`" + `<span class="app-description">(${app.name})</span>` + "`" + ` : ''}
                            <span class="app-type">${app.type}</span>
                            ${app.aliases && app.aliases.length ? ` + "`" + `<span class="app-aliases">aka ${app.aliases.join(', ')}</span>` + "`" + ` : ''}
                        </div>
                        <div class="app-meta">
                            ${app.port ? ` + "`" + `<span class="app-port">:${app.port}</span>` + "`" + ` : ''}
                            ${app.uptime ? ` + "`" + `<span class="app-port">${app.uptime}</span>` + "`" + ` : ''}
                            <a class="app-url external-link" href="${app.url}" target="_blank" rel="noopener" onclick="event.stopPropagation()">
                                ${app.name}.${TLD} ${externalLinkIcon}
                            </a>
                        </div>
                    </div>
                    ${servicesHTML}
                    <div class="logs-panel" id="logs-${app.name}">
                        <div class="logs-header">
                            <span class="logs-title">Logs</span>
                            <div class="logs-actions">
                                <button onclick="event.stopPropagation(); copyLogs('${app.name}', event)">Copy</button>
                                <button onclick="event.stopPropagation(); clearLogs('${app.name}')">Clear</button>
                            </div>
                        </div>
                        <div class="logs-content" id="logs-content-${app.name}"></div>
                    </div>
                </div>
            ` + "`" + `;
        }

        async function toggleLogs(name) {
            const panel = document.getElementById('logs-' + name);
            const isVisible = panel.classList.contains('visible');

            // Hide all panels
            document.querySelectorAll('.logs-panel').forEach(p => p.classList.remove('visible'));

            if (!isVisible) {
                panel.classList.add('visible');
                expandedLogs = name;
                await fetchLogs(name);
            } else {
                expandedLogs = null;
            }
        }

        async function fetchLogs(name) {
            try {
                const res = await fetch('/api/logs?name=' + encodeURIComponent(name));
                const lines = await res.json();
                const content = document.getElementById('logs-content-' + name);
                if (content) {
                    const wasAtBottom = content.scrollHeight - content.scrollTop <= content.clientHeight + 50;
                    content.textContent = (lines || []).join('\n');
                    if (wasAtBottom) {
                        content.scrollTop = content.scrollHeight;
                    }
                }
            } catch (e) {
                console.error('Failed to fetch logs:', e);
            }
        }

        function clearLogs(name) {
            const content = document.getElementById('logs-content-' + name);
            if (content) content.textContent = '';
        }

        async function copyLogs(name, event) {
            const content = document.getElementById('logs-content-' + name);
            try {
                await navigator.clipboard.writeText(content.textContent);
                event.target.textContent = 'Copied!';
                setTimeout(() => event.target.textContent = 'Copy', 1500);
            } catch (err) {
                console.error('Failed to copy:', err);
            }
        }

        async function stop(name) {
            await fetch('/api/stop?name=' + encodeURIComponent(name));
        }

        async function restart(name) {
            await fetch('/api/restart?name=' + encodeURIComponent(name));
        }

        async function start(name) {
            await fetch('/api/start?name=' + encodeURIComponent(name));
        }

        function closeAllMenus() {
            document.querySelectorAll('.status-menu').forEach(m => m.classList.remove('visible'));
        }

        function handleDotClick(name, status, event) {
            event.stopPropagation();
            closeAllMenus();

            if (status === 'running') {
                const menu = document.getElementById('menu-' + name);
                if (menu) menu.classList.add('visible');
            } else {
                const dot = event.target;
                dot.className = 'status-dot starting';
                if (status === 'failed') {
                    restart(name);
                } else {
                    start(name);
                }
            }
        }

        async function doRestart(name, event) {
            closeAllMenus();
            const wrapper = event.target.closest('.status-dot-wrapper');
            const dot = wrapper ? wrapper.querySelector('.status-dot') : null;
            if (dot) dot.className = 'status-dot starting';
            await restart(name);
        }

        async function doStop(name) {
            closeAllMenus();
            await stop(name);
        }

        document.addEventListener('click', closeAllMenus);

        // Periodically refresh logs if panel is open
        setInterval(() => {
            if (expandedLogs) {
                fetchLogs(expandedLogs);
            }
        }, 2000);

        // Start SSE connection
        connectSSE();
    </script>
</body>
</html>
`
