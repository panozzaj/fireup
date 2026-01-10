package styles

import (
	_ "embed"
	"fmt"
)

// ThemeVars contains CSS custom properties for theming
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

// ThemeScript generates inline JavaScript to set theme before CSS loads
func ThemeScript(theme string) string {
	return fmt.Sprintf(`<script>
(function() {
    var theme = '%s';
    if (theme && theme !== 'system') {
        document.documentElement.setAttribute('data-theme', theme);
    }
})();
</script>`, theme)
}

// HeadCSS generates the common CSS for the <head> section including theme vars and base styles
func HeadCSS() string {
	return ThemeVars + "\n" + BaseStyles
}

// LogsCSS generates CSS for logs display including mark highlighting
func LogsCSS() string {
	return MarkHighlight
}
