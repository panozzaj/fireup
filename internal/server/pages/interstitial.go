package pages

import (
	_ "embed"
	"html/template"
	"strings"

	"github.com/panozzaj/roost-dev/internal/icons"
	"github.com/panozzaj/roost-dev/internal/logo"
	"github.com/panozzaj/roost-dev/internal/styles"
)

//go:embed interstitial.css
var interstitialCSS string

//go:embed interstitial.js
var interstitialJS string

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
	IconsJS     template.JS
	Script      template.JS
	// Icons for HTML template
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
        <div class="logo"><a href="//roost-dev.{{.TLD}}/" data-tooltip="roost-dev dashboard">{{.Logo}}</a></div>
        <div class="title-row">
            <h1>{{.DisplayName}}</h1>
            <div class="settings-dropdown">
                <button class="settings-btn" onclick="toggleSettings()" aria-label="Settings" data-tooltip="Settings">{{.IconGear}}</button>
                <div class="settings-menu" id="settings-menu">
                    <span class="settings-filename" id="settings-filename">{{.ConfigName}}.yml</span>
                    <button class="settings-action" id="copy-path-btn" onclick="copyConfigPath(event)" data-tooltip="Copy absolute path">{{.IconCopy}}</button>
                    <button class="settings-action" id="open-editor-btn" onclick="openConfig(event)" data-tooltip="Open in editor">{{.IconExternalLink}}</button>
                </div>
            </div>
        </div>
        <div class="status" id="status">{{.StatusText}}...</div>
        <div class="spinner" id="spinner"></div>
        <div class="logs-wrapper">
            <div class="logs-header">
                <div class="logs-title">Logs</div>
                <div class="logs-buttons" id="logs-buttons" style="display: none;">
                    <button class="btn icon-btn" id="copy-btn" onclick="copyLogs()" data-tooltip="Copy logs">{{.IconClipboard}}</button>
                    <button class="btn icon-btn" id="copy-agent-btn" onclick="copyForAgent()" data-tooltip="Copy for agent">{{.IconClipboardAgent}}</button>
                    <button class="btn icon-btn claude-btn" id="fix-btn" onclick="fixWithClaudeCode()" style="display: none;" data-tooltip="Fix with Claude Code">{{.IconClaude}}</button>
                </div>
            </div>
            <div class="logs" id="logs">
                <div class="logs-content" id="logs-content"><span class="logs-empty">Waiting for output...</span></div>
            </div>
        </div>
        <button class="btn btn-primary retry-btn" id="retry-btn" onclick="restartAndRetry()">Restart</button>
    </div>
    <script>{{.IconsJS}}</script>
    <script>{{.Script}}</script>
</body>
</html>
`))

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
		IconsJS:     template.JS(icons.JSObject()),
		Script:      template.JS(interstitialJS),
		// Icons for HTML template
		IconGear:           template.HTML(icons.Gear),
		IconCopy:           template.HTML(icons.Copy),
		IconExternalLink:   template.HTML(icons.ExternalLink),
		IconClipboard:      template.HTML(icons.Clipboard),
		IconClipboardAgent: template.HTML(icons.ClipboardAgent),
		IconClaude:         template.HTML(icons.Claude),
	}
	interstitialTmpl.Execute(&b, data)
	return b.String()
}
