package main

const (
	pfAnchorPath     = "/etc/pf.anchors/roost-dev"
	launchdPlistPath = "/Library/LaunchDaemons/dev.roost.pfctl.plist"

	// expectedPfPlistContent is the expected content of the pf LaunchDaemon plist.
	// Used by both isPfPlistOutdated() and runPortsInstall() to stay in sync.
	expectedPfPlistContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.roost.pfctl</string>
    <key>ProgramArguments</key>
    <array>
        <string>/sbin/pfctl</string>
        <string>-e</string>
        <string>-f</string>
        <string>/etc/pf.conf</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`
)

var version = "0.9.0"
