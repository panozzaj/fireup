package main

import (
	"fmt"
	"os"
	"os/exec"
)

func cmdOpen(args []string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fmt.Println(`fireup open - Open an app or the dashboard in a web browser

USAGE:
    fireup open [name]

If no name is given and the current directory matches a configured app,
opens that app. Otherwise opens the fireup dashboard.`)
			os.Exit(0)
		}
	}

	globalCfg, _ := getConfigWithDefaults()

	var appName string
	if len(args) > 0 {
		appName = args[0]
	} else if resolved, found := resolveAppFromCwd(); found {
		fmt.Fprintf(os.Stderr, "(detected %s from current directory)\n", resolved)
		appName = resolved
	}

	var url string
	if appName != "" {
		url = fmt.Sprintf("http://%s.%s", appName, globalCfg.TLD)
	} else {
		url = fmt.Sprintf("http://fireup.%s", globalCfg.TLD)
	}

	fmt.Printf("Opening %s ...\n", url)
	exec.Command("open", url).Start()
}
