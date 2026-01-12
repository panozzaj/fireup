package pages

import (
	"html/template"
	"strings"

	"github.com/panozzaj/roost-dev/internal/logo"
	"github.com/panozzaj/roost-dev/internal/styles"
)

// errorPageData holds data for the error page template
type errorPageData struct {
	Title       string
	Message     string
	Hint        template.HTML
	TLD         string
	ThemeScript template.HTML
	ThemeCSS    template.CSS
	PageCSS     template.CSS
	Logo        template.HTML
}

var errorPageTmpl = template.Must(template.New("error").Parse(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Title}}</title>
{{.ThemeScript}}
    <style>
{{.ThemeCSS}}
{{.PageCSS}}
    </style>
</head>
<body>
    <div class="container">
        <div class="logo"><a href="http://roost-dev.{{.TLD}}">{{.Logo}}</a></div>
        <h1>{{.Title}}</h1>
        <p class="message">{{.Message}}</p>
        {{.Hint}}
    </div>
</body>
</html>
`))

const errorPageCSS = `
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
.message {
    font-size: 16px;
    color: var(--text-secondary);
    margin-bottom: 16px;
}
.hint {
    font-family: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, monospace;
    font-size: 13px;
    color: var(--text-muted);
    background: var(--border-color);
    padding: 12px 16px;
    border-radius: 6px;
    display: inline-block;
}
`

// Error renders the error page
func Error(title, message, hint, tld, theme string) string {
	var b strings.Builder
	data := errorPageData{
		Title:       title,
		Message:     message,
		Hint:        template.HTML(hint),
		TLD:         tld,
		ThemeScript: template.HTML(styles.ThemeScript(theme)),
		ThemeCSS:    template.CSS(styles.ThemeVars + styles.BaseStyles),
		PageCSS:     template.CSS(errorPageCSS),
		Logo:        template.HTML(logo.Web()),
	}
	errorPageTmpl.Execute(&b, data)
	return b.String()
}
