package pages

import (
	"html/template"
	"strings"

	"github.com/panozzaj/fireup/internal/icons"
	"github.com/panozzaj/fireup/internal/styles"
)

// welcomeData contains data for the welcome page template
type welcomeData struct {
	ThemeScript template.HTML
	ThemeCSS    template.CSS
	TLD         string
	ConfigDir   string
	IconCheck   template.HTML
}

var welcomeTmpl = template.Must(template.New("welcome").Parse(welcomeHTML))

// Welcome renders the built-in welcome/test page
func Welcome(tld, configDir, theme string) string {
	data := welcomeData{
		ThemeScript: template.HTML(styles.ThemeScript(theme)),
		ThemeCSS:    template.CSS(styles.HeadCSS()),
		TLD:         tld,
		ConfigDir:   configDir,
		IconCheck:   template.HTML(icons.Check),
	}

	var b strings.Builder
	welcomeTmpl.Execute(&b, data)
	return b.String()
}

const welcomeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>fireup is working!</title>
{{.ThemeScript}}
    <style>
{{.ThemeCSS}}
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            max-width: 600px;
            text-align: center;
        }
        .checkmark {
            width: 80px;
            height: 80px;
            margin: 0 auto 24px;
            background: var(--success);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .checkmark svg {
            width: 40px;
            height: 40px;
            stroke: white;
            stroke-width: 3;
        }
        h1 {
            font-size: 28px;
            font-weight: 600;
            margin-bottom: 12px;
        }
        .subtitle {
            font-size: 16px;
            color: var(--text-muted);
            margin-bottom: 32px;
        }
        .card {
            background: var(--bg-secondary);
            border-radius: 12px;
            padding: 24px;
            text-align: left;
            margin-bottom: 24px;
        }
        .card h2 {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 16px;
            color: var(--text-secondary);
        }
        .step {
            display: flex;
            gap: 12px;
            margin-bottom: 16px;
        }
        .step:last-child { margin-bottom: 0; }
        .step-num {
            width: 24px;
            height: 24px;
            background: var(--accent-blue);
            color: white;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 12px;
            font-weight: 600;
            flex-shrink: 0;
        }
        .step-content {
            flex: 1;
        }
        .step-content p {
            font-size: 14px;
            color: var(--text-secondary);
            margin-bottom: 8px;
        }
        code {
            display: block;
            background: var(--bg-tertiary);
            padding: 12px;
            border-radius: 6px;
            font-family: "SF Mono", Monaco, monospace;
            font-size: 13px;
            color: var(--text-primary);
            overflow-x: auto;
        }
        .links {
            display: flex;
            gap: 16px;
            justify-content: center;
        }
        .links a {
            color: var(--accent-blue);
            text-decoration: none;
            font-size: 14px;
        }
        .links a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="checkmark">{{.IconCheck}}</div>
        <h1>fireup is working!</h1>
        <p class="subtitle">Your local development proxy is ready to use.</p>

        <div class="card">
            <h2>Quick Start</h2>
            <div class="step">
                <div class="step-num">1</div>
                <div class="step-content">
                    <p>Create a config file for your app:</p>
                    <code>echo "npm run dev" > {{.ConfigDir}}/myapp</code>
                </div>
            </div>
            <div class="step">
                <div class="step-num">2</div>
                <div class="step-content">
                    <p>Visit your app (it starts automatically):</p>
                    <code>http://myapp.{{.TLD}}</code>
                </div>
            </div>
        </div>

        <div class="links">
            <a href="//fireup.{{.TLD}}">Open Dashboard</a>
            <a href="//fireup.{{.TLD}}" onclick="event.preventDefault(); navigator.clipboard.writeText('fireup docs')">Copy docs command</a>
        </div>
    </div>
</body>
</html>
`
