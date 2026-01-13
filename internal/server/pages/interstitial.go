package pages

import (
	"html/template"
	"strings"

	"github.com/panozzaj/roost-dev/internal/logo"
	"github.com/panozzaj/roost-dev/internal/styles"
)

// interstitialData holds data for the interstitial page template
type interstitialData struct {
	AppName     string // Process name used for API calls (e.g., "forever-start-roost-dev-tests")
	DisplayName string // Display name with dots (e.g., "forever-start.roost-dev-tests")
	ConfigName  string // Config file name (e.g., "roost-dev-tests")
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
	// Icons
	IconGear           template.HTML
	IconCopy           template.HTML
	IconExternalLink   template.HTML
	IconClipboard      template.HTML
	IconClipboardAgent template.HTML
	IconClaude         template.HTML
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
    <div class="container" data-error="{{.ErrorMsg}}" data-app="{{.AppName}}" data-display="{{.DisplayName}}" data-config="{{.ConfigName}}" data-tld="{{.TLD}}" data-failed="{{.Failed}}">
        <div class="logo"><a href="//roost-dev.{{.TLD}}/" title="roost-dev dashboard">{{.Logo}}</a></div>
        <div class="title-row">
            <h1>{{.DisplayName}}</h1>
            <div class="settings-dropdown">
                <button class="settings-btn" onclick="toggleSettings()" aria-label="Settings">{{.IconGear}}</button>
                <div class="settings-menu" id="settings-menu">
                    <span class="settings-filename" id="settings-filename">{{.ConfigName}}.yml</span>
                    <button class="settings-action" id="copy-path-btn" onclick="copyConfigPath(event)" data-tooltip="Copy absolute path">{{.IconCopy}}</button>
                    <button class="settings-action" id="open-editor-btn" onclick="openConfig(event)" data-tooltip="Open in editor">{{.IconExternalLink}}</button>
                </div>
            </div>
        </div>
        <div class="status" id="status">{{.StatusText}}...</div>
        <div class="spinner" id="spinner"></div>
        <div class="logs" id="logs">
            <div class="logs-header">
                <div class="logs-title">Logs</div>
                <div class="logs-buttons" id="logs-buttons" style="display: none;">
                    <button class="btn icon-btn" id="copy-btn" onclick="copyLogs()" title="Copy logs">{{.IconClipboard}}</button>
                    <button class="btn icon-btn" id="copy-agent-btn" onclick="copyForAgent()" title="Copy for agent">{{.IconClipboardAgent}}</button>
                    <button class="btn icon-btn claude-btn" id="fix-btn" onclick="fixWithClaudeCode()" style="display: none;" title="Fix with Claude Code">{{.IconClaude}}</button>
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
.title-row {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-bottom: 16px;
}
.title-row h1 {
    margin: 0;
}
.settings-dropdown {
    position: relative;
}
.settings-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    transition: color 0.2s, background 0.2s;
}
.settings-btn:hover {
    color: var(--text-secondary);
    background: var(--btn-bg);
}
.settings-btn svg {
    width: 18px;
    height: 18px;
    display: block;
    transition: transform 0.6s ease;
}
.settings-btn.open svg {
    transform: rotate(150deg);
}
.settings-menu {
    display: none;
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    margin-top: 8px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 8px 12px;
    white-space: nowrap;
    z-index: 100;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
}
.settings-menu.open {
    display: flex;
    align-items: center;
    gap: 12px;
}
.settings-filename {
    font-family: "SF Mono", Monaco, monospace;
    font-size: 13px;
    color: var(--text-secondary);
}
.settings-action {
    position: relative;
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    transition: color 0.2s, background 0.2s;
}
.settings-action:hover {
    color: var(--text-primary);
    background: var(--btn-bg);
}
.settings-action svg {
    width: 16px;
    height: 16px;
    display: block;
}
/* CSS Tooltips */
.settings-action::after {
    content: attr(data-tooltip);
    position: absolute;
    bottom: -28px;
    left: 50%;
    transform: translateX(-50%);
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    font-size: 11px;
    padding: 4px 8px;
    border-radius: 4px;
    white-space: nowrap;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.15s;
    z-index: 101;
}
.settings-action:hover::after {
    opacity: 1;
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
const displayName = container.dataset.display;
const configName = container.dataset.config;
const tld = container.dataset.tld;
const baseUrl = window.location.protocol + '//roost-dev.' + tld;
let failed = container.dataset.failed === 'true';
let lastLogCount = 0;
let buttonsShown = false;
const startTime = Date.now();
const MIN_WAIT_MS = 500;

function showButtons() {
    if (!buttonsShown) {
        document.getElementById('logs-buttons').style.display = 'flex';
        buttonsShown = true;
    }
}

function toggleSettings() {
    const menu = document.getElementById('settings-menu');
    const btn = document.querySelector('.settings-btn');
    menu.classList.toggle('open');
    btn.classList.toggle('open');
}

// Close dropdown when clicking outside
document.addEventListener('click', function(e) {
    const dropdown = document.querySelector('.settings-dropdown');
    const btn = document.querySelector('.settings-btn');
    if (!dropdown.contains(e.target)) {
        document.getElementById('settings-menu').classList.remove('open');
        btn.classList.remove('open');
    }
});

function copyConfigPath(e) {
    e.stopPropagation();
    const path = window.configFullPath || '~/.config/roost-dev/' + configName + '.yml';
    const btn = document.getElementById('copy-path-btn');
    const origHTML = btn.innerHTML;
    navigator.clipboard.writeText(path).then(() => {
        btn.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>';
        setTimeout(() => {
            btn.innerHTML = origHTML;
            document.getElementById('settings-menu').classList.remove('open');
            document.querySelector('.settings-btn').classList.remove('open');
        }, 500);
    });
}

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
        const res = await fetch(baseUrl + '/api/analyze-logs?name=' + encodeURIComponent(appName));
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
            fetch(baseUrl + '/api/app-status?name=' + encodeURIComponent(appName)),
            fetch(baseUrl + '/api/logs?name=' + encodeURIComponent(appName))
        ]);
        const status = await statusRes.json();
        const lines = await logsRes.json();
        if (lines && lines.length > 0) {
            const content = document.getElementById('logs-content');
            content.innerHTML = ansiToHtml(lines.join('\n'));
            showButtons();
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
        const res = await fetch(baseUrl + '/api/open-terminal?name=' + encodeURIComponent(appName));
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
        'For full documentation: roost-dev docs\n\n' +
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

async function openConfig(e) {
    if (e) e.stopPropagation();
    const btn = document.getElementById('open-editor-btn');
    try {
        const res = await fetch(baseUrl + '/api/open-config?name=' + encodeURIComponent(configName));
        if (!res.ok) {
            console.error('Failed to open config');
            btn.style.color = '#e74c3c';
            setTimeout(() => btn.style.color = '', 2000);
            return;
        }
        btn.style.color = '#22c55e';
        document.getElementById('settings-menu').classList.remove('open');
        document.querySelector('.settings-btn').classList.remove('open');
        setTimeout(() => btn.style.color = '', 500);
    } catch (e) {
        console.error('Failed to open config:', e);
        btn.style.color = '#e74c3c';
        setTimeout(() => btn.style.color = '', 2000);
    }
}

async function fetchConfigPath() {
    try {
        const res = await fetch(baseUrl + '/api/config-path?name=' + encodeURIComponent(configName));
        const data = await res.json();
        if (data.path) {
            // Store full path and update tooltip
            window.configFullPath = data.path;
            document.getElementById('copy-path-btn').setAttribute('data-tooltip', data.path);
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
        const url = baseUrl + '/api/restart?name=' + encodeURIComponent(appName);
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
const themeSource = new EventSource(baseUrl + '/api/events');
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
    fetch(baseUrl + '/api/logs?name=' + encodeURIComponent(appName))
        .then(r => r.json())
        .then(lines => {
            if (lines && lines.length > 0) {
                document.getElementById('logs-content').innerHTML = ansiToHtml(lines.join('\n'));
                showButtons();
                analyzeLogsWithAI(lines);
            }
        });
} else {
    poll();
}
`

// Interstitial renders the interstitial page
// appName: process name for API calls (e.g., "forever-start-roost-dev-tests")
// displayName: display name with dots (e.g., "forever-start.roost-dev-tests")
// configName: config file name (e.g., "roost-dev-tests")
func Interstitial(appName, displayName, configName, tld, theme string, failed bool, errorMsg string) string {
	statusText := "Starting"
	if failed {
		statusText = "Failed to start"
	}

	var b strings.Builder
	data := interstitialData{
		AppName:     appName,
		DisplayName: displayName,
		ConfigName:  configName,
		TLD:         tld,
		StatusText:  statusText,
		Failed:      failed,
		ErrorMsg:    errorMsg,
		ThemeScript: template.HTML(styles.ThemeScript(theme)),
		ThemeCSS:    template.CSS(styles.HeadCSS()),
		PageCSS:     template.CSS(interstitialCSS),
		MarkCSS:     template.CSS(styles.MarkHighlight),
		Logo:        template.HTML(logo.Web()),
		Script:      template.JS(interstitialScript),
		// Icons
		IconGear:           template.HTML(IconGear),
		IconCopy:           template.HTML(IconCopy),
		IconExternalLink:   template.HTML(IconExternalLink),
		IconClipboard:      template.HTML(IconClipboard),
		IconClipboardAgent: template.HTML(IconClipboardAgent),
		IconClaude:         template.HTML(IconClaude),
	}
	interstitialTmpl.Execute(&b, data)
	return b.String()
}
