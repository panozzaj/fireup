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
                <div class="logs-title">Logs</div>
                <div class="logs-buttons">
                    <button class="btn copy-btn" id="copy-btn" onclick="copyLogs()">Copy</button>
                    <button class="btn copy-btn" id="copy-agent-btn" onclick="copyForAgent()">Copy for agent</button>
                    <button class="btn copy-btn" id="fix-btn" onclick="fixWithClaudeCode()" style="display: none;">Fix with Claude Code</button>
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
.copy-btn {
    padding: 4px 12px;
    font-size: 12px;
    margin-top: -4px;
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
    btn.textContent = 'Opening...';
    btn.disabled = true;
    try {
        const res = await fetch('http://roost-dev.' + tld + '/api/open-terminal?name=' + encodeURIComponent(appName));
        if (!res.ok) {
            console.error('Failed to open terminal');
            btn.textContent = 'Error';
            setTimeout(() => {
                btn.textContent = 'Fix with Claude Code';
                btn.disabled = false;
            }, 2000);
            return;
        }
        btn.textContent = 'Opened!';
        setTimeout(() => {
            btn.textContent = 'Fix with Claude Code';
            btn.disabled = false;
        }, 1000);
    } catch (e) {
        console.error('Failed to open terminal:', e);
        btn.textContent = 'Error';
        setTimeout(() => {
            btn.textContent = 'Fix with Claude Code';
            btn.disabled = false;
        }, 2000);
    }
}

function copyLogs() {
    const content = document.getElementById('logs-content');
    const btn = document.getElementById('copy-btn');
    const text = content.textContent;
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
    btn.textContent = 'Copied!';
    setTimeout(() => btn.textContent = 'Copy', 500);
}

function copyForAgent() {
    const content = document.getElementById('logs-content');
    const btn = document.getElementById('copy-agent-btn');
    const logs = content.textContent;
    const bt = String.fromCharCode(96);
    const context = 'I am using roost-dev, a local development server that manages apps via config files in ~/.config/roost-dev/.\n\n' +
        'The app "' + appName + '" failed to start. The config file is at:\n' +
        '~/.config/roost-dev/' + appName + '.yml\n\n' +
        'Here are the startup logs:\n\n' +
        bt+bt+bt + '\n' + logs + '\n' + bt+bt+bt + '\n\n' +
        'Please help me understand and fix this error.';
    const textarea = document.createElement('textarea');
    textarea.value = context;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
    btn.textContent = 'Copied!';
    setTimeout(() => btn.textContent = 'Copy for agent', 500);
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
