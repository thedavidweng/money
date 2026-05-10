package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/store"
)

func newSetupCommand(_ context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Initialize money configuration and encrypted database",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := config.Setup(state.configPath, force)
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
			cfg, loadErr := config.Load(config.Options{ConfigPath: result.ConfigPath})
			if loadErr == nil {
				opened, openErr := store.OpenEncrypted(context.Background(), cfg.DatabasePath, cfg.DatabaseEncryptionKeyBytes)
				if openErr == nil {
					opened.Close()
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
					fmt.Fprintf(stdout, "Config written. Database failed to open: %s\nRun `money doctor` to diagnose.\n", openErr)
					return nil
				}
			}

			// Run doctor diagnostics
			diagnostics := runDiagnostics(result.ConfigPath)

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

			fmt.Fprintf(stdout, "Config:   %s\n", result.ConfigPath)
			fmt.Fprintf(stdout, "Secrets:  %s\n", result.EnvPath)
			fmt.Fprintf(stdout, "Database: %s\n", result.DatabasePath)
			if result.SecretCreated {
				fmt.Fprintln(stdout, "Encryption key: created")
			} else {
				fmt.Fprintln(stdout, "Encryption key: already exists")
			}
			if result.DBCreated {
				fmt.Fprintln(stdout, "Database: ready")
			}
			printDiagnostics(stdout, diagnostics)
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
		Use:   "doctor",
		Short: "Check configuration and system health",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fix {
				return runDoctorFix(state, stdout, dryRun)
			}
			diagnostics := runDiagnostics(state.configPath)
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

func runDoctorFix(state *runtimeState, stdout io.Writer, dryRun bool) error {
	result, err := config.Setup(state.configPath, false)
	if dryRun {
		if state.json {
			env := contracts.NewSuccess("doctor", map[string]any{
				"mode":   "fix-dry-run",
				"plan":   result,
				"errors": errString(err),
			})
			return contracts.WriteJSON(stdout, env)
		}
		fmt.Fprintf(stdout, "Would create/verify:\n  Config:   %s\n  Secrets:  %s\n  Database: %s\n", result.ConfigPath, result.EnvPath, result.DatabasePath)
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
	// Open DB to ensure migrations run
	cfg, loadErr := config.Load(config.Options{ConfigPath: result.ConfigPath})
	if loadErr == nil {
		opened, openErr := store.OpenEncrypted(context.Background(), cfg.DatabasePath, cfg.DatabaseEncryptionKeyBytes)
		if openErr == nil {
			opened.Close()
			result.DBCreated = true
		}
	}
	diagnostics := runDiagnostics(result.ConfigPath)
	if state.json {
		env := contracts.NewSuccess("doctor", map[string]any{
			"mode":        "fix",
			"result":      result,
			"diagnostics": diagnostics,
		})
		return contracts.WriteJSON(stdout, env)
	}
	fmt.Fprintln(stdout, "Repairs applied.")
	printDiagnostics(stdout, diagnostics)
	return nil
}

func runDiagnostics(configPath string) []Diagnostic {
	var diags []Diagnostic

	// Config section
	cfg, err := config.Load(config.Options{ConfigPath: configPath})
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
	}

	// Store section
	opened, openErr := store.OpenEncrypted(context.Background(), cfg.DatabasePath, cfg.DatabaseEncryptionKeyBytes)
	if openErr != nil {
		diags = append(diags, Diagnostic{
			Section: "Store", Code: "STORE_OPEN_FAILED", Status: "error",
			Message: openErr.Error(), Category: "internal",
		})
	} else {
		opened.Close()
		diags = append(diags, Diagnostic{
			Section: "Store", Code: "STORE_OK", Status: "ok",
			Message: "Database opened at " + cfg.DatabasePath, Category: "internal",
		})
	}

	// Providers section
	for _, name := range []string{"plaid", "bridge"} {
		pc, ok := cfg.Providers[name]
		if !ok || len(pc.Fields) == 0 {
			diags = append(diags, Diagnostic{
				Section: "Providers", Code: "PROVIDER_NOT_CONFIGURED", Status: "warn",
				Message:  name + " is not configured. Run `money providers configure " + name + "` to add credentials.",
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

func printDiagnostics(w io.Writer, diags []Diagnostic) {
	for _, d := range diags {
		icon := "✓"
		switch d.Status {
		case "warn":
			icon = "⚠"
		case "error":
			icon = "✗"
		}
		fmt.Fprintf(w, "  %s [%s] %s\n", icon, d.Section, d.Message)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func newConfigureCommand(state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure <provider>",
		Short: "Configure provider credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName := args[0]
			var spec config.ProviderSpec
			switch providerName {
			case "plaid":
				spec = config.PlaidSpec
			case "bridge":
				spec = config.BridgeSpec
			default:
				return cliError{
					command:  "providers.configure",
					code:     "UNKNOWN_PROVIDER",
					message:  "unknown provider: " + providerName,
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

			result, err := config.ConfigureProvider(state.configPath, spec, secrets, options, force)
			if err != nil {
				return cliError{
					command:  "providers.configure",
					code:     "CONFIG_WRITE_FAILED",
					message:  err.Error(),
					category: contracts.CategoryConfig,
					exitCode: 1,
				}
			}

			// Run provider-specific diagnostics
			diagnostics := runDiagnostics(result.ConfigPath)
			providerDiags := []Diagnostic{}
			for _, d := range diagnostics {
				if d.Section == "Config" || d.Section == "Providers" {
					providerDiags = append(providerDiags, d)
				}
			}

			if state.json {
				env := contracts.NewSuccess("providers.configure", map[string]any{
					"provider":    result.Provider,
					"keys_written": result.KeysWritten,
					"diagnostics": providerDiags,
				})
				return contracts.WriteJSON(stdout, env)
			}

			if result.KeysWritten > 0 {
				fmt.Fprintf(stdout, "%s configured (%d credentials written).\n", providerName, result.KeysWritten)
			} else {
				fmt.Fprintf(stdout, "%s credentials already present (use --force to overwrite).\n", providerName)
			}
			printDiagnostics(stdout, providerDiags)
			return nil
		},
	}

	// Use generic flag names; the provider argument determines which spec is used
	cmd.Flags().String("client-id", "", "provider client ID")
	cmd.Flags().String("secret", "", "provider secret")
	cmd.Flags().String("client-secret", "", "provider client secret")
	cmd.Flags().String("environment", "", "provider environment (e.g. sandbox)")
	cmd.Flags().String("products", "", "comma-separated products")
	cmd.Flags().String("country-codes", "", "comma-separated country codes")
	cmd.Flags().String("redirect-uri", "", "redirect URI")
	cmd.Flags().String("user-email", "", "bridge user email")
	cmd.Flags().Bool("force", false, "overwrite existing credentials")
	return cmd
}
