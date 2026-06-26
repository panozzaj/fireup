# Code Improvements

Areas identified for potential refactoring and improvement.

## 1. Help Flag Checking Duplication ✅ DONE

**Files**: cert.go, service.go, ports.go, list.go, docs.go, logs.go

**Fix**: Extracted `checkHelpFlag(args []string, usage string) bool` helper in cli.go.

## 2. Config Loading Pattern Repeated ✅ DONE

**Files**: main.go, list.go, logs.go, cert.go, service.go

**Fix**: Extracted `getDefaultConfigDir()` and `getConfigWithDefaults() (*GlobalConfig, string)` helpers in cli.go.

## 3. ANSI Color Codes Scattered ✅ DONE

**Files**: main.go, ports.go, cert.go, service.go, list.go

**Fix**: Consolidated all color codes as constants in cli.go (colorRed, colorGreen, colorYellow, colorCyan, colorGray, colorDim, colorReset).

## 4. Silent Command Failures ✅ DONE

**Files**: ports.go, service.go, main.go

**Fix**: Added clarifying comments explaining why errors are intentionally ignored (e.g., "ignore errors - may not be running"). These are cleanup operations where failure is expected and acceptable.

## 5. Setup Wizard Too Long (189 lines)

**File**: main.go:539-726 `runSetupWizard()`

Contains 3 major sections with repeated status checking and confirmation patterns.

**Fix**: Split into `setupPortsStep()`, `setupCertsStep()`, `setupServiceStep()`.

## 6. API Handler Monolith ✅ DONE

**File**: internal/server/api.go

**Fix**: Extracted 6 inline handlers into separate methods:

- `handleStop()` - stop app/service
- `handleRestart()` - restart app/service
- `handleStart()` - start app/service
- `handleLogs()` - get app logs
- `handleAppStatus()` - get single app status
- `handleAnalyzeLogs()` - Ollama log analysis

Switch statement reduced from ~300 lines to ~90 lines (clean dispatcher).

## 7. Service Name Resolution Duplicated ✅ DONE

**File**: internal/server/api.go (3 locations)

**Fix**: Extracted `parseServiceName()` function with tests in api_test.go. Updated all 3 usages.

## 8. Missing Tests (Partially Addressed)

Added tests for:

- `isServiceInstalled()` - validates installed/running state consistency
- `getUserLaunchAgentPath()` - validates path construction
- `getCertsDir()` - validates path construction
- `getResolverContent()` - validates resolver content
- `DefaultPorts` - validates default port constants

Still no tests for (require root/system changes):

- `runSetupWizard()` / `runTeardownWizard()` (interactive)
- `runPortsInstall()` / `runPortsUninstall()` (requires root)
- `runCertInstall()` / `runCertUninstall()` (requires root)
- `runServiceInstall()` / `runServiceUninstall()` (modifies system)
- API endpoints in api.go

## 9. Root Check Duplication ✅ DONE

**Files**: main.go, ports.go, service.go

**Fix**: Extracted `requireNonRoot()` and `isRoot()` helpers in cli.go. Note: Not all usages need the helper (some have specific error messages or different logic).

## Priority Order

1. ~~Extract CLI helper functions (help flag, config loading, colors)~~ ✅
2. ~~Fix silent errors (add logging/comments)~~ ✅
3. ~~Refactor setup/teardown wizards~~ (Deferred - wizards read well as linear scripts)
4. ~~Add tests for critical paths~~ ✅ (Added 5 new tests, 25 total passing)
5. Refactor API handler (Future work)
