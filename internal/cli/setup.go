package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/plaidlogin"
	"github.com/thedavidweng/money/internal/prompt"
	"github.com/thedavidweng/money/internal/store"
)

func newSetupCommand(_ context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "setup",
		Short:   "Initialize money configuration and encrypted database",
		Example: "  money setup\n  money setup --force",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := config.Setup(state.configPath, state.profile, force)
			if err != nil {
				return cliError{
					command:  "setup",
					code:     "CONFIG_WRITE_FAILED",
					message:  err.Error(),
					category: contracts.CategoryConfig,
					exitCode: 1,
				}
			}

			// Open the store to run migrations
			cfg, loadErr := config.Load(config.Options{ConfigPath: result.ConfigPath, Profile: state.profile})
			if loadErr == nil {
				opened, openErr := store.OpenEncrypted(context.Background(), cfg.DatabasePath, cfg.DatabaseEncryptionKeyBytes)
				if openErr == nil {
					_ = opened.Close()
					result.DBCreated = true
				} else if state.json {
					env := contracts.NewSuccess("setup", result)
					env.Warnings = append(env.Warnings, contracts.Warning{
						Code:     "DB_OPEN_FAILED",
						Message:  openErr.Error(),
						Category: contracts.CategoryInternal,
					})
					return contracts.WriteJSON(stdout, env)
				} else {
					_, _ = fmt.Fprintf(stdout, "Config written. Database failed to open: %s\nRun `money doctor` to diagnose.\n", openErr)
					return nil
				}
			}

			// Run doctor diagnostics
			diagnostics := runDiagnostics(result.ConfigPath, state.profile)

			if state.json {
				env := contracts.NewSuccess("setup", map[string]any{
					"config_path":    result.ConfigPath,
					"env_path":       result.EnvPath,
					"database_path":  result.DatabasePath,
					"secret_created": result.SecretCreated,
					"db_created":     result.DBCreated,
					"diagnostics":    diagnostics,
				})
				return contracts.WriteJSON(stdout, env)
			}

			_, _ = fmt.Fprintf(stdout, "Config:   %s\n", result.ConfigPath)
			_, _ = fmt.Fprintf(stdout, "Secrets:  %s\n", result.EnvPath)
			_, _ = fmt.Fprintf(stdout, "Database: %s\n", result.DatabasePath)
			if result.SecretCreated {
				_, _ = fmt.Fprintln(stdout, "Encryption key: created")
			} else {
				_, _ = fmt.Fprintln(stdout, "Encryption key: already exists")
			}
			if result.DBCreated {
				_, _ = fmt.Fprintln(stdout, "Database: ready")
			}
			printDiagnostics(stdout, diagnostics)

			// Post-setup interactive wizard: offer to configure providers
			if !state.json && state.stdin != nil {
				if err := runSetupWizard(cmd.Context(), state, stdout, diagnostics); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing encryption key")
	return cmd
}

// Diagnostic represents a single doctor check result.
type Diagnostic struct {
	Section  string `json:"section"`
	Code     string `json:"code"`
	Status   string `json:"status"` // ok, warn, error
	Message  string `json:"message"`
	Category string `json:"category"`
}

func newDoctorCommand(_ context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var fix, dryRun bool
	cmd := &cobra.Command{
		Use:     "doctor",
		Short:   "Check configuration and system health",
		Example: "  money doctor\n  money doctor --fix\n  money doctor --fix --dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Only log to slog when stderr is a TTY (not piped/combined).
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			if f, ok := state.stderr.(*os.File); ok && isTerminal(f) {
				logger = slog.New(slog.NewTextHandler(state.stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
			}
			if fix {
				return runDoctorFix(state, stdout, logger, dryRun)
			}
			diagnostics := runDiagnostics(state.configPath, state.profile)
			logDiagnostics(logger, diagnostics)
			if state.json {
				env := contracts.NewSuccess("doctor", map[string]any{"diagnostics": diagnostics})
				return contracts.WriteJSON(stdout, env)
			}
			printDiagnostics(stdout, diagnostics)
			hasError := false
			for _, d := range diagnostics {
				if d.Status == "error" {
					hasError = true
					break
				}
			}
			if hasError {
				return cliExit{exitCode: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "attempt to repair common issues")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show repair plan without writing")
	return cmd
}

func runDoctorFix(state *runtimeState, stdout io.Writer, logger *slog.Logger, dryRun bool) error {
	result, err := config.Setup(state.configPath, state.profile, false)
	if dryRun {
		if state.json {
			env := contracts.NewSuccess("doctor", map[string]any{
				"mode":   "fix-dry-run",
				"plan":   result,
				"errors": errString(err),
			})
			return contracts.WriteJSON(stdout, env)
		}
		_, _ = fmt.Fprintf(stdout, "Would create/verify:\n  Config:   %s\n  Secrets:  %s\n  Database: %s\n", result.ConfigPath, result.EnvPath, result.DatabasePath)
		return nil
	}
	if err != nil {
		return cliError{
			command:  "doctor",
			code:     "CONFIG_WRITE_FAILED",
			message:  err.Error(),
			category: contracts.CategoryConfig,
			exitCode: 1,
		}
	}
	// Open DB to ensure migrations run, then fix links and sync.
	ctx := context.Background()
	cfg, loadErr := config.Load(config.Options{ConfigPath: result.ConfigPath, Profile: state.profile})
	if loadErr == nil {
		opened, openErr := store.OpenEncrypted(ctx, cfg.DatabasePath, cfg.DatabaseEncryptionKeyBytes)
		if openErr == nil {
			result.DBCreated = true
			linksFixed, _ := runDoctorFixLinks(ctx, opened)
			syncFixed, _ := runDoctorFixSync(ctx, opened)
			_ = opened.Close()
			result.LinksFixed = linksFixed
			result.SyncFixed = syncFixed
			if linksFixed > 0 {
				logFixResult(logger, "links", linksFixed)
			}
			if syncFixed > 0 {
				logFixResult(logger, "sync", syncFixed)
			}
		}
	}
	diagnostics := runDiagnostics(result.ConfigPath, state.profile)
	logDiagnostics(logger, diagnostics)
	if state.json {
		env := contracts.NewSuccess("doctor", map[string]any{
			"mode":        "fix",
			"result":      result,
			"diagnostics": diagnostics,
		})
		return contracts.WriteJSON(stdout, env)
	}
	_, _ = fmt.Fprintln(stdout, "Repairs applied.")
	printDiagnostics(stdout, diagnostics)
	return nil
}

func runDiagnostics(configPath string, profile string) []Diagnostic {
	var diags []Diagnostic

	// Config section
	cfg, err := config.Load(config.Options{ConfigPath: configPath, Profile: profile})
	if err != nil {
		diags = append(diags, Diagnostic{
			Section: "Config", Code: "CONFIG_LOAD_FAILED", Status: "error",
			Message: err.Error(), Category: "config",
		})
		return diags
	}
	diags = append(diags, Diagnostic{
		Section: "Config", Code: "CONFIG_OK", Status: "ok",
		Message: "Config loaded from " + cfg.ConfigPath, Category: "config",
	})

	// Config warnings
	for _, w := range cfg.Warnings {
		diags = append(diags, Diagnostic{
			Section: "Warnings", Code: w.Code, Status: "warn",
			Message: w.Message, Category: string(w.Category),
		})
	}

	// Env file permissions
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(cfg.EnvPath); err == nil {
			perm := info.Mode().Perm()
			if perm&0o077 != 0 {
				diags = append(diags, Diagnostic{
					Section: "Warnings", Code: "ENV_FILE_PERMISSIONS",
					Status:   "warn",
					Message:  fmt.Sprintf("%s has permissions %o; recommend 600. Run `money doctor --fix` to repair.", cfg.EnvPath, perm),
					Category: "config",
				})
			}
		}
		authPath := plaidlogin.DashboardAuthPath(cfg.ConfigPath)
		if info, err := os.Stat(authPath); err == nil {
			perm := info.Mode().Perm()
			if perm&0o077 != 0 {
				diags = append(diags, Diagnostic{
					Section: "Warnings", Code: "PLAID_DASHBOARD_AUTH_PERMISSIONS",
					Status:   "warn",
					Message:  fmt.Sprintf("%s has permissions %o; recommend 600.", authPath, perm),
					Category: "config",
				})
			}
		}
	}

	// Store section
	ctx := context.Background()
	opened, openErr := store.OpenEncrypted(ctx, cfg.DatabasePath, cfg.DatabaseEncryptionKeyBytes)
	if openErr != nil {
		diags = append(diags, Diagnostic{
			Section: "Store", Code: "STORE_OPEN_FAILED", Status: "error",
			Message: openErr.Error(), Category: "internal",
		})
		return diags
	}
	defer func() { _ = opened.Close() }()
	diags = append(diags, Diagnostic{
		Section: "Store", Code: "STORE_OK", Status: "ok",
		Message: "Database opened at " + cfg.DatabasePath, Category: "internal",
	})

	// Links section
	diags = append(diags, appendLinksDiagnostics(ctx, opened)...)

	// Sync section
	diags = append(diags, appendSyncDiagnostics(ctx, opened)...)

	// Providers section
	for _, name := range []string{"plaid", "bridge"} {
		pc, ok := cfg.Providers[name]
		if !ok || len(pc.Fields) == 0 {
			var helpURL string
			if spec, ok := config.ProviderSpecByName(name); ok {
				helpURL = spec.HelpURL
			}
			msg := name + " is not configured. Run `money providers configure " + name + "` to add credentials."
			if helpURL != "" {
				msg += " Get credentials: " + helpURL
			}
			diags = append(diags, Diagnostic{
				Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn",
				Message:  msg,
				Category: "config",
			})
		} else {
			diags = append(diags, Diagnostic{
				Section: "Providers", Code: "PROVIDER_CONFIGURED", Status: "ok",
				Message: name + " credentials present.", Category: "config",
			})
		}
	}

	return diags
}

func appendLinksDiagnostics(ctx context.Context, db *store.SQLiteStore) []Diagnostic {
	var diags []Diagnostic
	items, err := db.ListProviderItems(ctx, store.ProviderItemQuery{})
	if err != nil {
		diags = append(diags, Diagnostic{
			Section: "Links", Code: "LINKS_QUERY_FAILED", Status: "error",
			Message: err.Error(), Category: "internal",
		})
		return diags
	}
	if len(items) == 0 {
		diags = append(diags, Diagnostic{
			Section: "Links", Code: "LINKS_NONE", Status: "ok",
			Message: "No linked provider items.", Category: "internal",
		})
		return diags
	}
	// Count items per provider.
	counts := map[string]int{}
	for _, item := range items {
		counts[item.Provider]++
	}
	for provider, count := range counts {
		diags = append(diags, Diagnostic{
			Section: "Links", Code: "LINKS_OK", Status: "ok",
			Message: fmt.Sprintf("%s: %d linked item(s)", provider, count), Category: "internal",
		})
	}
	return diags
}

func appendSyncDiagnostics(ctx context.Context, db *store.SQLiteStore) []Diagnostic {
	var diags []Diagnostic
	items, err := db.ListProviderItems(ctx, store.ProviderItemQuery{})
	if err != nil {
		diags = append(diags, Diagnostic{
			Section: "Sync", Code: "SYNC_QUERY_FAILED", Status: "error",
			Message: err.Error(), Category: "internal",
		})
		return diags
	}
	if len(items) == 0 {
		diags = append(diags, Diagnostic{
			Section: "Sync", Code: "SYNC_NO_ITEMS", Status: "ok",
			Message: "No provider items to sync.", Category: "internal",
		})
		return diags
	}

	runs, err := db.LatestSyncRuns(ctx)
	if err != nil {
		diags = append(diags, Diagnostic{
			Section: "Sync", Code: "SYNC_QUERY_FAILED", Status: "error",
			Message: err.Error(), Category: "internal",
		})
		return diags
	}

	// Index latest runs by provider item ID.
	latestByItem := map[string]store.SyncRunSummary{}
	for _, r := range runs {
		latestByItem[r.ProviderItemID] = r
	}

	for _, item := range items {
		run, ok := latestByItem[item.ID]
		if !ok {
			diags = append(diags, Diagnostic{
				Section: "Sync", Code: "SYNC_NO_RUNS", Status: "warn",
				Message:  fmt.Sprintf("%s (%s): never synced", item.ID, item.Provider),
				Category: "internal",
			})
			continue
		}
		if run.Status == "error" {
			diags = append(diags, Diagnostic{
				Section: "Sync", Code: "SYNC_LAST_ERROR", Status: "error",
				Message:  fmt.Sprintf("%s (%s): last sync failed at %s — %s", item.ID, item.Provider, run.StartedAt, run.ErrorMessage),
				Category: "internal",
			})
		} else {
			diags = append(diags, Diagnostic{
				Section: "Sync", Code: "SYNC_OK", Status: "ok",
				Message:  fmt.Sprintf("%s (%s): last sync ok at %s", item.ID, item.Provider, run.StartedAt),
				Category: "internal",
			})
		}
	}
	return diags
}

// runDoctorFixLinks removes provider items with non-active status.
// Returns the number of items removed.
func runDoctorFixLinks(ctx context.Context, db *store.SQLiteStore) (int, error) {
	items, err := db.ListProviderItems(ctx, store.ProviderItemQuery{})
	if err != nil {
		return 0, err
	}
	fixed := 0
	for _, item := range items {
		if item.Status != "active" {
			if err := db.RemoveProviderItem(ctx, item.ID); err != nil {
				return fixed, err
			}
			fixed++
		}
	}
	return fixed, nil
}

// runDoctorFixSync marks stuck sync runs (no finished_at) as "interrupted".
// Returns the number of runs updated.
func runDoctorFixSync(ctx context.Context, db *store.SQLiteStore) (int, error) {
	return db.MarkStuckSyncRunsInterrupted(ctx)
}

// logDiagnostics emits each diagnostic as a structured slog entry.
func logDiagnostics(logger *slog.Logger, diags []Diagnostic) {
	for _, d := range diags {
		level := slog.LevelInfo
		switch d.Status {
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
		logger.LogAttrs(context.Background(), level, "doctor diagnostic",
			slog.String("section", d.Section),
			slog.String("code", d.Code),
			slog.String("status", d.Status),
			slog.String("message", d.Message),
			slog.String("category", d.Category),
		)
	}
}

// logFixResult logs the result of a doctor fix operation.
func logFixResult(logger *slog.Logger, target string, count int) {
	logger.Info("doctor fix",
		slog.String("target", target),
		slog.Int("fixed", count),
	)
}

func printDiagnostics(w io.Writer, diags []Diagnostic) {
	for _, d := range diags {
		icon := "✓"
		switch d.Status {
		case "warn":
			icon = "⚠"
		case "error":
			icon = "✗"
		}
		_, _ = fmt.Fprintf(w, "  %s [%s] %s\n", icon, d.Section, d.Message)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func runSetupWizard(ctx context.Context, state *runtimeState, stdout io.Writer, diags []Diagnostic) error {
	// Count unconfigured providers
	unconfigured := []string{}
	for _, d := range diags {
		if d.Section == "Providers" && d.Status == "warn" && strings.HasPrefix(d.Code, "PROVIDER_NOT_CONFIGURED") {
			for _, name := range []string{"plaid", "bridge"} {
				if strings.Contains(d.Message, name) {
					unconfigured = append(unconfigured, name)
					break
				}
			}
		}
	}
	if len(unconfigured) == 0 {
		return nil
	}

	reader := bufio.NewReader(state.stdin)
	selector := state.prompter
	if selector == nil {
		selector = prompt.HuhSelector{Input: state.stdin, Output: stdout}
	}
	for {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintf(stdout, "No providers are configured yet. To link financial institutions, you need at least one provider.\n")
		_, _ = fmt.Fprintln(stdout)
		choices := make([]prompt.Choice, 0, len(unconfigured)+1)
		for _, name := range unconfigured {
			if spec, ok := config.ProviderSpecByName(name); ok {
				choices = append(choices, prompt.Choice{Label: name + " - " + spec.HelpURL, Value: name})
			}
		}
		choices = append(choices, prompt.Choice{Label: "Skip for now", Value: "skip"})
		choice, err := selector.Select("Choose a provider to configure", choices)
		if err != nil {
			return fmt.Errorf("failed to read selection: %w", err)
		}
		if choice != "skip" {
			providerName := choice
			if providerName == "plaid" {
				method, err := selector.Select("How do you want to configure Plaid?", []prompt.Choice{
					{Label: "Sign in with Plaid Dashboard and fetch API keys automatically", Value: "dashboard"},
					{Label: "Paste client ID and secret manually", Value: "manual"},
					{Label: "Skip Plaid for now", Value: "skip"},
				})
				if err != nil {
					return err
				}
				switch method {
				case "skip":
					_, _ = fmt.Fprintln(stdout)
					_, _ = fmt.Fprintf(stdout, "You can always run `money providers configure plaid` later.\n")
					return nil
				case "dashboard":
					stderr := state.stderr
					if stderr == nil {
						stderr = io.Discard
					}
					return runPlaidLoginCLI(ctx, state, stdout, stderr, plaidLoginCLIOptions{
						CommandName: "plaid.login",
						Environment: "sandbox",
					})
				case "manual":
				default:
					return fmt.Errorf("unknown Plaid setup method %q", method)
				}
			}
			// Build a minimal cobra.Command just to hold flags for runInteractiveProviderConfigure
			fakeCmd := &cobra.Command{}
			fakeCmd.Flags().Bool("force", false, "")
			selectedSpec, _ := providerSpecByName(providerName)
			for _, field := range selectedSpec.SecretFields {
				fakeCmd.Flags().String(strings.ReplaceAll(field, "_", "-"), "", "")
			}
			for field := range selectedSpec.OptionalFields {
				fakeCmd.Flags().String(strings.ReplaceAll(field, "_", "-"), "", "")
			}
			if err := runInteractiveProviderConfigure(state, stdout, providerName, fakeCmd, reader); err != nil {
				return err
			}
			// Refresh diagnostics to see if there are still unconfigured providers
			_, loadErr := config.Load(config.Options{ConfigPath: state.configPath, Profile: state.profile})
			if loadErr == nil {
				diags = runDiagnostics(state.configPath, state.profile)
				unconfigured = []string{}
				for _, d := range diags {
					if d.Section == "Providers" && d.Status == "warn" && strings.HasPrefix(d.Code, "PROVIDER_NOT_CONFIGURED") {
						for _, name := range []string{"plaid", "bridge"} {
							if strings.Contains(d.Message, name) {
								unconfigured = append(unconfigured, name)
								break
							}
						}
					}
				}
			}
			if len(unconfigured) == 0 {
				_, _ = fmt.Fprintln(stdout)
				_, _ = fmt.Fprintf(stdout, "All providers configured. Run `money link <institution>` to connect an institution.\n")
				return nil
			}
			_, _ = fmt.Fprintln(stdout)
			_, _ = fmt.Fprintf(stdout, "Would you like to configure another provider? (y/n): ")
			resp, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}
			if strings.ToLower(strings.TrimSpace(resp)) != "y" {
				_, _ = fmt.Fprintln(stdout)
				_, _ = fmt.Fprintf(stdout, "You can always run `money providers configure <provider>` later.\n")
				return nil
			}
		} else {
			_, _ = fmt.Fprintln(stdout)
			_, _ = fmt.Fprintf(stdout, "You can always run `money providers configure <provider>` later.\n")
			return nil
		}
	}
}

func providerSpecByName(name string) (config.ProviderSpec, error) {
	switch name {
	case "plaid":
		return config.PlaidSpec, nil
	case "bridge":
		return config.BridgeSpec, nil
	}
	return config.ProviderSpec{}, fmt.Errorf("unknown provider: %s", name)
}

func promptForProviderCredentials(stdout io.Writer, reader *bufio.Reader, spec config.ProviderSpec, secrets map[string]string) error {
	// Skip entirely if all fields are already filled.
	allFilled := true
	for _, field := range spec.SecretFields {
		if secrets[field] == "" {
			allFilled = false
			break
		}
	}
	if allFilled {
		return nil
	}

	if spec.HelpURL != "" {
		_, _ = fmt.Fprintf(stdout, "\n! %s credentials are required.\n", spec.Name)
		_, _ = fmt.Fprintf(stdout, "\n  1. Open %s in your browser\n", spec.HelpURL)
		_, _ = fmt.Fprintf(stdout, "     (or copy the URL and open it manually)\n")
		if err := openBrowser(spec.HelpURL); err != nil {
			_, _ = fmt.Fprintf(stdout, "     ! Could not open browser automatically.\n")
		}
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintf(stdout, "  2. Copy the following fields from your %s dashboard:\n", spec.Name)
		for i, field := range spec.SecretFields {
			_, _ = fmt.Fprintf(stdout, "     %d. %s\n", i+1, strings.ReplaceAll(field, "_", "-"))
		}
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintf(stdout, "  Press Enter once you have copied them.")
		if _, err := reader.ReadString('\n'); err != nil {
			return fmt.Errorf("failed to read ready signal: %w", err)
		}
	}

	_, _ = fmt.Fprintln(stdout)
	for _, field := range spec.SecretFields {
		if secrets[field] != "" {
			continue
		}
		for {
			_, _ = fmt.Fprintf(stdout, "%s: ", strings.ReplaceAll(field, "_", "-"))
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", field, err)
			}
			secrets[field] = strings.TrimSpace(input)
			if secrets[field] != "" {
				break
			}
			_, _ = fmt.Fprintf(stdout, "  ! %s is required. Please provide a value.\n", strings.ReplaceAll(field, "_", "-"))
		}
	}
	_, _ = fmt.Fprintln(stdout)
	return nil
}

func runInteractiveProviderConfigure(state *runtimeState, stdout io.Writer, providerName string, cmd *cobra.Command, reader *bufio.Reader) error {
	spec, err := providerSpecByName(providerName)
	if err != nil {
		return cliError{
			command:  "providers.configure",
			code:     "UNKNOWN_PROVIDER",
			message:  err.Error(),
			category: contracts.CategoryValidation,
			exitCode: 2,
		}
	}

	force, _ := cmd.Flags().GetBool("force")
	secrets := map[string]string{}
	options := map[string]string{}
	for _, field := range spec.SecretFields {
		val, _ := cmd.Flags().GetString(strings.ReplaceAll(field, "_", "-"))
		secrets[field] = val
	}
	for field := range spec.OptionalFields {
		val, _ := cmd.Flags().GetString(strings.ReplaceAll(field, "_", "-"))
		options[field] = val
	}

	// Interactive prompt for missing secrets
	if !state.json && state.stdin != nil {
		needsPrompt := false
		for _, field := range spec.SecretFields {
			if secrets[field] == "" {
				needsPrompt = true
				break
			}
		}
		if needsPrompt {
			if reader == nil {
				reader = bufio.NewReader(state.stdin)
			}
			if err := promptForProviderCredentials(stdout, reader, spec, secrets); err != nil {
				return err
			}
		}
	}

	// Final validation before writing
	var missing []string
	for _, field := range spec.SecretFields {
		if secrets[field] == "" {
			missing = append(missing, strings.ReplaceAll(field, "_", "-"))
		}
	}
	if len(missing) > 0 {
		return cliError{
			command:  "providers.configure",
			code:     "MISSING_CREDENTIALS",
			message:  fmt.Sprintf("%s is missing required credentials: %s. Run interactively or pass them as flags.", providerName, strings.Join(missing, ", ")),
			category: contracts.CategoryValidation,
			exitCode: 2,
		}
	}
	if err := confirmProviderCredentialOverwrite(state, stdout, spec, &force); err != nil {
		return err
	}

	result, err := config.ConfigureProvider(state.configPath, state.profile, spec, secrets, options, force)
	if err != nil {
		return cliError{
			command:  "providers.configure",
			code:     "CONFIG_WRITE_FAILED",
			message:  err.Error(),
			category: contracts.CategoryConfig,
			exitCode: 1,
		}
	}

	diagnostics := runDiagnostics(result.ConfigPath, state.profile)
	providerDiags := []Diagnostic{}
	for _, d := range diagnostics {
		if d.Section == "Config" || d.Section == "Providers" {
			providerDiags = append(providerDiags, d)
		}
	}

	if state.json {
		env := contracts.NewSuccess("providers.configure", map[string]any{
			"provider":     result.Provider,
			"keys_written": result.KeysWritten,
			"diagnostics":  providerDiags,
		})
		return contracts.WriteJSON(stdout, env)
	}

	if result.KeysWritten > 0 {
		_, _ = fmt.Fprintf(stdout, "%s configured (%d credentials written).\n", providerName, result.KeysWritten)
	} else {
		_, _ = fmt.Fprintf(stdout, "%s credentials already present (use --force to overwrite).\n", providerName)
	}
	printDiagnostics(stdout, providerDiags)
	return nil
}

func confirmProviderCredentialOverwrite(state *runtimeState, output io.Writer, spec config.ProviderSpec, force *bool) error {
	if *force {
		return nil
	}
	conflicts, envPath, err := config.ProviderCredentialConflicts(state.configPath, state.profile, spec)
	if err != nil {
		return err
	}
	if len(conflicts) == 0 {
		return nil
	}
	message := fmt.Sprintf("%s credentials already exist in %s; rerun with --force to overwrite %s.", spec.Name, envPath, strings.Join(conflicts, ", "))
	if state.json || state.stdin == nil {
		return cliError{
			command:   "providers.configure",
			code:      "CONFIRMATION_REQUIRED",
			message:   message,
			category:  contracts.CategorySafety,
			retryable: false,
			exitCode:  10,
		}
	}
	selector := state.prompter
	if selector == nil {
		selector = prompt.HuhSelector{Input: state.stdin, Output: output}
	}
	choice, err := selector.Select("Overwrite existing "+spec.Name+" credentials?", []prompt.Choice{
		{Label: "No, keep existing credentials", Value: "no"},
		{Label: "Yes, overwrite credentials", Value: "yes"},
	})
	if err != nil {
		return err
	}
	if choice != "yes" {
		return cliError{
			command:   "providers.configure",
			code:      "CONFIRMATION_REQUIRED",
			message:   message,
			category:  contracts.CategorySafety,
			retryable: false,
			exitCode:  10,
		}
	}
	*force = true
	return nil
}

func newConfigureCommand(state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "configure <provider>",
		Short:   "Configure provider credentials",
		Example: "  money providers configure plaid\n  money providers configure bridge --force",
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"plaid", "bridge"}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var reader *bufio.Reader
			if state.stdin != nil {
				reader = bufio.NewReader(state.stdin)
			}
			return runInteractiveProviderConfigure(state, stdout, args[0], cmd, reader)
		},
	}

	// Use generic flag names; the provider argument determines which spec is used
	cmd.Flags().String("client-id", "", "provider client ID")
	cmd.Flags().String("secret", "", "provider secret")
	cmd.Flags().String("client-secret", "", "provider client secret")
	cmd.Flags().String("environment", "", "provider environment (e.g. sandbox)")
	_ = cmd.RegisterFlagCompletionFunc("environment", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"sandbox", "production"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().String("products", "", "comma-separated products")
	cmd.Flags().String("country-codes", "", "comma-separated country codes")
	cmd.Flags().String("redirect-uri", "", "redirect URI")
	cmd.Flags().String("user-email", "", "bridge user email")
	cmd.Flags().Bool("force", false, "overwrite existing credentials")
	return cmd
}
