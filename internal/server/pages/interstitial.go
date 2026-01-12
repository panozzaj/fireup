package pages

import (
	"html/template"
	"strings"

	"github.com/panozzaj/roost-dev/internal/logo"
	"github.com/panozzaj/roost-dev/internal/styles"
)

// interstitialData holds data for the interstitial page template
type interstitialData struct {
	AppName     string
	TLD         string
	StatusText  string
	Failed      bool
	ErrorMsg    string
	ThemeScript template.HTML
	ThemeCSS    template.CSS
	PageCSS     template.CSS
	MarkCSS     template.CSS
	Logo        template.HTML
	Script      template.JS
}

var interstitialTmpl = template.Must(template.New("interstitial").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.StatusText}} {{.AppName}}</title>
{{.ThemeScript}}
    <style>
{{.ThemeCSS}}
{{.PageCSS}}
{{.MarkCSS}}
    </style>
</head>
<body>
    <div class="container" data-error="{{.ErrorMsg}}" data-app="{{.AppName}}" data-tld="{{.TLD}}" data-failed="{{.Failed}}">
        <div class="logo"><a href="http://roost-dev.{{.TLD}}/" title="roost-dev dashboard">{{.Logo}}</a></div>
        <h1>{{.AppName}}</h1>
        <div class="status" id="status">{{.StatusText}}...</div>
        <div class="spinner" id="spinner"></div>
        <div class="logs" id="logs">
            <div class="logs-header">
                <div class="logs-title">Logs <span class="config-path" id="config-path"></span></div>
                <div class="logs-buttons">
                    <button class="btn icon-btn" id="copy-btn" onclick="copyLogs()" title="Copy logs">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
                            <rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect>
                        </svg>
                    </button>
                    <button class="btn icon-btn" id="copy-agent-btn" onclick="copyForAgent()" title="Copy for agent">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
                            <rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect>
                            <path d="M9 14h.01M15 14h.01M10 18c.5.3 1.2.5 2 .5s1.5-.2 2-.5"></path>
                        </svg>
                    </button>
                    <button class="btn icon-btn claude-btn" id="fix-btn" onclick="fixWithClaudeCode()" style="display: none;" title="Fix with Claude Code">
                        <svg viewBox="0 0 16 16" fill="currentColor">
                            <path d="m3.127 10.604 3.135-1.76.053-.153-.053-.085H6.11l-.525-.032-1.791-.048-1.554-.065-1.505-.08-.38-.081L0 7.832l.036-.234.32-.214.455.04 1.009.069 1.513.105 1.097.064 1.626.17h.259l.036-.105-.089-.065-.068-.064-1.566-1.062-1.695-1.121-.887-.646-.48-.327-.243-.306-.104-.67.435-.48.585.04.15.04.593.456 1.267.981 1.654 1.218.242.202.097-.068.012-.049-.109-.181-.9-1.626-.96-1.655-.428-.686-.113-.411a2 2 0 0 1-.068-.484l.496-.674L4.446 0l.662.089.279.242.411.94.666 1.48 1.033 2.014.302.597.162.553.06.17h.105v-.097l.085-1.134.157-1.392.154-1.792.052-.504.25-.605.497-.327.387.186.319.456-.045.294-.19 1.23-.37 1.93-.243 1.29h.142l.161-.16.654-.868 1.097-1.372.484-.545.565-.601.363-.287h.686l.505.751-.226.775-.707.895-.585.759-.839 1.13-.524.904.048.072.125-.012 1.897-.403 1.024-.186 1.223-.21.553.258.06.263-.218.536-1.307.323-1.533.307-2.284.54-.028.02.032.04 1.029.098.44.024h1.077l2.005.15.525.346.315.424-.053.323-.807.411-3.631-.863-.872-.218h-.12v.073l.726.71 1.331 1.202 1.667 1.55.084.383-.214.302-.226-.032-1.464-1.101-.565-.497-1.28-1.077h-.084v.113l.295.432 1.557 2.34.08.718-.112.234-.404.141-.444-.08-.911-1.28-.94-1.44-.759-1.291-.093.053-.448 4.821-.21.246-.484.186-.403-.307-.214-.496.214-.98.258-1.28.21-1.016.19-1.263.112-.42-.008-.028-.092.012-.953 1.307-1.448 1.957-1.146 1.227-.274.109-.477-.247.045-.44.266-.39 1.586-2.018.956-1.25.617-.723-.004-.105h-.036l-4.212 2.736-.75.096-.324-.302.04-.496.154-.162 1.267-.871z"/>
                        </svg>
                    </button>
                    <button class="btn icon-btn" id="config-btn" onclick="openConfig()" title="Open config file">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                            <circle cx="12" cy="12" r="3"></circle>
                            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
                        </svg>
                    </button>
                </div>
            </div>
            <div class="logs-content" id="logs-content"><span class="logs-empty">Waiting for output...</span></div>
        </div>
        <button class="btn btn-primary retry-btn" id="retry-btn" onclick="restartAndRetry()">Restart</button>
    </div>
    <script>{{.Script}}</script>
</body>
</html>
`))

// interstitialCSS contains page-specific CSS for the interstitial page
const interstitialCSS = `
body {
    padding: 60px 40px 40px;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
}
.container {
    text-align: center;
    max-width: 700px;
    width: 100%;
}
.logo {
    font-family: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, "DejaVu Sans Mono", monospace;
    font-size: 12px;
    white-space: pre;
    margin-bottom: 40px;
    letter-spacing: 0;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}
.logo a {
    color: var(--text-muted);
    text-decoration: none;
    transition: color 0.3s;
}
.logo a:hover {
    background: linear-gradient(90deg, #ff6b6b, #feca57, #48dbfb, #ff9ff3, #54a0ff, #5f27cd);
    background-size: 200% auto;
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    animation: rainbow 2s linear infinite;
}
@keyframes rainbow {
    0% { background-position: 0% center; }
    100% { background-position: 200% center; }
}
h1 {
    font-size: 24px;
    margin: 0 0 16px 0;
    color: var(--text-primary);
}
.status {
    font-size: 16px;
    color: var(--text-secondary);
    margin-bottom: 24px;
}
.status.error {
    color: #f87171;
    text-align: center;
}
.spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border-color);
    border-top-color: #22c55e;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto 24px;
}
@keyframes spin {
    to { transform: rotate(360deg); }
}
.logs {
    background: var(--bg-logs);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 16px;
    text-align: left;
    max-height: 350px;
    overflow-y: auto;
    margin-bottom: 24px;
}
.logs-title {
    color: var(--text-secondary);
    font-size: 12px;
    margin-bottom: 8px;
}
.config-path {
    color: var(--text-muted);
    font-family: "SF Mono", Monaco, monospace;
    font-size: 11px;
    margin-left: 8px;
}
.logs-content {
    font-family: "SF Mono", Monaco, monospace;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--text-secondary);
    min-height: 100px;
}
.logs-empty {
    color: var(--text-muted);
    font-style: italic;
}
.btn {
    background: var(--btn-bg);
    color: var(--text-primary);
    border: none;
    padding: 10px 24px;
    border-radius: 6px;
    font-size: 14px;
    cursor: pointer;
}
.btn:hover {
    background: var(--btn-hover);
}
.btn-primary {
    background: #22c55e;
    color: #fff;
}
.btn-primary:hover:not(:disabled) {
    background: #16a34a;
}
.btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}
.retry-btn {
    display: none;
}
.logs-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    position: sticky;
    top: -16px;
    background: var(--bg-logs);
    z-index: 1;
    margin: -16px -16px 8px -16px;
    padding: 16px 16px 8px 16px;
    border-bottom: 1px solid var(--border-color);
}
.logs-buttons {
    display: flex;
    gap: 8px;
}
.icon-btn {
    padding: 6px;
    margin-top: -4px;
}
.icon-btn svg {
    width: 18px;
    height: 18px;
    display: block;
}
.claude-btn {
    animation: claude-pulse 5s ease-in-out infinite;
}
@keyframes claude-pulse {
    0%, 40%, 100% { color: #da7756; }
    50%, 90% { color: #C15F3C; }
}
`

// interstitialScript contains the JavaScript for the interstitial page
const interstitialScript = `
const container = document.querySelector('.container');
const appName = container.dataset.app;
const tld = container.dataset.tld;
let failed = container.dataset.failed === 'true';
let lastLogCount = 0;
const startTime = Date.now();
const MIN_WAIT_MS = 500;

function ansiToHtml(text) {
    const colors = {
        '30': '#000', '31': '#e74c3c', '32': '#2ecc71', '33': '#f1c40f',
        '34': '#3498db', '35': '#9b59b6', '36': '#1abc9c', '37': '#ecf0f1',
        '90': '#7f8c8d', '91': '#e74c3c', '92': '#2ecc71', '93': '#f1c40f',
        '94': '#3498db', '95': '#9b59b6', '96': '#1abc9c', '97': '#fff'
    };
    let result = '';
    let i = 0;
    let openSpans = 0;
    while (i < text.length) {
        if (text[i] === '\x1b' && text[i+1] === '[') {
            let j = i + 2;
            while (j < text.length && text[j] !== 'm') j++;
            const codes = text.slice(i+2, j).split(';');
            i = j + 1;
            for (const code of codes) {
                if (code === '0' || code === '39' || code === '22' || code === '23') {
                    if (openSpans > 0) { result += '</span>'; openSpans--; }
                } else if (colors[code]) {
                    result += '<span style="color:' + colors[code] + '">';
                    openSpans++;
                } else if (code === '1') {
                    result += '<span style="font-weight:bold">';
                    openSpans++;
                } else if (code === '3') {
                    result += '<span style="font-style:italic">';
                    openSpans++;
                }
            }
        } else {
            const c = text[i];
            if (c === '<') result += '&lt;';
            else if (c === '>') result += '&gt;';
            else if (c === '&') result += '&amp;';
            else result += c;
            i++;
        }
    }
    while (openSpans-- > 0) result += '</span>';
    return result;
}

function stripAnsi(text) {
    return text.replace(/\x1b\[[0-9;]*m/g, '').replace(/\[\?25[hl]/g, '');
}

async function analyzeLogsWithAI(lines) {
    try {
        const res = await fetch('http://roost-dev.' + tld + '/api/analyze-logs?name=' + encodeURIComponent(appName));
        const data = await res.json();
        if (!data.enabled || data.error || !data.errorLines || data.errorLines.length === 0) return;
        const errorSet = new Set(data.errorLines);
        const content = document.getElementById('logs-content');
        const highlighted = lines.map((line, idx) => {
            const html = ansiToHtml(line);
            return errorSet.has(idx) ? '<mark>' + html + '</mark>' : html;
        }).join('\n');
        content.innerHTML = highlighted;
    } catch (e) {
        console.log('AI analysis skipped:', e);
    }
}

async function poll() {
    try {
        const [statusRes, logsRes] = await Promise.all([
            fetch('http://roost-dev.' + tld + '/api/app-status?name=' + encodeURIComponent(appName)),
            fetch('http://roost-dev.' + tld + '/api/logs?name=' + encodeURIComponent(appName))
        ]);
        const status = await statusRes.json();
        const lines = await logsRes.json();
        if (lines && lines.length > 0) {
            const content = document.getElementById('logs-content');
            content.innerHTML = ansiToHtml(lines.join('\n'));
            if (lines.length > lastLogCount) {
                const logsDiv = document.getElementById('logs');
                logsDiv.scrollTop = logsDiv.scrollHeight;
                lastLogCount = lines.length;
            }
        }
        if (status.status === 'running') {
            const elapsed = Date.now() - startTime;
            if (elapsed < MIN_WAIT_MS) {
                document.getElementById('status').textContent = 'Almost ready...';
                setTimeout(poll, MIN_WAIT_MS - elapsed);
                return;
            }
            document.getElementById('status').textContent = 'Ready! Redirecting...';
            document.getElementById('spinner').style.borderTopColor = '#22c55e';
            setTimeout(() => location.reload(), 300);
            return;
        } else if (status.status === 'failed') {
            showError(status.error);
            return;
        }
        setTimeout(poll, 200);
    } catch (e) {
        console.error('Poll failed:', e);
        setTimeout(poll, 1000);
    }
}

function showError(msg) {
    document.getElementById('spinner').style.display = 'none';
    const statusEl = document.getElementById('status');
    statusEl.textContent = 'Failed to start' + (msg ? ': ' + stripAnsi(msg) : '');
    statusEl.classList.add('error');
    const btn = document.getElementById('retry-btn');
    btn.style.display = 'inline-block';
    btn.disabled = false;
    btn.textContent = 'Restart';
    // Show the Fix with Claude Code button
    document.getElementById('fix-btn').style.display = 'inline-block';
}

async function fixWithClaudeCode() {
    const btn = document.getElementById('fix-btn');
    const origHTML = btn.innerHTML;
    btn.disabled = true;
    try {
        const res = await fetch('http://roost-dev.' + tld + '/api/open-terminal?name=' + encodeURIComponent(appName));
        if (!res.ok) {
            console.error('Failed to open terminal');
            btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>';
            setTimeout(() => {
                btn.innerHTML = origHTML;
                btn.disabled = false;
            }, 2000);
            return;
        }
        btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>';
        setTimeout(() => {
            btn.innerHTML = origHTML;
            btn.disabled = false;
        }, 1000);
    } catch (e) {
        console.error('Failed to open terminal:', e);
        btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>';
        setTimeout(() => {
            btn.innerHTML = origHTML;
            btn.disabled = false;
        }, 2000);
    }
}

function getLogsText() {
    // Use selection if user has selected text within logs, otherwise use all logs
    const selection = window.getSelection();
    const logsContent = document.getElementById('logs-content');
    if (selection && selection.toString().trim() && logsContent.contains(selection.anchorNode)) {
        return selection.toString();
    }
    return logsContent.textContent;
}

function copyLogs() {
    const btn = document.getElementById('copy-btn');
    const origHTML = btn.innerHTML;
    const text = getLogsText();
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>';
    setTimeout(() => btn.innerHTML = origHTML, 500);
}

function copyForAgent() {
    const btn = document.getElementById('copy-agent-btn');
    const origHTML = btn.innerHTML;
    const logs = getLogsText();
    const bt = String.fromCharCode(96);
    const configPath = window.configFullPath || '~/.config/roost-dev/' + appName + '.yml';
    const context = 'I am using roost-dev, a local development server that manages apps via config files in ~/.config/roost-dev/.\n\n' +
        'The app "' + appName + '" failed to start. The config file is at:\n' +
        configPath + '\n\n' +
        'Here are the startup logs:\n\n' +
        bt+bt+bt + '\n' + logs + '\n' + bt+bt+bt + '\n\n' +
        'To restart the app: roost-dev restart ' + appName + '\n' +
        'To learn more about roost-dev commands: roost-dev --help\n\n' +
        'Please help me understand and fix this error.';
    const textarea = document.createElement('textarea');
    textarea.value = context;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
    btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>';
    setTimeout(() => btn.innerHTML = origHTML, 500);
}

async function openConfig() {
    const btn = document.getElementById('config-btn');
    const origHTML = btn.innerHTML;
    btn.disabled = true;
    try {
        const res = await fetch('http://roost-dev.' + tld + '/api/open-config?name=' + encodeURIComponent(appName));
        if (!res.ok) {
            console.error('Failed to open config');
            btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>';
            setTimeout(() => {
                btn.innerHTML = origHTML;
                btn.disabled = false;
            }, 2000);
            return;
        }
        btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg>';
        setTimeout(() => {
            btn.innerHTML = origHTML;
            btn.disabled = false;
        }, 1000);
    } catch (e) {
        console.error('Failed to open config:', e);
        btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>';
        setTimeout(() => {
            btn.innerHTML = origHTML;
            btn.disabled = false;
        }, 2000);
    }
}

async function fetchConfigPath() {
    try {
        const res = await fetch('http://roost-dev.' + tld + '/api/config-path?name=' + encodeURIComponent(appName));
        const data = await res.json();
        if (data.path) {
            // Store full path for Copy for agent, show relative for display
            window.configFullPath = data.path;
            // Strip ~/.config/roost-dev/ prefix for display
            let displayPath = data.path;
            const homePrefix = data.path.match(/^\/Users\/[^/]+\/.config\/roost-dev\//);
            if (homePrefix) {
                displayPath = data.path.slice(homePrefix[0].length);
            }
            document.getElementById('config-path').textContent = displayPath;
        }
    } catch (e) {
        console.log('Failed to fetch config path:', e);
    }
}

async function restartAndRetry() {
    const btn = document.getElementById('retry-btn');
    const statusEl = document.getElementById('status');
    btn.textContent = 'Restarting...';
    btn.disabled = true;
    statusEl.textContent = 'Restarting...';
    statusEl.classList.remove('error');
    document.getElementById('spinner').style.display = 'block';
    document.getElementById('logs-content').innerHTML = '<span class="logs-empty">Restarting...</span>';
    try {
        const url = 'http://roost-dev.' + tld + '/api/restart?name=' + encodeURIComponent(appName);
        const res = await fetch(url);
        if (!res.ok) throw new Error('Restart API returned ' + res.status);
        failed = false;
        lastLogCount = 0;
        statusEl.textContent = 'Starting...';
        btn.style.display = 'none';
        btn.textContent = 'Restart';
        btn.disabled = false;
        poll();
    } catch (e) {
        console.error('Restart failed:', e);
        btn.textContent = 'Restart';
        btn.disabled = false;
        statusEl.textContent = 'Restart failed: ' + e.message;
        statusEl.classList.add('error');
        document.getElementById('spinner').style.display = 'none';
    }
}

// Fetch config path on load
fetchConfigPath();

// Listen for theme changes from dashboard via SSE
const themeSource = new EventSource('http://roost-dev.' + tld + '/api/events');
themeSource.onmessage = (event) => {
    try {
        const data = JSON.parse(event.data);
        if (data.type === 'theme') {
            if (data.theme === 'system') {
                document.documentElement.removeAttribute('data-theme');
            } else {
                document.documentElement.setAttribute('data-theme', data.theme);
            }
        }
    } catch (e) {}
};

if (failed) {
    const errorMsg = container.dataset.error || '';
    showError(errorMsg);
    fetch('http://roost-dev.' + tld + '/api/logs?name=' + encodeURIComponent(appName))
        .then(r => r.json())
        .then(lines => {
            if (lines && lines.length > 0) {
                document.getElementById('logs-content').innerHTML = ansiToHtml(lines.join('\n'));
                analyzeLogsWithAI(lines);
            }
        });
} else {
    poll();
}
`

// Interstitial renders the interstitial page
func Interstitial(appName, tld, theme string, failed bool, errorMsg string) string {
	statusText := "Starting"
	if failed {
		statusText = "Failed to start"
	}

	var b strings.Builder
	data := interstitialData{
		AppName:     appName,
		TLD:         tld,
		StatusText:  statusText,
		Failed:      failed,
		ErrorMsg:    errorMsg,
		ThemeScript: template.HTML(styles.ThemeScript(theme)),
		ThemeCSS:    template.CSS(styles.ThemeVars + styles.BaseStyles),
		PageCSS:     template.CSS(interstitialCSS),
		MarkCSS:     template.CSS(styles.MarkHighlight),
		Logo:        template.HTML(logo.Web()),
		Script:      template.JS(interstitialScript),
	}
	interstitialTmpl.Execute(&b, data)
	return b.String()
}
