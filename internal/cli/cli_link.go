package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/linking"
	"github.com/thedavidweng/money/internal/providers"
)

func newLinkCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var providerName, institutionID string
	var noOpen bool
	var additionalConsentedProducts, requiredIfSupportedProducts, optionalProducts string
	cmd := &cobra.Command{
		Use:     "link <institution-query>",
		Short:   "Link an institution through a Provider",
		Example: "  money link \"Chase\"\n  money link \"Wells Fargo\" --provider plaid\n  money link \"Amex\" --no-open",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if providerName != "plaid" {
				return cliError{
					command:   "link",
					code:      "FEATURE_NOT_IMPLEMENTED",
					message:   providerName + " institution-first Link flow is not implemented yet.",
					category:  contracts.CategoryAPI,
					retryable: false,
					exitCode:  6,
				}
			}
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath, Profile: state.profile})
			if err != nil {
				return err
			}
			registry := providers.NewRegistry(cfg)
			provider, ok := registry.Get(providerName)
			if !ok {
				return fmt.Errorf("%s provider is not registered", providerName)
			}
			diagnostics := provider.ValidateConfig(ctx)
			if len(diagnostics) > 0 {
				availability := supportedProviderAvailability(providerName, diagnostics)
				if !state.json {
					writeProviderAvailability(stdout, availability)
				}
				msg := providerName + " is supported for institution linking but unavailable locally: " + diagnostics[0].Message + " " + availability[0].Guidance
				if spec, ok := config.ProviderSpecByName(providerName); ok && spec.HelpURL != "" {
					msg += " Get credentials: " + spec.HelpURL
				}
				return cliError{
					command:   "link",
					code:      diagnostics[0].Code,
					message:   msg,
					category:  contracts.CategoryAuth,
					retryable: false,
					exitCode:  3,
				}
			}
			if state.json {
				return cliError{
					command:   "link",
					code:      "INTERACTIVE_LINK_REQUIRES_HUMAN_MODE",
					message:   providerName + " Link requires a local browser callback; omit --json for the live link flow.",
					category:  contracts.CategoryValidation,
					retryable: false,
					exitCode:  2,
				}
			}
			institutions, err := provider.SearchInstitutions(ctx, args[0])
			if err != nil {
				return err
			}
			institution, err := selectLinkInstitution(institutions, institutionID)
			if err != nil {
				if len(institutions) > 1 && institutionID == "" {
					for _, candidate := range institutions {
						_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", candidate.Provider, candidate.ProviderInstitutionID, candidate.Name)
					}
				}
				return cliError{
					command:   "link",
					code:      "INSTITUTION_SELECTION_REQUIRED",
					message:   err.Error(),
					category:  contracts.CategoryValidation,
					retryable: false,
					exitCode:  2,
				}
			}
			return runPlaidLinkFlow(ctx, state, provider, plaidLinkFlowOptions{
				CommandName:                 "link",
				Institution:                 institution,
				RedirectURI:                 cfg.Providers["plaid"].Fields["redirect_uri"],
				NoOpen:                      noOpen,
				AdditionalConsentedProducts: additionalConsentedProducts,
				RequiredIfSupportedProducts: requiredIfSupportedProducts,
				OptionalProducts:            optionalProducts,
			}, stdout)
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "plaid", "provider to use")
	_ = cmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"plaid", "bridge"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&institutionID, "institution-id", "", "provider institution id from search results")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "print the Link URL without opening a browser")
	cmd.Flags().StringVar(&additionalConsentedProducts, "additional-consented-products", "", "comma-separated Plaid products to collect consent for without initializing")
	cmd.Flags().StringVar(&requiredIfSupportedProducts, "required-if-supported-products", "", "comma-separated Plaid products required when the institution supports them")
	cmd.Flags().StringVar(&optionalProducts, "optional-products", "", "comma-separated optional Plaid products")
	return cmd
}

func newProviderLinkCommand(ctx context.Context, state *runtimeState, providerName string, stdout io.Writer) *cobra.Command {
	providerCmd := &cobra.Command{Use: providerName}
	var noOpen bool
	var additionalConsentedProducts, requiredIfSupportedProducts, optionalProducts string
	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Link a " + providerName + " Provider Item",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath, Profile: state.profile})
			if err != nil {
				return err
			}
			registry := providers.NewRegistry(cfg)
			provider, ok := registry.Get(providerName)
			if !ok {
				return fmt.Errorf("%s provider is not registered", providerName)
			}
			diagnostics := provider.ValidateConfig(ctx)
			if len(diagnostics) > 0 {
				msg := diagnostics[0].Message
				if spec, ok := config.ProviderSpecByName(providerName); ok && spec.HelpURL != "" {
					msg += " Get credentials: " + spec.HelpURL
				}
				return cliError{
					command:   "providers." + providerName + ".link",
					code:      diagnostics[0].Code,
					message:   msg,
					category:  contracts.CategoryAuth,
					retryable: false,
					exitCode:  3,
				}
			}
			if state.json {
				return cliError{
					command:   "providers." + providerName + ".link",
					code:      "INTERACTIVE_LINK_REQUIRES_HUMAN_MODE",
					message:   providerName + " Link requires a local browser callback; omit --json for the live link flow.",
					category:  contracts.CategoryValidation,
					retryable: false,
					exitCode:  2,
				}
			}
			switch providerName {
			case "plaid":
				return runPlaidLinkFlow(ctx, state, provider, plaidLinkFlowOptions{
					CommandName:                 "providers.plaid.link",
					RedirectURI:                 cfg.Providers["plaid"].Fields["redirect_uri"],
					NoOpen:                      noOpen,
					AdditionalConsentedProducts: additionalConsentedProducts,
					RequiredIfSupportedProducts: requiredIfSupportedProducts,
					OptionalProducts:            optionalProducts,
				}, stdout)
			case "bridge":
				return runBridgeLinkFlow(ctx, state, provider, cfg.Providers["bridge"].Fields["callback_url"], noOpen, stdout)
			default:
				return cliError{
					command:   "providers." + providerName + ".link",
					code:      "FEATURE_NOT_IMPLEMENTED",
					message:   providerName + " Link flow is not implemented yet.",
					category:  contracts.CategoryAPI,
					retryable: false,
					exitCode:  6,
				}
			}
		},
	}
	linkCmd.Flags().BoolVar(&noOpen, "no-open", false, "print the Link URL without opening a browser")
	if providerName == "plaid" {
		linkCmd.Flags().StringVar(&additionalConsentedProducts, "additional-consented-products", "", "comma-separated Plaid products to collect consent for without initializing")
		linkCmd.Flags().StringVar(&requiredIfSupportedProducts, "required-if-supported-products", "", "comma-separated Plaid products required when the institution supports them")
		linkCmd.Flags().StringVar(&optionalProducts, "optional-products", "", "comma-separated optional Plaid products")
	}
	providerCmd.AddCommand(linkCmd)
	return providerCmd
}

type linkSessionServer interface {
	LinkURL() string
	Wait(ctx context.Context) (providers.LinkCallback, error)
	Shutdown(ctx context.Context) error
}

type localLinkSessionServer struct {
	helper *linking.PlaidLinkHelper
	server *linking.LocalPlaidLinkServer
}

func (s localLinkSessionServer) LinkURL() string {
	return s.server.URL
}

func (s localLinkSessionServer) Wait(ctx context.Context) (providers.LinkCallback, error) {
	return s.helper.Wait(ctx)
}

func (s localLinkSessionServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

var startPlaidLinkSessionServer = func(linkToken string, state string, timeout time.Duration) (linkSessionServer, error) {
	helper := linking.NewPlaidLinkHelper(linking.PlaidLinkHelperConfig{
		LinkToken: linkToken,
		State:     state,
		Timeout:   timeout,
	})
	server, err := helper.StartLocalServer()
	if err != nil {
		return nil, err
	}
	return localLinkSessionServer{helper: helper, server: server}, nil
}

var openBrowser = func(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Run()
	case "linux":
		return exec.Command("xdg-open", url).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	default:
		return fmt.Errorf("browser opening is not implemented on %s", runtime.GOOS)
	}
}

type plaidLinkFlowOptions struct {
	CommandName                 string
	Institution                 providers.Institution
	RedirectURI                 string
	NoOpen                      bool
	AdditionalConsentedProducts string
	RequiredIfSupportedProducts string
	OptionalProducts            string
}

func commaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func runPlaidLinkFlow(ctx context.Context, state *runtimeState, provider providers.Provider, opts plaidLinkFlowOptions, stdout io.Writer) error {
	activeStore, err := requireStore(state)
	if err != nil {
		return err
	}
	linkState, err := linking.NewLinkState()
	if err != nil {
		return err
	}
	session, err := provider.CreateLinkSession(ctx, providers.LinkRequest{
		Institution:                 opts.Institution,
		RedirectURI:                 opts.RedirectURI,
		State:                       linkState,
		AdditionalConsentedProducts: commaList(opts.AdditionalConsentedProducts),
		RequiredIfSupportedProducts: commaList(opts.RequiredIfSupportedProducts),
		OptionalProducts:            commaList(opts.OptionalProducts),
	})
	if err != nil {
		return err
	}
	if session.LinkToken == "" {
		return fmt.Errorf("plaid Link session did not return a link token")
	}
	server, err := startPlaidLinkSessionServer(session.LinkToken, session.State, 5*time.Minute)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if !state.json && state.stderr != nil {
		_, _ = fmt.Fprintf(state.stderr, "Plaid Link URL: %s\n", server.LinkURL())
	}
	if !opts.NoOpen && !state.json {
		if state.stdin == nil {
			return fmt.Errorf("stdin is required before opening a browser")
		}
		if state.stderr != nil {
			_, _ = fmt.Fprintln(state.stderr, "Press Enter to open the browser.")
		}
		if _, err := bufio.NewReader(state.stdin).ReadString('\n'); err != nil {
			return err
		}
		if err := openBrowser(server.LinkURL()); err != nil {
			return err
		}
	}

	callback, err := server.Wait(ctx)
	if err != nil {
		return err
	}
	result, err := linking.CompleteProviderLink(ctx, activeStore, provider, session, callback)
	if err != nil {
		var canceled linking.LinkCanceledError
		if errors.As(err, &canceled) {
			return cliError{
				command:   opts.CommandName,
				code:      "LINK_CANCELED",
				message:   canceled.Error(),
				category:  contracts.CategorySafety,
				retryable: true,
				exitCode:  10,
			}
		}
		var flowErr linking.LinkFlowError
		if errors.As(err, &flowErr) {
			return cliError{
				command:   opts.CommandName,
				code:      "LINK_ERROR",
				message:   flowErr.Error(),
				category:  contracts.CategoryAPI,
				retryable: false,
				exitCode:  6,
			}
		}
		return err
	}
	if state.json {
		env := contracts.NewSuccess(opts.CommandName, map[string]any{
			"provider":         result.Provider,
			"provider_item_id": result.ProviderItemID,
			"institution_id":   result.InstitutionID,
		})
		return contracts.WriteJSON(stdout, env)
	}
	_, _ = fmt.Fprintf(stdout, "Linked %s Provider Item %s.\n", result.Provider, result.ProviderItemID)
	_, _ = fmt.Fprintln(stdout, "No sync was run. Run `money sync` after linking.")
	return nil
}

type plaidSandboxLinkOptions struct {
	Environment   string
	InstitutionID string
	Products      string
}

func runPlaidSandboxLink(ctx context.Context, state *runtimeState, sandboxCreator providers.SandboxPublicTokenCreator, provider providers.Provider, opts plaidSandboxLinkOptions, stdout io.Writer) error {
	environment := opts.Environment
	if environment == "" {
		environment = "sandbox"
	}
	if environment != "sandbox" {
		return cliError{
			command:   "plaid.sandbox.link",
			code:      "INVALID_ENVIRONMENT",
			message:   "money plaid sandbox link requires providers.plaid.environment to be sandbox.",
			category:  contracts.CategoryValidation,
			retryable: false,
			exitCode:  2,
		}
	}
	products := commaList(opts.Products)
	for _, product := range products {
		if product == "balance" {
			return cliError{
				command:   "plaid.sandbox.link",
				code:      "INVALID_PRODUCT",
				message:   "Plaid Sandbox product balance is not supported; choose explicit initial products such as transactions.",
				category:  contracts.CategoryValidation,
				retryable: false,
				exitCode:  2,
			}
		}
	}
	activeStore, err := requireStore(state)
	if err != nil {
		return err
	}
	linkState, err := linking.NewLinkState()
	if err != nil {
		return err
	}
	publicToken, err := sandboxCreator.CreateSandboxPublicToken(ctx, providers.SandboxPublicTokenRequest{
		InstitutionID: opts.InstitutionID,
		Products:      products,
	})
	if err != nil {
		return err
	}
	session := providers.LinkSession{
		Provider: "plaid",
		State:    linkState,
		Products: products,
	}
	result, err := linking.CompleteProviderLink(ctx, activeStore, provider, session, providers.LinkCallback{
		PublicToken: publicToken,
		State:       linkState,
		Status:      "success",
		Metadata: providers.LinkMetadata{
			Institution: providers.LinkInstitutionMetadata{ID: opts.InstitutionID, Name: "Plaid Sandbox"},
		},
	})
	if err != nil {
		return err
	}
	if state.json {
		env := contracts.NewSuccess("plaid.sandbox.link", map[string]any{
			"provider":         result.Provider,
			"provider_item_id": result.ProviderItemID,
			"institution_id":   result.InstitutionID,
		})
		return contracts.WriteJSON(stdout, env)
	}
	_, _ = fmt.Fprintf(stdout, "Linked %s Sandbox Provider Item %s.\n", result.Provider, result.ProviderItemID)
	_, _ = fmt.Fprintln(stdout, "No sync was run. Run `money sync` after linking.")
	return nil
}

func runBridgeLinkFlow(ctx context.Context, state *runtimeState, provider providers.Provider, callbackURL string, noOpen bool, stdout io.Writer) error {
	activeStore, err := requireStore(state)
	if err != nil {
		return err
	}
	linkState, err := linking.NewLinkState()
	if err != nil {
		return err
	}
	session, err := provider.CreateLinkSession(ctx, providers.LinkRequest{
		RedirectURI: callbackURL,
		State:       linkState,
	})
	if err != nil {
		return err
	}
	if session.URL == "" {
		return fmt.Errorf("bridge connect session did not return a URL")
	}
	_, _ = fmt.Fprintf(stdout, "Bridge Connect URL: %s\n", session.URL)
	if !noOpen {
		if state.stdin == nil {
			return fmt.Errorf("stdin is required before opening a browser")
		}
		_, _ = fmt.Fprintln(stdout, "Press Enter to open the browser.")
		if _, err := bufio.NewReader(state.stdin).ReadString('\n'); err != nil {
			return err
		}
		if err := openBrowser(session.URL); err != nil {
			return err
		}
	}
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	result, err := linking.CompleteProviderLink(pollCtx, activeStore, provider, session, providers.LinkCallback{State: session.State})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Linked %s Provider Item %s.\n", result.Provider, result.ProviderItemID)
	_, _ = fmt.Fprintln(stdout, "No sync was run. Run `money sync` after linking.")
	return nil
}

func selectLinkInstitution(institutions []providers.Institution, institutionID string) (providers.Institution, error) {
	if institutionID != "" {
		for _, institution := range institutions {
			if institution.ProviderInstitutionID == institutionID || institution.ID == institutionID {
				return institution, nil
			}
		}
		return providers.Institution{}, fmt.Errorf("institution %q was not returned by the provider search", institutionID)
	}
	if len(institutions) == 0 {
		return providers.Institution{}, fmt.Errorf("no institutions matched the query")
	}
	if len(institutions) > 1 {
		return providers.Institution{}, fmt.Errorf("multiple institutions matched; rerun with --institution-id")
	}
	return institutions[0], nil
}

