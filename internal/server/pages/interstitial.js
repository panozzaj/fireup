// Interstitial page JavaScript
var container = document.querySelector('.container')
var appName = container.dataset.app
var configName = container.dataset.config
var tld = container.dataset.tld
// When accessed via Tailscale (.ts.net), use same origin for API calls
// Otherwise use roost-dev.{tld} for local development
var isTailscale = window.location.hostname.endsWith('.ts.net')
var baseUrl = isTailscale ? window.location.origin : window.location.protocol + '//roost-dev.' + tld
var failed = container.dataset.failed === 'true'
var lastLogCount = 0
var buttonsShown = false
var startTime = Date.now()
var MIN_WAIT_MS = 500

function showButtons() {
    if (!buttonsShown) {
        document.getElementById('logs-buttons').style.display = 'flex'
        buttonsShown = true
    }
}

function toggleSettings() {
    var menu = document.getElementById('settings-menu')
    var btn = document.querySelector('.settings-btn')
    menu.classList.toggle('open')
    btn.classList.toggle('open')
}

// Close dropdown when clicking outside
document.addEventListener('click', function (e) {
    var dropdown = document.querySelector('.settings-dropdown')
    var btn = document.querySelector('.settings-btn')
    if (!dropdown.contains(e.target)) {
        document.getElementById('settings-menu').classList.remove('open')
        btn.classList.remove('open')
    }
})

function copyConfigPath(e) {
    e.stopPropagation()
    var path = window.configFullPath || '~/.config/roost-dev/' + configName + '.yml'
    var btn = document.getElementById('copy-path-btn')
    var origHTML = btn.innerHTML
    navigator.clipboard.writeText(path).then(function () {
        btn.innerHTML = ICONS.checkGreen
        setTimeout(function () {
            btn.innerHTML = origHTML
            document.getElementById('settings-menu').classList.remove('open')
            document.querySelector('.settings-btn').classList.remove('open')
        }, 500)
    })
}

function ansiToHtml(text) {
    var colors = {
        30: '#000',
        31: '#e74c3c',
        32: '#2ecc71',
        33: '#f1c40f',
        34: '#3498db',
        35: '#9b59b6',
        36: '#1abc9c',
        37: '#ecf0f1',
        90: '#7f8c8d',
        91: '#e74c3c',
        92: '#2ecc71',
        93: '#f1c40f',
        94: '#3498db',
        95: '#9b59b6',
        96: '#1abc9c',
        97: '#fff',
    }
    var result = ''
    var i = 0
    var openSpans = 0
    while (i < text.length) {
        if (text[i] === '\x1b' && text[i + 1] === '[') {
            var j = i + 2
            while (j < text.length && text[j] !== 'm') j++
            var codes = text.slice(i + 2, j).split(';')
            i = j + 1
            for (var k = 0; k < codes.length; k++) {
                var code = codes[k]
                if (code === '0' || code === '39' || code === '22' || code === '23') {
                    if (openSpans > 0) {
                        result += '</span>'
                        openSpans--
                    }
                } else if (colors[code]) {
                    result += '<span style="color:' + colors[code] + '">'
                    openSpans++
                } else if (code === '1') {
                    result += '<span style="font-weight:bold">'
                    openSpans++
                } else if (code === '3') {
                    result += '<span style="font-style:italic">'
                    openSpans++
                }
            }
        } else {
            var c = text[i]
            if (c === '<') result += '&lt;'
            else if (c === '>') result += '&gt;'
            else if (c === '&') result += '&amp;'
            else result += c
            i++
        }
    }
    while (openSpans-- > 0) result += '</span>'
    return result
}

function stripAnsi(text) {
    return text.replace(/\x1b\[[0-9;]*m/g, '').replace(/\[\?25[hl]/g, '')
}

function analyzeLogsWithAI(lines) {
    fetch(baseUrl + '/api/analyze-logs?name=' + encodeURIComponent(appName))
        .then(function (res) {
            return res.json()
        })
        .then(function (data) {
            if (!data.enabled || data.error || !data.errorLines || data.errorLines.length === 0) return
            var errorSet = new Set(data.errorLines)
            var content = document.getElementById('logs-content')
            var highlighted = lines
                .map(function (line, idx) {
                    var html = ansiToHtml(line)
                    return errorSet.has(idx) ? '<mark>' + html + '</mark>' : html
                })
                .join('\n')
            content.innerHTML = highlighted
        })
        .catch(function (e) {
            console.log('AI analysis skipped:', e)
        })
}

function poll() {
    Promise.all([
        fetch(baseUrl + '/api/app-status?name=' + encodeURIComponent(appName)),
        fetch(baseUrl + '/api/logs?name=' + encodeURIComponent(appName)),
    ])
        .then(function (responses) {
            return Promise.all([responses[0].json(), responses[1].json()])
        })
        .then(function (results) {
            var status = results[0]
            var lines = results[1]
            if (lines && lines.length > 0) {
                var content = document.getElementById('logs-content')
                content.innerHTML = ansiToHtml(lines.join('\n'))
                showButtons()
                if (lines.length > lastLogCount) {
                    var logsDiv = document.getElementById('logs')
                    logsDiv.scrollTop = logsDiv.scrollHeight
                    lastLogCount = lines.length
                }
            }
            if (status.status === 'running') {
                var elapsed = Date.now() - startTime
                if (elapsed < MIN_WAIT_MS) {
                    document.getElementById('status').textContent = 'Almost ready...'
                    setTimeout(poll, MIN_WAIT_MS - elapsed)
                    return
                }
                document.getElementById('status').textContent = 'Ready! Redirecting...'
                document.getElementById('spinner').style.borderTopColor = '#22c55e'
                setTimeout(function () {
                    location.reload()
                }, 300)
                return
            } else if (status.status === 'failed') {
                showError(status.error)
                // Keep polling slower to detect external restarts
                setTimeout(poll, 2000)
                return
            } else if (status.status === 'starting') {
                // Service was restarted externally - update UI
                if (failed) {
                    failed = false
                    lastLogCount = 0
                    document.getElementById('status').textContent = 'Starting...'
                    document.getElementById('status').classList.remove('error')
                    document.getElementById('spinner').style.display = 'block'
                    document.getElementById('spinner').style.borderTopColor = ''
                    document.getElementById('retry-btn').style.display = 'none'
                    document.getElementById('fix-btn').style.display = 'none'
                    document.getElementById('logs-content').innerHTML = '<span class="logs-empty">Restarting...</span>'
                }
            }
            setTimeout(poll, 200)
        })
        .catch(function (e) {
            console.error('Poll failed:', e)
            setTimeout(poll, 1000)
        })
}

function showError(msg) {
    document.getElementById('spinner').style.display = 'none'
    var statusEl = document.getElementById('status')
    statusEl.textContent = 'Failed to start' + (msg ? ': ' + stripAnsi(msg) : '')
    statusEl.classList.add('error')
    var btn = document.getElementById('retry-btn')
    btn.style.display = 'inline-block'
    btn.disabled = false
    btn.textContent = 'Restart'
    // Show the Fix with Claude Code button
    document.getElementById('fix-btn').style.display = 'inline-block'
}

function fixWithClaudeCode() {
    var btn = document.getElementById('fix-btn')
    var origHTML = btn.innerHTML
    btn.disabled = true
    fetch(baseUrl + '/api/open-terminal?name=' + encodeURIComponent(appName))
        .then(function (res) {
            if (!res.ok) {
                console.error('Failed to open terminal')
                btn.innerHTML = ICONS.xRed
                setTimeout(function () {
                    btn.innerHTML = origHTML
                    btn.disabled = false
                }, 2000)
                return
            }
            btn.innerHTML = ICONS.checkGreen
            setTimeout(function () {
                btn.innerHTML = origHTML
                btn.disabled = false
            }, 1000)
        })
        .catch(function (e) {
            console.error('Failed to open terminal:', e)
            btn.innerHTML = ICONS.xRed
            setTimeout(function () {
                btn.innerHTML = origHTML
                btn.disabled = false
            }, 2000)
        })
}

function getLogsText() {
    // Use selection if user has selected text within logs, otherwise use all logs
    var selection = window.getSelection()
    var logsContent = document.getElementById('logs-content')
    if (selection && selection.toString().trim() && logsContent.contains(selection.anchorNode)) {
        return selection.toString()
    }
    return logsContent.textContent
}

function copyLogs() {
    var btn = document.getElementById('copy-btn')
    var origHTML = btn.innerHTML
    var text = getLogsText()
    var textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    btn.innerHTML = ICONS.checkGreen
    setTimeout(function () {
        btn.innerHTML = origHTML
    }, 500)
}

function copyForAgent() {
    var btn = document.getElementById('copy-agent-btn')
    var origHTML = btn.innerHTML
    var logs = getLogsText()
    var bt = String.fromCharCode(96)
    var configPath = window.configFullPath || '~/.config/roost-dev/' + appName + '.yml'
    var context =
        'I am using roost-dev, a local development server that manages apps via config files in ~/.config/roost-dev/.\n\n' +
        'The app "' +
        appName +
        '" failed to start. The config file is at:\n' +
        configPath +
        '\n\n' +
        'Here are the startup logs:\n\n' +
        bt +
        bt +
        bt +
        '\n' +
        logs +
        '\n' +
        bt +
        bt +
        bt +
        '\n\n' +
        'Useful commands:\n' +
        '  roost-dev restart ' +
        appName +
        '  # Restart this app\n' +
        '  roost-dev logs ' +
        appName +
        '     # View logs\n' +
        '  roost-dev --help                 # CLI help\n' +
        '  roost-dev docs                   # Full documentation\n\n' +
        'Please help me understand and fix this error.'
    var textarea = document.createElement('textarea')
    textarea.value = context
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    btn.innerHTML = ICONS.checkGreen
    setTimeout(function () {
        btn.innerHTML = origHTML
    }, 500)
}

function openConfig(e) {
    if (e) e.stopPropagation()
    var btn = document.getElementById('open-editor-btn')
    fetch(baseUrl + '/api/open-config?name=' + encodeURIComponent(configName))
        .then(function (res) {
            if (!res.ok) {
                console.error('Failed to open config')
                btn.style.color = '#e74c3c'
                setTimeout(function () {
                    btn.style.color = ''
                }, 2000)
                return
            }
            btn.style.color = '#22c55e'
            document.getElementById('settings-menu').classList.remove('open')
            document.querySelector('.settings-btn').classList.remove('open')
            setTimeout(function () {
                btn.style.color = ''
            }, 500)
        })
        .catch(function (e) {
            console.error('Failed to open config:', e)
            btn.style.color = '#e74c3c'
            setTimeout(function () {
                btn.style.color = ''
            }, 2000)
        })
}

function fetchConfigPath() {
    fetch(baseUrl + '/api/config-path?name=' + encodeURIComponent(configName))
        .then(function (res) {
            return res.json()
        })
        .then(function (data) {
            if (data.path) {
                // Store full path and update tooltip
                window.configFullPath = data.path
                document.getElementById('copy-path-btn').setAttribute('data-tooltip', data.path)
            }
        })
        .catch(function (e) {
            console.log('Failed to fetch config path:', e)
        })
}

function restartAndRetry() {
    var btn = document.getElementById('retry-btn')
    var statusEl = document.getElementById('status')
    btn.textContent = 'Restarting...'
    btn.disabled = true
    statusEl.textContent = 'Restarting...'
    statusEl.classList.remove('error')
    document.getElementById('spinner').style.display = 'block'
    document.getElementById('logs-content').innerHTML = '<span class="logs-empty">Restarting...</span>'
    var url = baseUrl + '/api/restart?name=' + encodeURIComponent(appName)
    fetch(url)
        .then(function (res) {
            if (!res.ok) throw new Error('Restart API returned ' + res.status)
            failed = false
            lastLogCount = 0
            statusEl.textContent = 'Starting...'
            btn.style.display = 'none'
            btn.textContent = 'Restart'
            btn.disabled = false
            poll()
        })
        .catch(function (e) {
            console.error('Restart failed:', e)
            btn.textContent = 'Restart'
            btn.disabled = false
            statusEl.textContent = 'Restart failed: ' + e.message
            statusEl.classList.add('error')
            document.getElementById('spinner').style.display = 'none'
        })
}

// Fetch config path on load
fetchConfigPath()

// Listen for theme changes from dashboard via SSE
var themeSource = new EventSource(baseUrl + '/api/events')
themeSource.onmessage = function (event) {
    try {
        var data = JSON.parse(event.data)
        if (data.type === 'theme') {
            if (data.theme === 'system') {
                document.documentElement.removeAttribute('data-theme')
            } else {
                document.documentElement.setAttribute('data-theme', data.theme)
            }
        }
    } catch (_e) {}
}

if (failed) {
    var errorMsg = container.dataset.error || ''
    showError(errorMsg)
    fetch(baseUrl + '/api/logs?name=' + encodeURIComponent(appName))
        .then(function (r) {
            return r.json()
        })
        .then(function (lines) {
            if (lines && lines.length > 0) {
                document.getElementById('logs-content').innerHTML = ansiToHtml(lines.join('\n'))
                showButtons()
                analyzeLogsWithAI(lines)
            }
            // Start polling to detect external restarts (slower interval for failed state)
            setTimeout(poll, 2000)
        })
} else {
    poll()
}
