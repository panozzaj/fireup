package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/panozzaj/fireup/internal/certs"
	"github.com/panozzaj/fireup/internal/diff"
)

func cmdCert(args []string) {
	if len(args) == 0 {
		printCertUsage()
		os.Exit(0)
	}

	subcmd := args[0]
	subargs := args[1:]

	switch subcmd {
	case "install":
		cmdCertInstall(subargs)
	case "uninstall":
		cmdCertUninstall(subargs)
	case "status":
		cmdCertStatus(subargs)
	case "-h", "--help", "help":
		printCertUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown cert command: %s\n\n", subcmd)
		printCertUsage()
		os.Exit(1)
	}
}

func printCertUsage() {
	fmt.Println(`fireup cert - Manage HTTPS certificates

USAGE:
    fireup cert <command>

COMMANDS:
    install     Generate CA and trust it (requires sudo for trust)
    uninstall   Remove CA and certificates
    status      Show certificate status

DESCRIPTION:
    Generates a local Certificate Authority (CA) that fireup uses to
    dynamically create certificates for any domain. After running 'cert install':

    - https://myapp.test will work (any domain!)
    - Certificates are generated on-demand
    - No browser warnings

EXAMPLES:
    fireup cert install    # Generate CA and enable HTTPS
    fireup cert status     # Check certificate status`)
}

func cmdCertInstall(args []string) {
	fs := flag.NewFlagSet("cert install", flag.ExitOnError)

	var tld string
	var configDir string

	fs.StringVar(&tld, "tld", "test", "TLD for certificate (e.g., test)")
	fs.StringVar(&configDir, "dir", getDefaultConfigDir(), "Configuration directory")

	fs.Usage = func() {
		fmt.Println(`fireup cert install - Generate and trust the fireup CA

USAGE:
    fireup cert install [options]

OPTIONS:`)
		fs.PrintDefaults()
		fmt.Println(`
DESCRIPTION:
    Generates a local Certificate Authority (CA) and installs it into
    your system trust store. fireup then uses this CA to dynamically
    generate certificates for any domain on-the-fly.

    After running this:
    - https://myapp.test will work in browsers
    - https://anyother.test will also work
    - No certificate warnings
    - No need to regenerate certs when adding new apps`)
	}

	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			fs.Usage()
			os.Exit(0)
		}
	}

	fs.Parse(args)

	requireNotAgent("cert install")

	// Load saved config for TLD
	globalCfg, _ := loadGlobalConfig(configDir)
	if globalCfg != nil && tld == "test" {
		tld = globalCfg.TLD
	}

	if err := runCertInstall(configDir, tld); err != nil {
		log.Fatalf("Certificate install failed: %v", err)
	}

	// Restart service if running so it picks up the new CA
	restartServiceIfRunning()
}

func cmdCertUninstall(args []string) {
	if checkHelpFlag(args, `fireup cert uninstall - Remove HTTPS certificates

USAGE:
    fireup cert uninstall

Removes the fireup CA and certificates from the config directory.
Also removes the CA from the system trust store (requires sudo).`) {
		os.Exit(0)
	}

	requireNotAgent("cert uninstall")
	if err := runCertUninstall(); err != nil {
		log.Fatalf("Certificate uninstall failed: %v", err)
	}
}

func cmdCertStatus(args []string) {
	if checkHelpFlag(args, `fireup cert status - Show certificate status

USAGE:
    fireup cert status

Shows whether certificates are installed and their details.`) {
		os.Exit(0)
	}

	runCertStatus()
}

func getCertsDir() string {
	return filepath.Join(getDefaultConfigDir(), "certs")
}

// certInstallPlan returns a Plan for certificate installation preview.
// Note: The actual cert generation is handled by the certs package, so we use
// a description-only plan for preview purposes.
func certInstallPlan() *diff.Plan {
	certsDir := getCertsDir()
	plan := diff.NewPlan()
	plan.CreateStatic(filepath.Join(certsDir, "ca.pem"), "Certificate Authority certificate (generated)")
	plan.CreateStatic(filepath.Join(certsDir, "ca-key.pem"), "Certificate Authority private key (generated)")
	return plan
}

// certUninstallPlan returns a Plan for certificate removal.
// The same Plan is used for both preview and execution to ensure they stay in sync.
func certUninstallPlan() *diff.Plan {
	certsDir := getCertsDir()
	plan := diff.NewPlan()
	plan.Delete(filepath.Join(certsDir, "ca.pem"))
	plan.Delete(filepath.Join(certsDir, "ca-key.pem"))
	return plan
}

func runCertInstall(configDir, tld string) error {
	certsDir := getCertsDir()

	// Check if CA already exists
	caPath := filepath.Join(certsDir, "ca.pem")
	if _, err := os.Stat(caPath); err == nil {
		fmt.Printf("CA already exists: %s\n", caPath)
		fmt.Println("To regenerate, run 'fireup cert uninstall' first.")
		return nil
	}

	// Show summary and confirm (? shows full diff)
	plan := certInstallPlan()
	summary := plan.Summary()
	if summary == nil {
		return nil // No changes needed
	}
	summary.Print()
	fmt.Println("This will also add the CA to your system keychain (requires sudo).")
	if !confirmStep("Install HTTPS certificates?") {
		return fmt.Errorf("installation cancelled")
	}

	// Generate CA
	fmt.Println("Generating fireup CA...")
	if err := certs.GenerateCA(certsDir); err != nil {
		return fmt.Errorf("generating CA: %w", err)
	}

	fmt.Println()
	fmt.Printf("CA certificate: %s\n", caPath)
	fmt.Println()

	// Install CA into system trust store (macOS)
	fmt.Println("Installing CA into system trust store...")
	fmt.Println("(You may be prompted for your password)")
	fmt.Println()

	cmd := exec.Command("sudo", "security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", caPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Println()
		fmt.Println("To install manually:")
		fmt.Printf("  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", caPath)
		fmt.Println()
		fmt.Println("Or double-click the CA file and trust it in Keychain Access.")
		return fmt.Errorf("failed to install CA into system trust store")
	}

	fmt.Println()
	fmt.Println("CA installed successfully!")
	fmt.Println()
	fmt.Println("HTTPS is now enabled with dynamic certificate generation.")
	fmt.Println("Any *.test domain will automatically get a valid certificate.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Restart fireup:")
	fmt.Println("     fireup service uninstall && fireup service install")
	fmt.Println()
	fmt.Println("  2. Restart your browser (quit fully and reopen)")
	fmt.Println("     This is needed for browsers to trust the new CA.")
	fmt.Println()
	fmt.Printf("Then visit: https://fireup.%s\n", tld)

	return nil
}

func runCertUninstall() error {
	certsDir := getCertsDir()

	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		fmt.Printf("%s✓ No certificates installed%s\n", colorGreen, colorReset)
		return nil
	}

	// Show summary and confirm (? shows full diff)
	plan := certUninstallPlan()
	if !confirmWithPlan(plan, "Remove HTTPS certificates?") {
		return fmt.Errorf("removal cancelled")
	}

	// Execute the plan for file deletion
	if err := plan.Execute(); err != nil {
		return err
	}

	// Also try to remove the certs directory itself if empty
	os.Remove(certsDir)

	fmt.Printf("%s✓ Certificates removed%s\n", colorGreen, colorReset)
	return nil
}

func runCertStatus() {
	certsDir := getCertsDir()

	// Load TLD from config
	globalCfg, _ := getConfigWithDefaults()
	tld := globalCfg.TLD

	caFile := filepath.Join(certsDir, "ca.pem")
	keyFile := filepath.Join(certsDir, "ca-key.pem")

	fmt.Printf("TLD: .%s\n", tld)
	fmt.Printf("Certs directory: %s\n", certsDir)
	fmt.Println()

	caExists := false

	if info, err := os.Stat(caFile); err == nil {
		caExists = true
		fmt.Printf("CA Certificate: %s\n", caFile)
		fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("CA Certificate: not found")
	}

	if _, err := os.Stat(keyFile); err == nil {
		fmt.Printf("CA Key: %s\n", keyFile)
	} else {
		fmt.Println("CA Key: not found")
	}

	fmt.Println()
	if caExists {
		fmt.Println("Status: HTTPS enabled (dynamic certificate generation)")
		fmt.Printf("  https://myapp.%s will work\n", tld)
		fmt.Printf("  https://anyapp.%s will work\n", tld)
		fmt.Println()
		fmt.Println("Certificates are generated on-demand for each domain.")
	} else {
		fmt.Println("Status: HTTPS not configured")
		fmt.Println("  Run 'fireup cert install' to enable HTTPS")
	}
}
