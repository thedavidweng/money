package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/core"
	"github.com/thedavidweng/money/internal/linking"
	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
	"github.com/thedavidweng/money/internal/syncer"
)

type runtimeState struct {
	store      store.Store
	demo       bool
	json       bool
	configPath string
	stdin      io.Reader
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	state := &runtimeState{stdin: stdin}
	root := newRootCommand(ctx, state, stdout, stderr)
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stderr)
	root.SetErr(stderr)

	if err := root.ExecuteContext(ctx); err != nil {
		var exitErr cliExit
		if errors.As(err, &exitErr) {
			return exitErr.exitCode
		}
		if cliErr, ok := err.(cliError); ok {
			if state.json {
				env := contracts.NewError(cliErr.command, cliErr.code, cliErr.message, cliErr.category, cliErr.retryable)
				if writeErr := contracts.WriteJSON(stdout, env); writeErr != nil {
					fmt.Fprintln(stderr, writeErr)
				}
			} else {
				fmt.Fprintln(stderr, cliErr.message)
			}
			return cliErr.exitCode
		}
		if state.json {
			env := contracts.NewError(commandName(root), "COMMAND_FAILED", err.Error(), contracts.CategoryInternal, false)
			if writeErr := contracts.WriteJSON(stdout, env); writeErr != nil {
				fmt.Fprintln(stderr, writeErr)
			}
		} else {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	return 0
}

func newRootCommand(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "money",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&state.json, "json", false, "write a JSON envelope to stdout")
	root.PersistentFlags().StringVar(&state.configPath, "config", "", "config file path")

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			env := contracts.NewSuccess("version", map[string]string{"version": "0.0.0"})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})

	root.AddCommand(newDemoCommand(ctx, state, stdout, stderr))
	root.AddCommand(newAccountsCommand(ctx, state, stdout))
	root.AddCommand(newTransactionsCommand(ctx, state, stdout))
	root.AddCommand(newCategoriesCommand(ctx, state, stdout))
	root.AddCommand(newTagsCommand(ctx, state, stdout))
	root.AddCommand(newRecurringCommand(ctx, state, stdout))
	root.AddCommand(newProvidersCommand(ctx, state, stdout))
	root.AddCommand(newLinkCommand(ctx, state, stdout))
	root.AddCommand(newSyncCommand(ctx, state, stdout))

	txAlias := newTransactionsCommand(ctx, state, stdout)
	txAlias.Use = "tx"
	txAlias.Aliases = nil
	root.AddCommand(txAlias)

	return root
}

type cliError struct {
	command   string
	code      string
	message   string
	category  contracts.Category
	retryable bool
	exitCode  int
}

func (e cliError) Error() string {
	return e.message
}

type cliExit struct {
	exitCode int
}

func (e cliExit) Error() string {
	return fmt.Sprintf("exit %d", e.exitCode)
}

func newDemoCommand(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo <command...>",
		Short: "Run a command against bundled non-persistent sample data",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			demoStore, err := store.OpenDemo(ctx)
			if err != nil {
				return err
			}
			state.store = demoStore
			state.demo = true
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if closer, ok := state.store.(interface{ Close() error }); ok {
				_ = closer.Close()
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newAccountsCommand(ctx, state, stdout))
	cmd.AddCommand(newTransactionsCommand(ctx, state, stdout))
	cmd.AddCommand(newCategoriesCommand(ctx, state, stdout))
	cmd.AddCommand(newTagsCommand(ctx, state, stdout))
	cmd.AddCommand(newRecurringCommand(ctx, state, stdout))
	txAlias := newTransactionsCommand(ctx, state, stdout)
	txAlias.Use = "tx"
	txAlias.Aliases = nil
	cmd.AddCommand(txAlias)
	return cmd
}

func newAccountsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "accounts"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			accounts, err := activeStore.ListAccounts(ctx)
			if err != nil {
				return err
			}
			if !state.json {
				for _, account := range accounts {
					fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", account.DisplayName, account.Type, account.CurrentBalance, account.Currency)
				}
				return nil
			}
			env := contracts.NewSuccess("accounts.list", map[string]any{"accounts": accounts})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	cmd.AddCommand(newCreateManualCommand(ctx, state, stdout))
	return cmd
}

func newCreateManualCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var name, accountType, subtype, balance, currency, alias string
	var dryRun, confirm bool
	cmd := &cobra.Command{
		Use:   "create-manual",
		Short: "Create a local manual account",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.json && !dryRun && !confirm {
				return fmt.Errorf("JSON manual account writes require --dry-run or --confirm")
			}
			if strings.TrimSpace(name) == "" || strings.TrimSpace(accountType) == "" || strings.TrimSpace(balance) == "" || strings.TrimSpace(currency) == "" {
				return fmt.Errorf("manual account requires --name, --type, --balance, and --currency")
			}
			unsignedMinorUnits, err := core.ParseUnsignedDecimalMinorUnits(balance)
			if err != nil {
				return err
			}
			signedMinorUnits, position, err := core.SignedManualBalance(accountType, unsignedMinorUnits)
			if err != nil {
				return err
			}
			plan := manualAccountPlan{
				WillWrite:         confirm,
				AccountName:       name,
				AccountType:       accountType,
				SignedBalance:     core.FormatMinorUnits(signedMinorUnits, currency),
				Currency:          currency,
				FinancialPosition: position,
			}
			if dryRun {
				plan.WillWrite = false
				return writeManualPlan(stdout, state, plan)
			}
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			account, err := activeStore.CreateManualAccount(ctx, core.Account{
				Name:                     name,
				Alias:                    alias,
				Type:                     accountType,
				Subtype:                  subtype,
				CurrentBalanceMinorUnits: signedMinorUnits,
				Currency:                 currency,
				Source:                   core.Source{Kind: "manual"},
			})
			if err != nil {
				return err
			}
			if state.json {
				env := contracts.NewSuccess("accounts.create_manual", map[string]any{"account": account, "plan": plan})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Created %s with balance %s %s\n", account.DisplayName, account.CurrentBalance, account.Currency)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "account name")
	cmd.Flags().StringVar(&accountType, "type", "", "account type")
	cmd.Flags().StringVar(&subtype, "subtype", "", "account subtype")
	cmd.Flags().StringVar(&balance, "balance", "", "unsigned balance")
	cmd.Flags().StringVar(&currency, "currency", "USD", "currency")
	cmd.Flags().StringVar(&alias, "alias", "", "local account alias")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show write plan without saving")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "save the manual account")
	return cmd
}

func newTransactionsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transactions",
		Aliases: []string{"transaction"},
	}
	cmd.AddCommand(newTransactionsListCommand(ctx, state, stdout))
	var searchLimit int
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search transactions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			transactions, err := activeStore.SearchTransactions(ctx, args[0], searchLimit)
			if err != nil {
				return err
			}
			return writeTransactionsPage(stdout, state, "transactions.search", transactions, searchLimit, 0)
		},
	}
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "maximum transactions to return")
	cmd.AddCommand(searchCmd)
	return cmd
}

func newTransactionsListCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var removedMode string
	var accountID, categoryID, merchant, tagID, dateFrom, dateTo string
	var needsReview, pending, recurring string
	var limit, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			needsReviewFilter, err := optionalBoolFlag("needs-review", needsReview)
			if err != nil {
				return err
			}
			pendingFilter, err := optionalBoolFlag("pending", pending)
			if err != nil {
				return err
			}
			recurringFilter, err := optionalBoolFlag("recurring", recurring)
			if err != nil {
				return err
			}
			transactions, err := activeStore.ListTransactions(ctx, store.TransactionListQuery{
				AccountID:   accountID,
				CategoryID:  categoryID,
				Merchant:    merchant,
				TagID:       tagID,
				DateFrom:    dateFrom,
				DateTo:      dateTo,
				NeedsReview: needsReviewFilter,
				Pending:     pendingFilter,
				Recurring:   recurringFilter,
				RemovedMode: store.RemovedMode(removedMode),
				Limit:       limit,
				Offset:      offset,
			})
			if err != nil {
				return err
			}
			return writeTransactionsPage(stdout, state, "transactions.list", transactions, limit, offset)
		},
	}
	cmd.Flags().StringVar(&removedMode, "removed", string(store.RemovedExclude), "removed transaction mode: exclude, include, or only")
	cmd.Flags().StringVar(&accountID, "account", "", "filter by account id")
	cmd.Flags().StringVar(&categoryID, "category", "", "filter by category id")
	cmd.Flags().StringVar(&merchant, "merchant", "", "filter by merchant or transaction name")
	cmd.Flags().StringVar(&tagID, "tag", "", "filter by tag id")
	cmd.Flags().StringVar(&dateFrom, "date-from", "", "filter by inclusive start date")
	cmd.Flags().StringVar(&dateTo, "date-to", "", "filter by inclusive end date")
	cmd.Flags().StringVar(&needsReview, "needs-review", "", "filter by review state: true or false")
	cmd.Flags().StringVar(&pending, "pending", "", "filter by pending state: true or false")
	cmd.Flags().StringVar(&recurring, "recurring", "", "filter by recurring state: true or false")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum transactions to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "transactions to skip")
	return cmd
}

func newCategoriesCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "categories"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List categories",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			categories, err := activeStore.ListCategories(ctx)
			if err != nil {
				return err
			}
			env := contracts.NewSuccess("categories.list", map[string]any{"categories": categories})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	return cmd
}

func newTagsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "tags"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			tags, err := activeStore.ListTags(ctx)
			if err != nil {
				return err
			}
			env := contracts.NewSuccess("tags.list", map[string]any{"tags": tags})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	return cmd
}

func newRecurringCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "recurring"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List recurring transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			recurringItems, err := activeStore.ListRecurring(ctx)
			if err != nil {
				return err
			}
			env := contracts.NewSuccess("recurring.list", map[string]any{"recurring": recurringItems})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	return cmd
}

func newSyncCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var providerName, providerItemID string
	var verbose bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync linked Provider Items",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			syncStore, ok := activeStore.(syncer.Store)
			if !ok {
				return fmt.Errorf("active store cannot sync provider items")
			}
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath})
			if err != nil {
				return err
			}
			result, err := syncer.Sync(ctx, syncStore, providers.NewRegistry(cfg), syncer.Options{
				Provider:       providerName,
				ProviderItemID: providerItemID,
			})
			if state.json {
				return writeSyncJSON(stdout, result, err)
			}
			writeSyncHuman(stdout, result, verbose)
			return err
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "sync only one provider")
	cmd.Flags().StringVar(&providerItemID, "provider-item", "", "sync only one provider item")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show per-provider-item sync details")
	return cmd
}

func newProvidersCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "providers"}
	cmd.AddCommand(newProviderLinkCommand(ctx, state, "plaid", stdout))
	cmd.AddCommand(newProviderLinkCommand(ctx, state, "bridge", stdout))
	return cmd
}

func newLinkCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var providerName, institutionID string
	var noOpen bool
	cmd := &cobra.Command{
		Use:   "link <institution-query>",
		Short: "Link an institution through a Provider",
		Args:  cobra.ExactArgs(1),
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
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath})
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
				return cliError{
					command:   "link",
					code:      diagnostics[0].Code,
					message:   providerName + " is supported for institution linking but unavailable locally: " + diagnostics[0].Message + " " + availability[0].Guidance,
					category:  contracts.CategoryAuth,
					retryable: false,
					exitCode:  3,
				}
			}
			if state.json {
				return cliError{
					command:   "link",
					code:      "INTERACTIVE_LINK_REQUIRES_HUMAN_MODE",
					message:   "Plaid Link requires a local browser callback; omit --json for the live link flow.",
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
						fmt.Fprintf(stdout, "%s\t%s\t%s\n", candidate.Provider, candidate.ProviderInstitutionID, candidate.Name)
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
			return runPlaidLinkFlow(ctx, state, provider, institution, cfg.Providers["plaid"].Fields["redirect_uri"], noOpen, stdout)
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "plaid", "provider to use")
	cmd.Flags().StringVar(&institutionID, "institution-id", "", "provider institution id from search results")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "print the Link URL without opening a browser")
	return cmd
}

func newProviderLinkCommand(ctx context.Context, state *runtimeState, providerName string, stdout io.Writer) *cobra.Command {
	providerCmd := &cobra.Command{Use: providerName}
	var noOpen bool
	providerCmd.AddCommand(&cobra.Command{
		Use:   "link",
		Short: "Link a " + providerName + " Provider Item",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath})
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
				return cliError{
					command:   "providers." + providerName + ".link",
					code:      diagnostics[0].Code,
					message:   diagnostics[0].Message,
					category:  contracts.CategoryAuth,
					retryable: false,
					exitCode:  3,
				}
			}
			if state.json {
				return cliError{
					command:   "providers." + providerName + ".link",
					code:      "INTERACTIVE_LINK_REQUIRES_HUMAN_MODE",
					message:   providerName + " Link requires a browser flow; omit --json for the live link flow.",
					category:  contracts.CategoryValidation,
					retryable: false,
					exitCode:  2,
				}
			}
			switch providerName {
			case "plaid":
				return runPlaidLinkFlow(ctx, state, provider, providers.Institution{}, cfg.Providers["plaid"].Fields["redirect_uri"], noOpen, stdout)
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
	})
	providerCmd.PersistentFlags().BoolVar(&noOpen, "no-open", false, "print the Link URL without opening a browser")
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
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("browser opening is only implemented on darwin")
	}
	return exec.Command("open", url).Run()
}

func runPlaidLinkFlow(ctx context.Context, state *runtimeState, provider providers.Provider, institution providers.Institution, redirectURI string, noOpen bool, stdout io.Writer) error {
	activeStore, err := requireStore(state)
	if err != nil {
		return err
	}
	linkStore, ok := activeStore.(linking.Store)
	if !ok {
		return fmt.Errorf("active store cannot persist linked provider items")
	}
	linkState, err := linking.NewLinkState()
	if err != nil {
		return err
	}
	session, err := provider.CreateLinkSession(ctx, providers.LinkRequest{
		Institution: institution,
		RedirectURI: redirectURI,
		State:       linkState,
	})
	if err != nil {
		return err
	}
	if session.LinkToken == "" {
		return fmt.Errorf("Plaid Link session did not return a link token")
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

	fmt.Fprintf(stdout, "Plaid Link URL: %s\n", server.LinkURL())
	if !noOpen {
		if state.stdin == nil {
			return fmt.Errorf("stdin is required before opening a browser")
		}
		fmt.Fprintln(stdout, "Press Enter to open the browser.")
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
	result, err := linking.CompleteProviderLink(ctx, linkStore, provider, session, callback)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Linked %s Provider Item %s.\n", result.Provider, result.ProviderItemID)
	fmt.Fprintln(stdout, "No sync was run. Run `money sync` after linking.")
	return nil
}

func runBridgeLinkFlow(ctx context.Context, state *runtimeState, provider providers.Provider, callbackURL string, noOpen bool, stdout io.Writer) error {
	activeStore, err := requireStore(state)
	if err != nil {
		return err
	}
	linkStore, ok := activeStore.(linking.Store)
	if !ok {
		return fmt.Errorf("active store cannot persist linked provider items")
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
		return fmt.Errorf("Bridge connect session did not return a URL")
	}
	fmt.Fprintf(stdout, "Bridge Connect URL: %s\n", session.URL)
	if !noOpen {
		if state.stdin == nil {
			return fmt.Errorf("stdin is required before opening a browser")
		}
		fmt.Fprintln(stdout, "Press Enter to open the browser.")
		if _, err := bufio.NewReader(state.stdin).ReadString('\n'); err != nil {
			return err
		}
		if err := openBrowser(session.URL); err != nil {
			return err
		}
	}
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	result, err := linking.CompleteProviderLink(pollCtx, linkStore, provider, session, providers.LinkCallback{State: session.State})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Linked %s Provider Item %s.\n", result.Provider, result.ProviderItemID)
	fmt.Fprintln(stdout, "No sync was run. Run `money sync` after linking.")
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
	fmt.Fprintln(stdout, "provider\tstatus\tcode\tguidance")
	for _, row := range rows {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", row.Provider, row.Status, row.Code, row.Guidance)
	}
}

type manualAccountPlan struct {
	WillWrite         bool   `json:"will_write"`
	AccountName       string `json:"account_name"`
	AccountType       string `json:"account_type"`
	SignedBalance     string `json:"signed_balance"`
	Currency          string `json:"currency"`
	FinancialPosition string `json:"financial_position"`
}

func writeManualPlan(stdout io.Writer, state *runtimeState, plan manualAccountPlan) error {
	if state.json {
		env := contracts.NewSuccess("accounts.create_manual", map[string]any{"plan": plan})
		env.Meta.Demo = state.demo
		return contracts.WriteJSON(stdout, env)
	}
	fmt.Fprintf(stdout, "Would create %s with balance %s %s\n", plan.AccountName, plan.SignedBalance, plan.Currency)
	return nil
}

func writeTransactions(stdout io.Writer, state *runtimeState, command string, transactions []core.Transaction) error {
	if !state.json {
		for _, tx := range transactions {
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", tx.Date, tx.MerchantName, tx.Amount, tx.CategorySource)
		}
		return nil
	}
	env := contracts.NewSuccess(command, map[string]any{"transactions": transactions})
	env.Meta.Demo = state.demo
	return contracts.WriteJSON(stdout, env)
}

func writeTransactionsPage(stdout io.Writer, state *runtimeState, command string, transactions []core.Transaction, limit int, offset int) error {
	if !state.json {
		for _, tx := range transactions {
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", tx.Date, tx.MerchantName, tx.Amount, tx.CategorySource)
		}
		return nil
	}
	env := contracts.NewSuccess(command, map[string]any{"transactions": transactions})
	env.Meta.Demo = state.demo
	env.Meta.Pagination = &contracts.Pagination{
		Limit:   limit,
		Offset:  offset,
		HasMore: len(transactions) == limit,
	}
	return contracts.WriteJSON(stdout, env)
}

func optionalBoolFlag(name string, value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("--%s must be true or false", name)
	}
	return &parsed, nil
}

func writeSyncJSON(stdout io.Writer, result syncer.Result, err error) error {
	if err == nil {
		return contracts.WriteJSON(stdout, contracts.NewSuccess("sync", result))
	}
	var partial syncer.PartialFailure
	if errors.As(err, &partial) {
		env := contracts.NewError("sync", "SYNC_PARTIAL_FAILURE", err.Error(), contracts.CategoryAPI, true)
		env.Data = result
		if writeErr := contracts.WriteJSON(stdout, env); writeErr != nil {
			return writeErr
		}
		return cliExit{exitCode: 6}
	}
	return err
}

func writeSyncHuman(stdout io.Writer, result syncer.Result, verbose bool) {
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "warning\t%s\t%s\n", warning.Code, warning.Message)
	}
	if len(result.Items) == 0 {
		return
	}
	var okCount, errorCount int
	for _, item := range result.Items {
		if item.Status == "ok" {
			okCount++
		} else {
			errorCount++
		}
		if verbose {
			fmt.Fprintf(stdout, "%s\t%s\t%s\taccounts=%d\tadded=%d\tmodified=%d\tremoved=%d\n",
				item.Provider, item.ProviderItemID, item.Status, item.AccountsSeen,
				item.TransactionsAdded, item.TransactionsModified, item.TransactionsRemoved)
		}
	}
	if !verbose {
		fmt.Fprintf(stdout, "synced\tok=%d\terrors=%d\n", okCount, errorCount)
	}
}

func requireStore(state *runtimeState) (store.Store, error) {
	if state.store == nil {
		cfg, err := config.Load(config.Options{ConfigPath: state.configPath})
		if err != nil {
			return nil, err
		}
		opened, err := store.OpenEncrypted(context.Background(), cfg.DatabasePath, cfg.DatabaseEncryptionKeyBytes)
		if err != nil {
			return nil, err
		}
		state.store = opened
	}
	return state.store, nil
}

func commandName(cmd *cobra.Command) string {
	if cmd == nil {
		return "unknown"
	}
	return strings.ReplaceAll(cmd.CommandPath(), "money ", ".")
}
