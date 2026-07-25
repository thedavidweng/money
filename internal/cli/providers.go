package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/plaidlogin"
	"github.com/thedavidweng/money/internal/prompt"
	"github.com/thedavidweng/money/internal/providers"
)

type plaidLoginCLIOptions struct {
	CommandName  string
	NoOpen       bool
	Team         string
	Environment  string
	Products     string
	CountryCodes string
	RedirectURI  string
	Force        bool
}

var runPlaidLoginCLI = runPlaidLoginLive

func newProvidersCommand(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "providers",
		Short:   "Manage financial data providers",
		Example: "  money providers plaid link\n  money providers bridge link\n  money providers configure plaid",
	}
	cmd.AddCommand(newPlaidProviderCommand(ctx, state, stdout, stderr))
	cmd.AddCommand(newProviderLinkCommand(ctx, state, "bridge", stdout))
	cmd.AddCommand(newConfigureCommand(state, stdout))
	return cmd
}

func newPlaidProviderCommand(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := newProviderLinkCommand(ctx, state, "plaid", stdout)
	cmd.AddCommand(newPlaidLoginCommand(ctx, state, stdout, stderr, "providers.plaid.login"))
	cmd.AddCommand(newPlaidLogoutCommand(state, stdout, "providers.plaid.logout"))
	return cmd
}

func newPlaidCommand(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plaid",
		Short:   "Plaid-specific setup and Dashboard commands",
		Example: "  money plaid login\n  money plaid logout\n  money plaid sandbox link",
	}
	cmd.AddCommand(newPlaidLoginCommand(ctx, state, stdout, stderr, "plaid.login"))
	cmd.AddCommand(newPlaidLogoutCommand(state, stdout, "plaid.logout"))
	cmd.AddCommand(newPlaidSandboxCommand(ctx, state, stdout))
	return cmd
}

func newPlaidSandboxCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sandbox",
		Short:   "Plaid Sandbox helpers",
		Example: "  money plaid sandbox link\n  money plaid sandbox link --institution-id ins_56 --products transactions,auth",
	}
	cmd.AddCommand(newPlaidSandboxLinkCommand(ctx, state, stdout))
	return cmd
}

func newPlaidSandboxLinkCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var institutionID, products string
	cmd := &cobra.Command{
		Use:     "link",
		Short:   "Create and store a Plaid Sandbox Provider Item",
		Example: "  money plaid sandbox link\n  money plaid sandbox link --institution-id ins_56 --products transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath, Profile: state.profile})
			if err != nil {
				return err
			}
			registry := providers.NewRegistry(cfg)
			provider, ok := registry.Get("plaid")
			if !ok {
				return fmt.Errorf("plaid provider is not registered")
			}
			sandboxCreator, ok := provider.(providers.SandboxPublicTokenCreator)
			if !ok {
				return fmt.Errorf("plaid provider does not support Sandbox public-token creation")
			}
			return runPlaidSandboxLink(ctx, state, sandboxCreator, provider, plaidSandboxLinkOptions{
				Environment:   cfg.Providers["plaid"].Fields["environment"],
				InstitutionID: institutionID,
				Products:      products,
			}, stdout)
		},
	}
	cmd.Flags().StringVar(&institutionID, "institution-id", "ins_56", "Plaid Sandbox institution ID")
	cmd.Flags().StringVar(&products, "products", "transactions", "comma-separated Plaid Sandbox products")
	_ = cmd.RegisterFlagCompletionFunc("products", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"transactions", "auth", "identity", "investments", "liabilities", "transfer"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func newPlaidLoginCommand(_ context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer, commandName string) *cobra.Command {
	var noOpen, force bool
	var team, environment, products, countryCodes, redirectURI string
	cmd := &cobra.Command{
		Use:     "login",
		Short:   "Sign in to Plaid Dashboard and fetch API keys",
		Example: "  money plaid login\n  money plaid login --environment production --no-open",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlaidLoginCLI(cmd.Context(), state, stdout, stderr, plaidLoginCLIOptions{
				CommandName:  commandName,
				NoOpen:       noOpen,
				Team:         team,
				Environment:  environment,
				Products:     products,
				CountryCodes: countryCodes,
				RedirectURI:  redirectURI,
				Force:        force,
			})
		},
	}
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "print the Dashboard OAuth URL without opening a browser")
	cmd.Flags().StringVar(&team, "team", "", "team selector by ID, client ID, name, or 1-based index")
	cmd.Flags().StringVar(&environment, "environment", "sandbox", "Plaid environment to write: sandbox or production")
	_ = cmd.RegisterFlagCompletionFunc("environment", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"sandbox", "production"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&products, "products", "", "comma-separated Plaid Link products to write to config")
	_ = cmd.RegisterFlagCompletionFunc("products", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"transactions", "auth", "identity", "investments", "liabilities", "transfer"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&countryCodes, "country-codes", "", "comma-separated Plaid country codes to write to config")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", "", "Plaid Link redirect URI to write to config")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing PLAID_CLIENT_ID and PLAID_SECRET")
	return cmd
}

func runPlaidLoginLive(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer, opts plaidLoginCLIOptions) error {
	meta, err := config.ResolveMetadata(config.Options{ConfigPath: state.configPath, Profile: state.profile})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cliError{
				command:   opts.CommandName,
				code:      plaidlogin.ErrorBaseConfigMissing,
				message:   "Base money config is missing. Run `money setup` first.",
				category:  contracts.CategoryConfig,
				retryable: false,
				exitCode:  3,
			}
		}
		return err
	}
	if meta.ReadOnly {
		return cliError{
			command:   opts.CommandName,
			code:      plaidlogin.ErrorReadOnlyViolation,
			message:   "Plaid Dashboard login would modify local config while read-only mode is enabled.",
			category:  contracts.CategorySafety,
			retryable: false,
			exitCode:  4,
		}
	}
	if err := validatePlaidLoginOverwrite(state, meta, &opts, stderr); err != nil {
		return err
	}
	stateValue, err := plaidlogin.NewRandomString(16)
	if err != nil {
		return err
	}
	verifier, err := plaidlogin.NewRandomString(32)
	if err != nil {
		return err
	}
	callback := plaidlogin.NewCallbackServer(stateValue, 5*time.Minute)
	localServer, err := callback.StartLocal()
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = localServer.Shutdown(shutdownCtx)
	}()
	authURL, err := plaidlogin.BuildAuthURL(plaidlogin.AuthConfig{
		Port:         localServer.Port,
		State:        stateValue,
		CodeVerifier: verifier,
	})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stderr, "Plaid Dashboard OAuth URL: %s\n", authURL)
	if !opts.NoOpen && !state.json {
		if state.stdin == nil {
			return fmt.Errorf("stdin is required before opening a browser")
		}
		_, _ = fmt.Fprintln(stderr, "Press Enter to open the browser.")
		if _, err := bufio.NewReader(state.stdin).ReadString('\n'); err != nil {
			return err
		}
		if err := openBrowser(authURL); err != nil {
			return err
		}
	}
	callbackResult, err := callback.Wait(ctx)
	if err != nil {
		return plaidLoginError(opts.CommandName, err)
	}
	result, err := plaidlogin.RunLogin(ctx, plaidlogin.LoginOptions{
		ConfigPath:   meta.ConfigPath,
		Profile:      state.profile,
		Environment:  opts.Environment,
		TeamSelector: opts.Team,
		TeamPrompt:   plaidLoginTeamPrompt(state, stderr),
		Products:     opts.Products,
		CountryCodes: opts.CountryCodes,
		RedirectURI:  opts.RedirectURI,
		Force:        opts.Force,
		CallbackCode: callbackResult.Code,
		RedirectPort: localServer.Port,
		CodeVerifier: verifier,
		State:        stateValue,
	})
	if err != nil {
		return plaidLoginError(opts.CommandName, err)
	}
	return writePlaidLoginResult(state, stdout, result, opts.CommandName)
}

func plaidLoginTeamPrompt(state *runtimeState, stderr io.Writer) prompt.Selector {
	if state.json || state.stdin == nil {
		return nil
	}
	if state.prompter != nil {
		return state.prompter
	}
	return prompt.HuhSelector{Input: state.stdin, Output: stderr}
}

func validatePlaidLoginOverwrite(state *runtimeState, meta config.Metadata, opts *plaidLoginCLIOptions, stderr io.Writer) error {
	if opts.Environment == "" {
		opts.Environment = "sandbox"
	}
	if opts.Force {
		return nil
	}
	cfg, err := config.Load(config.Options{ConfigPath: meta.ConfigPath, Profile: state.profile})
	if err != nil {
		if isMissingPlaidCredentialConfigError(err) {
			return nil
		}
		return err
	}
	fields := cfg.Providers["plaid"].Fields
	if fields["client_id"] == "" || fields["secret"] == "" || fields["environment"] == opts.Environment {
		return nil
	}
	message := fmt.Sprintf("Plaid %s credentials already exist; rerun with --force to overwrite them with %s credentials.", fields["environment"], opts.Environment)
	if state.json || state.stdin == nil {
		return cliError{
			command:   opts.CommandName,
			code:      "CONFIRMATION_REQUIRED",
			message:   message,
			category:  contracts.CategorySafety,
			retryable: false,
			exitCode:  10,
		}
	}
	selector := state.prompter
	if selector == nil {
		selector = prompt.HuhSelector{Input: state.stdin, Output: stderr}
	}
	choice, err := selector.Select("Overwrite existing Plaid credentials?", []prompt.Choice{
		{Label: "No, keep existing credentials", Value: "no"},
		{Label: "Yes, overwrite credentials", Value: "yes"},
	})
	if err != nil {
		return err
	}
	if choice != "yes" {
		return cliError{
			command:   opts.CommandName,
			code:      "CONFIRMATION_REQUIRED",
			message:   message,
			category:  contracts.CategorySafety,
			retryable: false,
			exitCode:  10,
		}
	}
	opts.Force = true
	return nil
}

func isMissingPlaidCredentialConfigError(err error) bool {
	var missing config.MissingEnvError
	if !errors.As(err, &missing) {
		return false
	}
	return missing.Path == "providers.plaid.client_id" || missing.Path == "providers.plaid.secret"
}

func writePlaidLoginResult(state *runtimeState, stdout io.Writer, result plaidlogin.LoginResult, commandName string) error {
	if state.json {
		return contracts.WriteJSON(stdout, contracts.NewSuccess(commandName, result))
	}
	_, _ = fmt.Fprintf(stdout, "Plaid Dashboard login complete for team %s.\n", result.TeamID)
	switch result.CredentialAction {
	case "preserved_existing":
		_, _ = fmt.Fprintf(stdout, "Plaid %s credentials already exist; preserved (use --force to overwrite).\n", result.Environment)
	default:
		_, _ = fmt.Fprintf(stdout, "Plaid %s credentials written to %s.\n", result.Environment, result.EnvPath)
	}
	_, _ = fmt.Fprintf(stdout, "Next: %s\n", result.NextCommand)
	return nil
}

func plaidLoginError(command string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return cliError{
			command:   command,
			code:      "PLAID_DASHBOARD_LOGIN_TIMEOUT",
			message:   "Plaid Dashboard login timed out. Run the command again to retry.",
			category:  contracts.CategoryAuth,
			retryable: true,
			exitCode:  3,
		}
	}
	if errors.Is(err, context.Canceled) {
		return cliError{
			command:   command,
			code:      "LOGIN_CANCELED",
			message:   "Plaid Dashboard login was canceled.",
			category:  contracts.CategorySafety,
			retryable: true,
			exitCode:  10,
		}
	}
	var dashErr plaidlogin.Error
	if !errors.As(err, &dashErr) {
		return err
	}
	category := contracts.CategoryAPI
	exitCode := 6
	retryable := false
	switch dashErr.Code {
	case "CONFIG_WRITE_FAILED":
		category = contracts.CategoryConfig
		exitCode = 1
	case plaidlogin.ErrorBaseConfigMissing:
		category = contracts.CategoryConfig
		exitCode = 3
	case plaidlogin.ErrorNotLoggedIn, plaidlogin.ErrorPlaidDashboardLoginRejected, plaidlogin.ErrorDashboardTokenRefreshFailed:
		category = contracts.CategoryAuth
		exitCode = 3
	case plaidlogin.ErrorTeamSelectionRequired, plaidlogin.ErrorPlaidEnvironmentNotProvisioned, plaidlogin.ErrorInvalidEnum:
		category = contracts.CategoryValidation
		exitCode = 7
	case plaidlogin.ErrorPlaidCredentialsOverwriteRequired:
		category = contracts.CategorySafety
		exitCode = 10
	case plaidlogin.ErrorAPIKeysFetchRequired:
		category = contracts.CategoryAuth
		exitCode = 3
		retryable = true
	case plaidlogin.ErrorReadOnlyViolation:
		category = contracts.CategorySafety
		exitCode = 4
	case plaidlogin.ErrorDashboardContractChanged:
		category = contracts.CategoryAPI
		exitCode = 6
	}
	message := dashErr.Error()
	if dashErr.Code == plaidlogin.ErrorDashboardContractChanged || dashErr.Code == plaidlogin.ErrorPlaidDashboardLoginRejected {
		message += " Run `money providers configure plaid` manually. Get credentials: " + config.PlaidSpec.HelpURL
	}
	return cliError{
		command:   command,
		code:      dashErr.Code,
		message:   message,
		category:  category,
		retryable: retryable,
		exitCode:  exitCode,
	}
}

func newPlaidLogoutCommand(state *runtimeState, stdout io.Writer, commandName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "Remove stored Plaid Dashboard auth without deleting API keys",
		Example: "  money plaid logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			meta, err := config.ResolveMetadata(config.Options{ConfigPath: state.configPath, Profile: state.profile})
			if err != nil {
				return err
			}
			if meta.ReadOnly {
				return cliError{
					command:   commandName,
					code:      plaidlogin.ErrorReadOnlyViolation,
					message:   "Plaid Dashboard logout would modify local auth files while read-only mode is enabled.",
					category:  contracts.CategorySafety,
					retryable: false,
					exitCode:  4,
				}
			}
			authPath := plaidlogin.DashboardAuthPath(meta.ConfigPath)
			removed, err := plaidlogin.DeleteAuthFile(authPath)
			if err != nil {
				return err
			}
			data := map[string]any{
				"provider":               "plaid",
				"dashboard_auth_removed": removed,
				"dashboard_auth_path":    authPath,
				"api_keys_preserved":     true,
				"env_path":               meta.EnvPath,
			}
			if state.json {
				return contracts.WriteJSON(stdout, contracts.NewSuccess(commandName, data))
			}
			if removed {
				_, _ = fmt.Fprintf(stdout, "Plaid Dashboard auth removed from %s.\n", authPath)
			} else {
				_, _ = fmt.Fprintf(stdout, "Plaid Dashboard auth was not present at %s.\n", authPath)
			}
			_, _ = fmt.Fprintf(stdout, "API keys remain in %s. To remove them, edit the file or run money providers configure plaid with new values.\n", meta.EnvPath)
			return nil
		},
	}
	return cmd
}

type providerAvailabilityRow struct {
	Provider string
	Status   string
	Code     string
	Guidance string
}

func supportedProviderAvailability(providerName string, diagnostics []providers.ConfigDiagnostic) []providerAvailabilityRow {
	if len(diagnostics) == 0 {
		return []providerAvailabilityRow{{Provider: providerName, Status: "available"}}
	}
	return []providerAvailabilityRow{{
		Provider: providerName,
		Status:   "unavailable",
		Code:     diagnostics[0].Code,
		Guidance: "Configure providers." + providerName + " credentials with env references.",
	}}
}

func writeProviderAvailability(stdout io.Writer, rows []providerAvailabilityRow) {
	_, _ = fmt.Fprintln(stdout, "provider\tstatus\tcode\tguidance")
	for _, row := range rows {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", row.Provider, row.Status, row.Code, row.Guidance)
	}
}
