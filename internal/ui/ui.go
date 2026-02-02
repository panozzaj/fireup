package ui

import (
	_ "embed"
	"html/template"
	"net/http"
	"strings"

	"github.com/panozzaj/fireup/internal/icons"
	"github.com/panozzaj/fireup/internal/styles"
)

//go:embed dashboard.css
var dashboardCSS string

//go:embed dashboard.js
var dashboardJS string

// dashboardData contains all data passed to the dashboard template
type dashboardData struct {
	ThemeScript  template.HTML
	ThemeCSS     template.CSS
	DashboardCSS template.CSS
	MarkCSS      template.CSS
	TLD          string
	Port         int
	InitialData  template.JS
	IconsJS      template.JS
	DashboardJS  template.JS
}

// dashboardTmpl is the parsed template for the dashboard page
var dashboardTmpl = template.Must(template.New("dashboard").Parse(dashboardHTML))

// ServeIndex serves the main dashboard HTML with initial app data
func ServeIndex(w http.ResponseWriter, r *http.Request, tld string, port int, initialData []byte, theme string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := dashboardData{
		ThemeScript:  template.HTML(styles.ThemeScript(theme)),
		ThemeCSS:     template.CSS(styles.HeadCSS()),
		DashboardCSS: template.CSS(dashboardCSS),
		MarkCSS:      template.CSS(styles.LogsCSS()),
		TLD:          tld,
		Port:         port,
		InitialData:  template.JS(string(initialData)),
		IconsJS:      template.JS(icons.JSObject()),
		DashboardJS:  template.JS(dashboardJS),
	}

	var b strings.Builder
	dashboardTmpl.Execute(&b, data)
	w.Write([]byte(b.String()))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>fireup</title>
{{.ThemeScript}}
    <style>
{{.ThemeCSS}}
{{.DashboardCSS}}
{{.MarkCSS}}
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>fireup</h1>
            <div class="filter-bar" id="filter-bar">
                <input type="text" id="filter-input" placeholder="Filter..." autocomplete="off">
                <span class="filter-shortcut" id="filter-shortcut"><kbd>/</kbd></span>
            </div>
            <div class="header-actions">
                <span class="connection-status">
                    <span class="connection-dot" id="connection-dot"></span>
                    <span id="connection-text">Connecting...</span>
                </span>
                <button class="theme-toggle" onclick="toggleTheme()" data-tooltip="Toggle theme">
                    <span id="theme-icon">&#9790;</span>
                </button>
            </div>
        </header>
        <main id="apps"></main>
    </div>

    <script>
        // Template variables
        var TLD = '{{.TLD}}';
        var PORT = {{.Port}};
        var INITIAL_DATA = {{.InitialData}};
        {{.IconsJS}}
    </script>
    <script>{{.DashboardJS}}</script>
</body>
</html>
`
