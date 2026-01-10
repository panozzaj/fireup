package styles

import (
	_ "embed"
)

// ThemeVars contains CSS custom properties for theming
// These should be placed in the :root selector
//
//go:embed theme.css
var ThemeVars string

// BaseStyles contains common base CSS styles
//
//go:embed base.css
var BaseStyles string

// MarkHighlight contains CSS for AI-highlighted log lines
//
//go:embed mark.css
var MarkHighlight string

// ThemeScript returns inline JavaScript to set theme before CSS loads
// The %s placeholder should be replaced with the theme value
const ThemeScript = `
    <script>
        (function() {
            var theme = '%s';
            if (theme && theme !== 'system') {
                document.documentElement.setAttribute('data-theme', theme);
            }
        })();
    </script>`
