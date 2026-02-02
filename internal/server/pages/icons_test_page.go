package pages

import (
	"html/template"
	"strings"

	"github.com/panozzaj/fireup/internal/styles"
)

// iconsTestData holds data for the icons test page template
type iconsTestData struct {
	ThemeScript template.HTML
	ThemeCSS    template.CSS
}

var iconsTestTmpl = template.Must(template.New("icons-test").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Icon Test Page - fireup</title>
{{.ThemeScript}}
    <style>
{{.ThemeCSS}}
body {
    padding: 40px;
    max-width: 1200px;
    margin: 0 auto;
}
h1 { color: var(--text-primary); margin-bottom: 8px; }
h2 { color: var(--text-secondary); margin-top: 32px; margin-bottom: 16px; border-bottom: 1px solid var(--border-color); padding-bottom: 8px; }
h3 { color: var(--text-secondary); margin-top: 24px; margin-bottom: 12px; font-size: 14px; }
.subtitle { color: var(--text-muted); margin-bottom: 32px; }

.icon-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 16px;
    margin-bottom: 24px;
}
.icon-card {
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 16px;
    text-align: center;
}
.icon-card:hover {
    border-color: var(--text-muted);
}
.icon-preview {
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 12px;
}
.icon-preview svg {
    width: 24px;
    height: 24px;
}
.icon-name {
    font-size: 12px;
    color: var(--text-secondary);
    font-family: monospace;
}
.icon-source {
    font-size: 10px;
    color: var(--text-muted);
    margin-top: 4px;
}

.button-preview {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    align-items: center;
    margin: 16px 0;
}
.btn {
    background: var(--btn-bg);
    color: var(--text-primary);
    border: none;
    padding: 8px 12px;
    border-radius: 6px;
    font-size: 14px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 6px;
}
.btn:hover {
    background: var(--btn-hover);
}
.btn svg {
    width: 16px;
    height: 16px;
}
.btn-icon-only {
    padding: 8px;
}
.btn-icon-only svg {
    width: 18px;
    height: 18px;
}

.color-swatch {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    margin: 8px 16px 8px 0;
}
.swatch {
    width: 32px;
    height: 32px;
    border-radius: 4px;
    border: 1px solid var(--border-color);
}
.color-code {
    font-family: monospace;
    font-size: 12px;
    color: var(--text-secondary);
}

.notes {
    background: var(--bg-logs);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 16px;
    margin: 16px 0;
    font-size: 13px;
    color: var(--text-secondary);
}
.notes code {
    background: var(--btn-bg);
    padding: 2px 6px;
    border-radius: 3px;
    font-size: 12px;
}
    </style>
</head>
<body>
    <h1>Icon Test Page</h1>
    <p class="subtitle">Preview icon options for fireup UI buttons</p>

    <h2>Claude Brand Colors</h2>
    <div>
        <div class="color-swatch">
            <div class="swatch" style="background: #da7756;"></div>
            <span class="color-code">#da7756 (terra cotta)</span>
        </div>
        <div class="color-swatch">
            <div class="swatch" style="background: #C15F3C;"></div>
            <span class="color-code">#C15F3C (crail)</span>
        </div>
    </div>

    <h2>Static Site Type Icons</h2>
    <p class="notes">For static file apps on dashboard - replaces status dot. Should be ~10px to align with dots.</p>
    <div class="icon-grid">
        <div class="icon-card" id="static-1">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="16 18 22 12 16 6"></polyline>
                    <polyline points="8 6 2 12 8 18"></polyline>
                </svg>
            </div>
            <div class="icon-name">static-1: code brackets</div>
            <div class="icon-source">Feather (orange)</div>
        </div>
        <div class="icon-card" id="static-2">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"></circle>
                    <line x1="2" y1="12" x2="22" y2="12"></line>
                    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>
                </svg>
            </div>
            <div class="icon-name">static-2: globe</div>
            <div class="icon-source">Feather (orange)</div>
        </div>
        <div class="icon-card" id="static-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
                    <polyline points="13 2 13 9 20 9"></polyline>
                </svg>
            </div>
            <div class="icon-name">static-3: file</div>
            <div class="icon-source">Feather (orange)</div>
        </div>
        <div class="icon-card" id="static-4">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="#e44d26">
                    <path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/>
                </svg>
            </div>
            <div class="icon-name">static-4: home filled</div>
            <div class="icon-source">Material (orange)</div>
        </div>
        <div class="icon-card" id="static-5">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                </svg>
            </div>
            <div class="icon-name">static-5: folder</div>
            <div class="icon-source">Feather (orange)</div>
        </div>
        <div class="icon-card" id="static-6">
            <div class="icon-preview">
                <svg viewBox="0 0 512 512">
                    <path fill="#e44d26" d="M107.6 471l-33-370.4h362.8l-33 370.2L256.2 512"/>
                    <path fill="#f06529" d="M256 480.5V131h148.3L376 447"/>
                    <path fill="#ebebeb" d="M142 176.3h114v45.4h-64.2l4.2 46.5h60v45.3H154.4M156.4 336.3H202l3.2 36.3 50.8 13.6v47.4l-93.2-26"/>
                    <path fill="#fff" d="M369.6 176.3H256v45.4h109.6M361.3 268.2H256v45.4h56l-5.3 59-50.7 13.6v47.2l93-25.8"/>
                </svg>
            </div>
            <div class="icon-name">static-6: HTML5 shield</div>
            <div class="icon-source">Official HTML5 logo</div>
        </div>
    </div>

    <h3>Static Icon Size Test (at 10px - same as status dot)</h3>
    <div style="display: flex; align-items: center; gap: 12px; padding: 16px; background: var(--bg-card); border-radius: 8px; margin-bottom: 24px;">
        <div style="width: 10px; height: 10px; border-radius: 50%; background: #22c55e;"></div>
        <span>Running app (green dot)</span>
    </div>
    <div style="display: flex; align-items: center; gap: 12px; padding: 16px; background: var(--bg-card); border-radius: 8px; margin-bottom: 24px;">
        <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" width="10" height="10"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>
        <span>Static app (code brackets)</span>
    </div>
    <div style="display: flex; align-items: center; gap: 12px; padding: 16px; background: var(--bg-card); border-radius: 8px; margin-bottom: 24px;">
        <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" width="10" height="10"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>
        <span>Static app (globe)</span>
    </div>
    <div style="display: flex; align-items: center; gap: 12px; padding: 16px; background: var(--bg-card); border-radius: 8px; margin-bottom: 24px;">
        <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" width="10" height="10"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path><polyline points="13 2 13 9 20 9"></polyline></svg>
        <span>Static app (file)</span>
    </div>
    <div style="display: flex; align-items: center; gap: 12px; padding: 16px; background: var(--bg-card); border-radius: 8px; margin-bottom: 24px;">
        <svg viewBox="0 0 24 24" fill="#e44d26" width="10" height="10"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>
        <span>Static app (home)</span>
    </div>
    <div style="display: flex; align-items: center; gap: 12px; padding: 16px; background: var(--bg-card); border-radius: 8px; margin-bottom: 24px;">
        <svg viewBox="0 0 24 24" fill="none" stroke="#e44d26" stroke-width="2.5" width="10" height="10"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
        <span>Static app (folder)</span>
    </div>

    <h2>Copy Icons</h2>
    <p class="notes">For "Copy logs" button - simple clipboard</p>
    <div class="icon-grid">
        <div class="icon-card" id="copy-1">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                </svg>
            </div>
            <div class="icon-name">copy-1: two-docs</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="copy-2">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
                    <rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect>
                </svg>
            </div>
            <div class="icon-name">copy-2: clipboard</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="copy-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="currentColor">
                    <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
                </svg>
            </div>
            <div class="icon-name">copy-3: material-filled</div>
            <div class="icon-source">Material Icons</div>
        </div>
    </div>

    <h2>Copy for Agent Icons</h2>
    <p class="notes">For "Copy for agent" button - clipboard with context/AI hint</p>
    <div class="icon-grid">
        <div class="icon-card" id="agent-1">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
                    <rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect>
                    <path d="M9 14h.01M15 14h.01M10 18c.5.3 1.2.5 2 .5s1.5-.2 2-.5"></path>
                </svg>
            </div>
            <div class="icon-name">agent-1: clipboard-robot</div>
            <div class="icon-source">Custom (clipboard + face)</div>
        </div>
        <div class="icon-card" id="agent-2">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                    <path d="M12 16l2 2 4-4"></path>
                </svg>
            </div>
            <div class="icon-name">agent-2: clipboard-check</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="agent-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
                </svg>
            </div>
            <div class="icon-name">agent-3: message-square</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="agent-4">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"></circle>
                    <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"></path>
                    <line x1="12" y1="17" x2="12.01" y2="17"></line>
                </svg>
            </div>
            <div class="icon-name">agent-4: help-circle</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
    </div>

    <h2>Claude / AI Fix Icons</h2>
    <p class="notes">For "Fix with Claude Code" button - Claude's official symbol in brand color</p>
    <div class="icon-grid">
        <div class="icon-card" id="claude-1">
            <div class="icon-preview">
                <svg viewBox="0 0 16 16" fill="#da7756">
                    <path d="m3.127 10.604 3.135-1.76.053-.153-.053-.085H6.11l-.525-.032-1.791-.048-1.554-.065-1.505-.08-.38-.081L0 7.832l.036-.234.32-.214.455.04 1.009.069 1.513.105 1.097.064 1.626.17h.259l.036-.105-.089-.065-.068-.064-1.566-1.062-1.695-1.121-.887-.646-.48-.327-.243-.306-.104-.67.435-.48.585.04.15.04.593.456 1.267.981 1.654 1.218.242.202.097-.068.012-.049-.109-.181-.9-1.626-.96-1.655-.428-.686-.113-.411a2 2 0 0 1-.068-.484l.496-.674L4.446 0l.662.089.279.242.411.94.666 1.48 1.033 2.014.302.597.162.553.06.17h.105v-.097l.085-1.134.157-1.392.154-1.792.052-.504.25-.605.497-.327.387.186.319.456-.045.294-.19 1.23-.37 1.93-.243 1.29h.142l.161-.16.654-.868 1.097-1.372.484-.545.565-.601.363-.287h.686l.505.751-.226.775-.707.895-.585.759-.839 1.13-.524.904.048.072.125-.012 1.897-.403 1.024-.186 1.223-.21.553.258.06.263-.218.536-1.307.323-1.533.307-2.284.54-.028.02.032.04 1.029.098.44.024h1.077l2.005.15.525.346.315.424-.053.323-.807.411-3.631-.863-.872-.218h-.12v.073l.726.71 1.331 1.202 1.667 1.55.084.383-.214.302-.226-.032-1.464-1.101-.565-.497-1.28-1.077h-.084v.113l.295.432 1.557 2.34.08.718-.112.234-.404.141-.444-.08-.911-1.28-.94-1.44-.759-1.291-.093.053-.448 4.821-.21.246-.484.186-.403-.307-.214-.496.214-.98.258-1.28.21-1.016.19-1.263.112-.42-.008-.028-.092.012-.953 1.307-1.448 1.957-1.146 1.227-.274.109-.477-.247.045-.44.266-.39 1.586-2.018.956-1.25.617-.723-.004-.105h-.036l-4.212 2.736-.75.096-.324-.302.04-.496.154-.162 1.267-.871z"/>
                </svg>
            </div>
            <div class="icon-name">claude-1: official (Bootstrap)</div>
            <div class="icon-source">Bootstrap Icons #da7756</div>
        </div>
        <div class="icon-card" id="claude-2">
            <div class="icon-preview">
                <svg viewBox="0 0 16 16" fill="#C15F3C">
                    <path d="m3.127 10.604 3.135-1.76.053-.153-.053-.085H6.11l-.525-.032-1.791-.048-1.554-.065-1.505-.08-.38-.081L0 7.832l.036-.234.32-.214.455.04 1.009.069 1.513.105 1.097.064 1.626.17h.259l.036-.105-.089-.065-.068-.064-1.566-1.062-1.695-1.121-.887-.646-.48-.327-.243-.306-.104-.67.435-.48.585.04.15.04.593.456 1.267.981 1.654 1.218.242.202.097-.068.012-.049-.109-.181-.9-1.626-.96-1.655-.428-.686-.113-.411a2 2 0 0 1-.068-.484l.496-.674L4.446 0l.662.089.279.242.411.94.666 1.48 1.033 2.014.302.597.162.553.06.17h.105v-.097l.085-1.134.157-1.392.154-1.792.052-.504.25-.605.497-.327.387.186.319.456-.045.294-.19 1.23-.37 1.93-.243 1.29h.142l.161-.16.654-.868 1.097-1.372.484-.545.565-.601.363-.287h.686l.505.751-.226.775-.707.895-.585.759-.839 1.13-.524.904.048.072.125-.012 1.897-.403 1.024-.186 1.223-.21.553.258.06.263-.218.536-1.307.323-1.533.307-2.284.54-.028.02.032.04 1.029.098.44.024h1.077l2.005.15.525.346.315.424-.053.323-.807.411-3.631-.863-.872-.218h-.12v.073l.726.71 1.331 1.202 1.667 1.55.084.383-.214.302-.226-.032-1.464-1.101-.565-.497-1.28-1.077h-.084v.113l.295.432 1.557 2.34.08.718-.112.234-.404.141-.444-.08-.911-1.28-.94-1.44-.759-1.291-.093.053-.448 4.821-.21.246-.484.186-.403-.307-.214-.496.214-.98.258-1.28.21-1.016.19-1.263.112-.42-.008-.028-.092.012-.953 1.307-1.448 1.957-1.146 1.227-.274.109-.477-.247.045-.44.266-.39 1.586-2.018.956-1.25.617-.723-.004-.105h-.036l-4.212 2.736-.75.096-.324-.302.04-.496.154-.162 1.267-.871z"/>
                </svg>
            </div>
            <div class="icon-name">claude-2: official (darker)</div>
            <div class="icon-source">Bootstrap Icons #C15F3C</div>
        </div>
        <div class="icon-card" id="claude-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="#da7756" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path>
                </svg>
            </div>
            <div class="icon-name">claude-3: wrench</div>
            <div class="icon-source">Lucide #da7756</div>
        </div>
        <div class="icon-card" id="claude-4">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="#da7756" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
                </svg>
            </div>
            <div class="icon-name">claude-4: star</div>
            <div class="icon-source">Feather #da7756</div>
        </div>
    </div>

    <h2>Open Config Icons</h2>
    <p class="notes">For "Open Config" button - file/settings/gear</p>
    <div class="icon-grid">
        <div class="icon-card" id="config-1">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path>
                    <polyline points="13 2 13 9 20 9"></polyline>
                </svg>
            </div>
            <div class="icon-name">config-1: file</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="config-2">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                    <polyline points="14 2 14 8 20 8"></polyline>
                    <line x1="16" y1="13" x2="8" y2="13"></line>
                    <line x1="16" y1="17" x2="8" y2="17"></line>
                    <polyline points="10 9 9 9 8 9"></polyline>
                </svg>
            </div>
            <div class="icon-name">config-2: file-text</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="config-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="3"></circle>
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
                </svg>
            </div>
            <div class="icon-name">config-3: settings-gear</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="config-4">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="4 7 4 4 20 4 20 7"></polyline>
                    <line x1="9" y1="20" x2="15" y2="20"></line>
                    <line x1="12" y1="4" x2="12" y2="20"></line>
                </svg>
            </div>
            <div class="icon-name">config-4: type</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="config-5">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="16 18 22 12 16 6"></polyline>
                    <polyline points="8 6 2 12 8 18"></polyline>
                </svg>
            </div>
            <div class="icon-name">config-5: code-brackets</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="config-6">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                    <polyline points="15 3 21 3 21 9"></polyline>
                    <line x1="10" y1="14" x2="21" y2="3"></line>
                </svg>
            </div>
            <div class="icon-name">config-6: external-link</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
    </div>

    <h2>Button Previews</h2>
    <h3>Icon-only buttons (with tooltips)</h3>
    <div class="button-preview">
        <button class="btn btn-icon-only" data-tooltip="Copy logs">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
        </button>
        <button class="btn btn-icon-only" data-tooltip="Copy for agent">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
            </svg>
        </button>
        <button class="btn btn-icon-only" data-tooltip="Fix with Claude Code">
            <svg viewBox="0 0 24 24" fill="none" stroke="#da7756" stroke-width="2" stroke-linecap="round">
                <circle cx="12" cy="12" r="2.5" fill="#da7756"/>
                <line x1="12" y1="2" x2="12" y2="6"/>
                <line x1="12" y1="18" x2="12" y2="22"/>
                <line x1="4.22" y1="4.22" x2="6.93" y2="6.93"/>
                <line x1="17.07" y1="17.07" x2="19.78" y2="19.78"/>
                <line x1="2" y1="12" x2="6" y2="12"/>
                <line x1="18" y1="12" x2="22" y2="12"/>
                <line x1="4.22" y1="19.78" x2="6.93" y2="17.07"/>
                <line x1="17.07" y1="6.93" x2="19.78" y2="4.22"/>
            </svg>
        </button>
        <button class="btn btn-icon-only" data-tooltip="Open config file">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                <polyline points="15 3 21 3 21 9"></polyline>
                <line x1="10" y1="14" x2="21" y2="3"></line>
            </svg>
        </button>
    </div>

    <h3>Icon + text buttons</h3>
    <div class="button-preview">
        <button class="btn" data-tooltip="Copy logs">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            Copy
        </button>
        <button class="btn" data-tooltip="Copy for agent">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
            </svg>
            Copy for agent
        </button>
        <button class="btn" data-tooltip="Fix with Claude Code">
            <svg viewBox="0 0 24 24" fill="none" stroke="#da7756" stroke-width="2" stroke-linecap="round">
                <circle cx="12" cy="12" r="2.5" fill="#da7756"/>
                <line x1="12" y1="2" x2="12" y2="6"/>
                <line x1="12" y1="18" x2="12" y2="22"/>
                <line x1="4.22" y1="4.22" x2="6.93" y2="6.93"/>
                <line x1="17.07" y1="17.07" x2="19.78" y2="19.78"/>
                <line x1="2" y1="12" x2="6" y2="12"/>
                <line x1="18" y1="12" x2="22" y2="12"/>
                <line x1="4.22" y1="19.78" x2="6.93" y2="17.07"/>
                <line x1="17.07" y1="6.93" x2="19.78" y2="4.22"/>
            </svg>
            Fix with Claude
        </button>
        <button class="btn" data-tooltip="Open config file">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                <polyline points="15 3 21 3 21 9"></polyline>
                <line x1="10" y1="14" x2="21" y2="3"></line>
            </svg>
            Open Config
        </button>
    </div>

    <h3>Alternative icon combinations</h3>
    <div class="button-preview">
        <button class="btn btn-icon-only" data-tooltip="Copy logs">
            <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
            </svg>
        </button>
        <button class="btn btn-icon-only" data-tooltip="Copy for agent">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
                <rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect>
                <path d="M9 14h.01M15 14h.01M10 18c.5.3 1.2.5 2 .5s1.5-.2 2-.5"></path>
            </svg>
        </button>
        <button class="btn btn-icon-only" data-tooltip="Fix with Claude Code">
            <svg viewBox="0 0 24 24" fill="none" stroke="#da7756" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"></path>
            </svg>
        </button>
        <button class="btn btn-icon-only" data-tooltip="Open config file">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
                <line x1="16" y1="13" x2="8" y2="13"></line>
                <line x1="16" y1="17" x2="8" y2="17"></line>
                <polyline points="10 9 9 9 8 9"></polyline>
            </svg>
        </button>
    </div>

    <h2>Settings Dropdown Icons</h2>
    <p class="notes">For settings widget next to app name - shows dropdown with config options</p>

    <h3>Settings Trigger Icons</h3>
    <div class="icon-grid">
        <div class="icon-card" id="settings-1">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="3"></circle>
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
                </svg>
            </div>
            <div class="icon-name">settings-1: gear</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="settings-2">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="1"></circle>
                    <circle cx="19" cy="12" r="1"></circle>
                    <circle cx="5" cy="12" r="1"></circle>
                </svg>
            </div>
            <div class="icon-name">settings-2: more-horizontal</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="settings-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="1"></circle>
                    <circle cx="12" cy="5" r="1"></circle>
                    <circle cx="12" cy="19" r="1"></circle>
                </svg>
            </div>
            <div class="icon-name">settings-3: more-vertical</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="settings-4">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="6 9 12 15 18 9"></polyline>
                </svg>
            </div>
            <div class="icon-name">settings-4: chevron-down</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
    </div>

    <h3>Copy Path Icon</h3>
    <div class="icon-grid">
        <div class="icon-card" id="copypath-1">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                </svg>
            </div>
            <div class="icon-name">copypath-1: copy</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="copypath-2">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"></path>
                    <rect x="8" y="2" width="8" height="4" rx="1" ry="1"></rect>
                </svg>
            </div>
            <div class="icon-name">copypath-2: clipboard</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="copypath-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
                    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
                </svg>
            </div>
            <div class="icon-name">copypath-3: link</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
    </div>

    <h3>Open in Editor Icon</h3>
    <div class="icon-grid">
        <div class="icon-card" id="edit-1">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                    <polyline points="15 3 21 3 21 9"></polyline>
                    <line x1="10" y1="14" x2="21" y2="3"></line>
                </svg>
            </div>
            <div class="icon-name">edit-1: external-link</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="edit-2">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                </svg>
            </div>
            <div class="icon-name">edit-2: edit/pencil</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="edit-3">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                    <polyline points="14 2 14 8 20 8"></polyline>
                    <line x1="16" y1="13" x2="8" y2="13"></line>
                    <line x1="16" y1="17" x2="8" y2="17"></line>
                    <polyline points="10 9 9 9 8 9"></polyline>
                </svg>
            </div>
            <div class="icon-name">edit-3: file-text</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
        <div class="icon-card" id="edit-4">
            <div class="icon-preview">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="16 18 22 12 16 6"></polyline>
                    <polyline points="8 6 2 12 8 18"></polyline>
                </svg>
            </div>
            <div class="icon-name">edit-4: code</div>
            <div class="icon-source">Feather/Lucide</div>
        </div>
    </div>

    <h3>Settings Dropdown Preview</h3>
    <div class="notes" style="background: var(--bg-card);">
        <p style="margin-bottom: 16px;"><strong>Example dropdown layout:</strong></p>
        <div style="display: inline-flex; align-items: center; gap: 8px; margin-bottom: 16px;">
            <span style="font-size: 18px; font-weight: 600;">forever-start.fireup</span>
            <button class="btn btn-icon-only" data-tooltip="Settings" style="padding: 4px;">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
                    <circle cx="12" cy="12" r="1"></circle>
                    <circle cx="19" cy="12" r="1"></circle>
                    <circle cx="5" cy="12" r="1"></circle>
                </svg>
            </button>
        </div>
        <div style="background: var(--bg-secondary); border: 1px solid var(--border-color); border-radius: 8px; padding: 8px; display: inline-block;">
            <div style="display: flex; align-items: center; gap: 12px; padding: 8px 12px; border-bottom: 1px solid var(--border-color);">
                <span style="font-family: monospace; font-size: 13px; color: var(--text-secondary);">fireup-tests.yml</span>
                <button class="btn btn-icon-only" data-tooltip="Copy path" style="padding: 4px;">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                    </svg>
                </button>
                <button class="btn btn-icon-only" data-tooltip="Open in editor" style="padding: 4px;">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                        <polyline points="15 3 21 3 21 9"></polyline>
                        <line x1="10" y1="14" x2="21" y2="3"></line>
                    </svg>
                </button>
            </div>
        </div>
    </div>

    <h2>Notes</h2>
    <div class="notes">
        <p><strong>Icon sources:</strong></p>
        <ul>
            <li><a href="https://feathericons.com/" target="_blank">Feather Icons</a> - MIT licensed, clean line icons</li>
            <li><a href="https://lucide.dev/" target="_blank">Lucide</a> - Feather fork with more icons, MIT licensed</li>
            <li><a href="https://fonts.google.com/icons" target="_blank">Material Icons</a> - Apache 2.0, filled style</li>
        </ul>
        <p><strong>Recommendations:</strong></p>
        <ul>
            <li>Use Feather/Lucide for consistency (all stroke-based)</li>
            <li>Claude sunburst should use brand color <code>#da7756</code></li>
            <li>All icon buttons need <code>title</code> attribute for accessibility</li>
        </ul>
        <p><strong>LLM Features:</strong></p>
        <ul>
            <li>"Copy for agent" and "Fix with Claude Code" should include instructions to run <code>fireup --help</code> so the LLM can learn how to use fireup CLI</li>
            <li>This helps agents understand available commands like restart, stop, logs, etc.</li>
        </ul>
    </div>
</body>
</html>
`))

// IconsTestPage renders the icons test page
func IconsTestPage(theme string) string {
	var b strings.Builder
	data := iconsTestData{
		ThemeScript: template.HTML(styles.ThemeScript(theme)),
		ThemeCSS:    template.CSS(styles.ThemeVars + styles.BaseStyles),
	}
	iconsTestTmpl.Execute(&b, data)
	return b.String()
}
