package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/core"
	"github.com/thedavidweng/money/internal/importsource"
	"github.com/thedavidweng/money/internal/linking"
	"github.com/thedavidweng/money/internal/plaidlogin"
	"github.com/thedavidweng/money/internal/prompt"
	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
	"github.com/thedavidweng/money/internal/syncer"
)

// Set via -ldflags at build time.
var (
	Version = "dev"
	Commit  = "dev"
)

type runtimeState struct {
	store      store.Store
	demo       bool
	json       bool
	configPath string
	profile    string
	stdin      io.Reader
	stderr     io.Writer
	prompter   prompt.Selector
}

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

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	state := &runtimeState{stdin: stdin, stderr: stderr}
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
		Use:   "money",
		Short: "A local-first finance backend for external AI agents",
		Long: `money is a local-first finance CLI.

If this is your first time, run:

  money setup

This creates your config, encryption key, and database. After setup, you can
link financial institutions and sync transactions locally.
`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVarP(&state.json, "json", "j", false, "write a JSON envelope to stdout")
	root.PersistentFlags().StringVar(&state.configPath, "config", "", "config file path")
	root.PersistentFlags().StringVar(&state.profile, "profile", "default", "configuration profile")

	root.AddGroup(&cobra.Group{ID: "start", Title: "Getting Started"})
	root.AddGroup(&cobra.Group{ID: "data", Title: "Data"})
	root.AddGroup(&cobra.Group{ID: "config", Title: "Configuration"})
	root.AddGroup(&cobra.Group{ID: "utils", Title: "Utilities"})

	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print version",
		GroupID: "utils",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.json {
				env := contracts.NewSuccess("version", map[string]string{"version": Version, "commit": Commit})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "money %s (commit %s)\n", Version, Commit)
			return nil
		},
	}
	root.AddCommand(versionCmd)

	setupCmd := newSetupCommand(ctx, state, stdout)
	setupCmd.GroupID = "start"
	root.AddCommand(setupCmd)

	doctorCmd := newDoctorCommand(ctx, state, stdout)
	doctorCmd.GroupID = "start"
	root.AddCommand(doctorCmd)

	demoCmd := newDemoCommand(ctx, state, stdout, stderr)
	demoCmd.GroupID = "start"
	root.AddCommand(demoCmd)

	accountsCmd := newAccountsCommand(ctx, state, stdout)
	accountsCmd.GroupID = "data"
	root.AddCommand(accountsCmd)

	transactionsCmd := newTransactionsCommand(ctx, state, stdout)
	transactionsCmd.GroupID = "data"
	root.AddCommand(transactionsCmd)

	categoriesCmd := newCategoriesCommand(ctx, state, stdout)
	categoriesCmd.GroupID = "data"
	root.AddCommand(categoriesCmd)

	tagsCmd := newTagsCommand(ctx, state, stdout)
	tagsCmd.GroupID = "data"
	root.AddCommand(tagsCmd)

	recurringCmd := newRecurringCommand(ctx, state, stdout)
	recurringCmd.GroupID = "data"
	root.AddCommand(recurringCmd)

	itemsCmd := newItemsCommand(ctx, state, stdout)
	itemsCmd.GroupID = "data"
	root.AddCommand(itemsCmd)

	investmentsCmd := newInvestmentsCommand(ctx, state, stdout)
	investmentsCmd.GroupID = "data"
	root.AddCommand(investmentsCmd)

	liabilitiesCmd := newLiabilitiesCommand(ctx, state, stdout)
	liabilitiesCmd.GroupID = "data"
	root.AddCommand(liabilitiesCmd)

	importCmd := newImportCommand(ctx, state, stdout)
	importCmd.GroupID = "data"
	root.AddCommand(importCmd)

	cashflowCmd := newCashflowCommand(ctx, state, stdout)
	cashflowCmd.GroupID = "data"
	root.AddCommand(cashflowCmd)

	netWorthCmd := newNetWorthCommand(ctx, state, stdout)
	netWorthCmd.GroupID = "data"
	root.AddCommand(netWorthCmd)

	budgetsCmd := newBudgetsCommand(ctx, state, stdout)
	budgetsCmd.GroupID = "data"
	root.AddCommand(budgetsCmd)

	rulesCmd := newRulesCommand(ctx, state, stdout)
	rulesCmd.GroupID = "data"
	root.AddCommand(rulesCmd)

	syncCmd := newSyncCommand(ctx, state, stdout)
	syncCmd.GroupID = "data"
	root.AddCommand(syncCmd)

	linkCmd := newLinkCommand(ctx, state, stdout)
	linkCmd.GroupID = "config"
	root.AddCommand(linkCmd)

	providersCmd := newProvidersCommand(ctx, state, stdout, stderr)
	providersCmd.GroupID = "config"
	root.AddCommand(providersCmd)

	plaidCmd := newPlaidCommand(ctx, state, stdout, stderr)
	plaidCmd.GroupID = "config"
	root.AddCommand(plaidCmd)

	feedbackCmd := newFeedbackCommand(state, stdout)
	feedbackCmd.GroupID = "utils"
	root.AddCommand(feedbackCmd)

	txAlias := newTransactionsCommand(ctx, state, stdout)
	txAlias.Use = "tx"
	txAlias.Aliases = nil
	txAlias.GroupID = "data"
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

func newInvestmentsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "investments"}
	cmd.AddCommand(&cobra.Command{
		Use:   "holdings",
		Short: "List investment holdings",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			holdings, err := activeStore.ListHoldings(ctx)
			if err != nil {
				return err
			}
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"ACCOUNT", "SECURITY", "QUANTITY", "PRICE", "VALUE", "CURRENCY"})
				table.SetBorder(false)
				for _, h := range holdings {
					table.Append([]string{h.AccountID, h.SecurityID, fmt.Sprintf("%.4f", h.Quantity), colorAmountFloat(stdout, h.InstitutionPrice), colorAmountFloat(stdout, h.InstitutionValue), h.Currency})
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("investments.holdings", map[string]any{"holdings": holdings})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "securities",
		Short: "List investment securities",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			securities, err := activeStore.ListSecurities(ctx)
			if err != nil {
				return err
			}
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"SECURITY ID", "NAME", "TICKER", "TYPE", "CLOSE PRICE", "CURRENCY"})
				table.SetBorder(false)
				for _, sec := range securities {
					ticker := "-"
					if sec.TickerSymbol != nil {
						ticker = *sec.TickerSymbol
					}
					table.Append([]string{sec.SecurityID, sec.Name, ticker, sec.Type, colorAmountFloat(stdout, sec.ClosePrice), sec.Currency})
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("investments.securities", map[string]any{"securities": securities})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	return cmd
}

func newLiabilitiesCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "liabilities"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List liabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			liabilities, err := activeStore.ListLiabilities(ctx)
			if err != nil {
				return err
			}
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"ACCOUNT", "TYPE", "NAME", "BALANCE", "CURRENCY"})
				table.SetBorder(false)
				for _, l := range liabilities {
					table.Append([]string{l.AccountID, l.Type, l.Name, colorAmountFloat(stdout, l.CurrentBalance), l.Currency})
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("liabilities.list", map[string]any{"liabilities": liabilities})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	return cmd
}

func newFeedbackCommand(state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "feedback",
		Short: "Open the project's GitHub issues page in a browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := "https://github.com/thedavidweng/money/issues"
			if state.json {
				env := contracts.NewSuccess("feedback", map[string]string{"url": url})
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Opening %s\n", url)
			if err := openBrowser(url); err != nil {
				fmt.Fprintf(stdout, "Could not open browser automatically. Visit %s\n", url)
			}
			return nil
		},
	}
}

func newDemoCommand(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo <command...>",
		Short: "Run a command against bundled non-persistent sample data",
		Long: `demo runs money commands against bundled non-persistent sample data.

This is a safe sandbox to explore the CLI without linking real institutions.
All data is stored in memory and discarded when the command exits.
`,
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

	cmd.AddGroup(&cobra.Group{ID: "data", Title: "Data"})

	accountsCmd := newAccountsCommand(ctx, state, stdout)
	accountsCmd.GroupID = "data"
	cmd.AddCommand(accountsCmd)

	transactionsCmd := newTransactionsCommand(ctx, state, stdout)
	transactionsCmd.GroupID = "data"
	cmd.AddCommand(transactionsCmd)

	categoriesCmd := newCategoriesCommand(ctx, state, stdout)
	categoriesCmd.GroupID = "data"
	cmd.AddCommand(categoriesCmd)

	tagsCmd := newTagsCommand(ctx, state, stdout)
	tagsCmd.GroupID = "data"
	cmd.AddCommand(tagsCmd)

	recurringCmd := newRecurringCommand(ctx, state, stdout)
	recurringCmd.GroupID = "data"
	cmd.AddCommand(recurringCmd)

	itemsCmd := newItemsCommand(ctx, state, stdout)
	itemsCmd.GroupID = "data"
	cmd.AddCommand(itemsCmd)

	investmentsCmd := newInvestmentsCommand(ctx, state, stdout)
	investmentsCmd.GroupID = "data"
	cmd.AddCommand(investmentsCmd)

	liabilitiesCmd := newLiabilitiesCommand(ctx, state, stdout)
	liabilitiesCmd.GroupID = "data"
	cmd.AddCommand(liabilitiesCmd)

	budgetsCmd := newBudgetsCommand(ctx, state, stdout)
	budgetsCmd.GroupID = "data"
	cmd.AddCommand(budgetsCmd)

	rulesCmd := newRulesCommand(ctx, state, stdout)
	rulesCmd.GroupID = "data"
	cmd.AddCommand(rulesCmd)

	txAlias := newTransactionsCommand(ctx, state, stdout)
	txAlias.Use = "tx"
	txAlias.Aliases = nil
	txAlias.GroupID = "data"
	cmd.AddCommand(txAlias)
	return cmd
}

func newAccountsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{Use: "accounts"}
	listCmd := &cobra.Command{
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
				table := tablewriter.NewWriter(stdout)
				if verbose {
					table.SetHeader([]string{"ID", "NAME", "TYPE", "BALANCE", "AVAILABLE", "AVAILABLE CREDIT", "CURRENCY", "SOURCE", "PROVIDER", "PROVIDER ACCOUNT ID", "UPDATED"})
				} else {
					table.SetHeader([]string{"NAME", "TYPE", "BALANCE", "CURRENCY", "SOURCE"})
				}
				table.SetBorder(false)
				table.SetAutoWrapText(false)
				for _, a := range accounts {
					if verbose {
						avail := "-"
						if a.AvailableBalance != nil {
							avail = *a.AvailableBalance
						}
						availCredit := "-"
						if a.AvailableCredit != nil {
							availCredit = *a.AvailableCredit
						}
						provider := "-"
						if a.Source.Provider != nil {
							provider = *a.Source.Provider
						}
						providerAccountID := "-"
						if a.Source.ProviderAccountID != nil {
							providerAccountID = *a.Source.ProviderAccountID
						}
						table.Append([]string{a.ID, a.DisplayName, a.Type, colorAmount(stdout, a.CurrentBalance), colorAmount(stdout, avail), colorAmount(stdout, availCredit), a.Currency, a.Source.Kind, provider, providerAccountID, a.UpdatedAt})
					} else {
						table.Append([]string{a.DisplayName, a.Type, colorAmount(stdout, a.CurrentBalance), a.Currency, a.Source.Kind})
					}
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("accounts.list", map[string]any{"accounts": accounts})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
	listCmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs, provider provenance, and available balances")
	cmd.AddCommand(listCmd)
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
			fmt.Fprintf(stdout, "Created %s with balance %s %s\n", account.DisplayName, colorAmount(stdout, account.CurrentBalance), account.Currency)
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
	var searchVerbose bool
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
			return writeTransactionsPage(stdout, state, "transactions.search", transactions, searchLimit, 0, searchVerbose)
		},
	}
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "maximum transactions to return")
	searchCmd.Flags().BoolVar(&searchVerbose, "verbose", false, "show local IDs, source provenance, notes, tags, and provider categories")
	cmd.AddCommand(searchCmd)
	return cmd
}

func newTransactionsListCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var removedMode string
	var accountID, categoryID, merchant, tagID, dateFrom, dateTo string
	var needsReview, pending, recurring string
	var limit, offset int
	var verbose bool
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
			return writeTransactionsPage(stdout, state, "transactions.list", transactions, limit, offset, verbose)
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
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs, source provenance, notes, tags, and provider categories")
	return cmd
}

func newCategoriesCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{Use: "categories"}
	listCmd := &cobra.Command{
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
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				if verbose {
					table.SetHeader([]string{"ID", "NAME", "GROUP", "HIDDEN"})
				} else {
					table.SetHeader([]string{"NAME", "GROUP", "HIDDEN"})
				}
				table.SetBorder(false)
				for _, c := range categories {
					group := "-"
					if c.GroupName != nil {
						group = *c.GroupName
					}
					hidden := ""
					if c.Hidden {
						hidden = "yes"
					}
					if verbose {
						table.Append([]string{c.ID, c.Name, group, hidden})
					} else {
						table.Append([]string{c.Name, group, hidden})
					}
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("categories.list", map[string]any{"categories": categories})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
	listCmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs")
	cmd.AddCommand(listCmd)
	return cmd
}

func newTagsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{Use: "tags"}
	listCmd := &cobra.Command{
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
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				if verbose {
					table.SetHeader([]string{"ID", "NAME"})
				} else {
					table.SetHeader([]string{"NAME"})
				}
				table.SetBorder(false)
				for _, t := range tags {
					if verbose {
						table.Append([]string{t.ID, t.Name})
					} else {
						table.Append([]string{t.Name})
					}
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("tags.list", map[string]any{"tags": tags})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
	listCmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs")
	cmd.AddCommand(listCmd)
	return cmd
}

func newItemsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "items"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List linked provider items",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			items, err := activeStore.ListProviderItems(ctx, store.ProviderItemQuery{})
			if err != nil {
				return err
			}
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"ID", "PROVIDER", "INSTITUTION", "ALIAS", "STATUS", "PRODUCTS"})
				table.SetBorder(false)
				for _, item := range items {
					alias := item.Alias
					if alias == "" {
						alias = "-"
					}
					table.Append([]string{item.ID, item.Provider, item.InstitutionID, alias, item.Status, strings.Join(item.Products, ",")})
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("items.list", map[string]any{"items": items})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "Get a linked provider item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			item, err := activeStore.GetProviderItem(ctx, args[0])
			if err != nil {
				return err
			}
			env := contracts.NewSuccess("items.get", map[string]any{"item": item})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	})
	renameCmd := &cobra.Command{
		Use:   "rename <id> <name>",
		Short: "Rename a linked provider item alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			if err := activeStore.UpdateProviderItemName(ctx, args[0], args[1]); err != nil {
				return err
			}
			if state.json {
				env := contracts.NewSuccess("items.rename", map[string]string{"id": args[0], "alias": args[1]})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Renamed %s to %s\n", args[0], args[1])
			return nil
		},
	}
	cmd.AddCommand(renameCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a linked provider item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			if err := activeStore.RemoveProviderItem(ctx, args[0]); err != nil {
				return err
			}
			if state.json {
				env := contracts.NewSuccess("items.remove", map[string]string{"id": args[0]})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Removed %s\n", args[0])
			return nil
		},
	})
	return cmd
}

func newRecurringCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{Use: "recurring"}
	listCmd := &cobra.Command{
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
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				if verbose {
					table.SetHeader([]string{"ID", "ACCOUNT", "MERCHANT", "AMOUNT", "FREQUENCY", "NEXT DATE"})
				} else {
					table.SetHeader([]string{"MERCHANT", "AMOUNT", "FREQUENCY", "NEXT DATE"})
				}
				table.SetBorder(false)
				for _, r := range recurringItems {
					nextDate := "-"
					if r.NextDate != nil {
						nextDate = *r.NextDate
					}
					if verbose {
						table.Append([]string{r.ID, r.AccountID, r.MerchantName, colorAmount(stdout, r.AverageAmount), r.Frequency, nextDate})
					} else {
						table.Append([]string{r.MerchantName, colorAmount(stdout, r.AverageAmount), r.Frequency, nextDate})
					}
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("recurring.list", map[string]any{"recurring": recurringItems})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
	listCmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs and account IDs")
	cmd.AddCommand(listCmd)
	return cmd
}

func newImportCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "import"}
	registry := importsource.DefaultRegistry()
	for _, name := range registry.Names() {
		sourceName := name
		var batchID string
		var dryRun, confirm bool
		sourceCmd := &cobra.Command{
			Use:   sourceName + " <file>",
			Short: "Import accounts and transactions from " + sourceName,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if state.json && !dryRun && !confirm {
					return fmt.Errorf("JSON import writes require --dry-run or --confirm")
				}
				if batchID == "" {
					batchID = time.Now().UTC().Format("20060102T150405Z")
				}
				source, ok := registry.Get(sourceName)
				if !ok {
					return fmt.Errorf("import source %q is not registered", sourceName)
				}
				if dryRun {
					if state.json {
						env := contracts.NewSuccess("import."+sourceName, map[string]any{"dry_run": true, "file": args[0], "batch_id": batchID})
						env.Meta.Demo = state.demo
						return contracts.WriteJSON(stdout, env)
					}
					fmt.Fprintf(stdout, "Would import %s from %s (batch %s)\n", sourceName, args[0], batchID)
					return nil
				}
				activeStore, err := requireStore(state)
				if err != nil {
					return err
				}
				importStore, ok := activeStore.(importsource.ImportStore)
				if !ok {
					return fmt.Errorf("active store does not support imports")
				}
				f, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer f.Close()
				result, err := source.Import(ctx, importStore, batchID, f)
				if err != nil {
					var importErr importsource.ImportError
					if errors.As(err, &importErr) {
						return cliError{
							command:   "import." + sourceName,
							code:      importErr.Code,
							message:   importErr.Message,
							category:  contracts.CategoryValidation,
							retryable: false,
							exitCode:  7,
						}
					}
					return err
				}
				if state.json {
					env := contracts.NewSuccess("import."+sourceName, map[string]any{"result": result, "file": args[0], "batch_id": batchID})
					env.Meta.Demo = state.demo
					return contracts.WriteJSON(stdout, env)
				}
				fmt.Fprintf(stdout, "Imported %d accounts and %d transactions from %s.\n", result.AccountsImported, result.TransactionsImported, args[0])
				if result.DuplicatesSkipped > 0 {
					fmt.Fprintf(stdout, "Skipped %d duplicate rows.\n", result.DuplicatesSkipped)
				}
				if len(result.PossibleDuplicates) > 0 {
					fmt.Fprintf(stdout, "Possible duplicates across sources: %d\n", len(result.PossibleDuplicates))
				}
				return nil
			},
		}
		sourceCmd.Flags().StringVar(&batchID, "batch-id", "", "import batch id (default: timestamp)")
		sourceCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show import plan without writing")
		sourceCmd.Flags().BoolVar(&confirm, "confirm", false, "confirm import")
		cmd.AddCommand(sourceCmd)
	}
	return cmd
}

func newCashflowCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var fromDate, toDate, period, currency string
	cmd := &cobra.Command{
		Use:   "cashflow",
		Short: "Show income and expenses over time",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromDate == "" || toDate == "" {
				return fmt.Errorf("cashflow requires --from and --to dates")
			}
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			periods, err := activeStore.CashflowSummary(ctx, fromDate, toDate, period, currency)
			if err != nil {
				return err
			}
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"PERIOD", "INCOME", "EXPENSES", "NET"})
				table.SetBorder(false)
				for _, p := range periods {
					table.Append([]string{p.Period, colorAmount(stdout, p.Income), colorAmount(stdout, p.Expenses), colorAmount(stdout, p.Net)})
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("cashflow", map[string]any{"periods": periods})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
	cmd.Flags().StringVar(&fromDate, "from", "", "start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&toDate, "to", "", "end date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&period, "period", "monthly", "grouping period: monthly or yearly")
	cmd.Flags().StringVar(&currency, "currency", "USD", "currency to report")
	return cmd
}

func newNetWorthCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "net-worth",
		Short: "Show current net worth across all visible accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			nw, err := activeStore.NetWorth(ctx)
			if err != nil {
				return err
			}
			if !state.json {
				fmt.Fprintf(stdout, "Net worth: %s %s\n", colorAmount(stdout, nw.Total), nw.Currency)
				return nil
			}
			env := contracts.NewSuccess("net_worth", map[string]any{"net_worth": nw})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
	return cmd
}

func newSyncCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var providerName, providerItemID, startDate, endDate string
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
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath, Profile: state.profile})
			if err != nil {
				return err
			}
			result, err := syncer.Sync(ctx, syncStore, providers.NewRegistry(cfg), syncer.Options{
				Provider:       providerName,
				ProviderItemID: providerItemID,
				StartDate:      startDate,
				EndDate:        endDate,
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
	cmd.Flags().StringVar(&startDate, "start-date", "", "backfill transactions from this date (YYYY-MM-DD); requires --end-date")
	cmd.Flags().StringVar(&endDate, "end-date", "", "backfill transactions until this date (YYYY-MM-DD); requires --start-date")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show per-provider-item sync details")
	return cmd
}

func newProvidersCommand(ctx context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "providers"}
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
		Use:   "plaid",
		Short: "Plaid-specific setup and Dashboard commands",
	}
	cmd.AddCommand(newPlaidLoginCommand(ctx, state, stdout, stderr, "plaid.login"))
	cmd.AddCommand(newPlaidLogoutCommand(state, stdout, "plaid.logout"))
	cmd.AddCommand(newPlaidSandboxCommand(ctx, state, stdout))
	return cmd
}

func newPlaidSandboxCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sandbox",
		Short: "Plaid Sandbox helpers",
	}
	cmd.AddCommand(newPlaidSandboxLinkCommand(ctx, state, stdout))
	return cmd
}

func newPlaidSandboxLinkCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var institutionID, products string
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Create and store a Plaid Sandbox Provider Item",
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
	return cmd
}

func newPlaidLoginCommand(_ context.Context, state *runtimeState, stdout io.Writer, stderr io.Writer, commandName string) *cobra.Command {
	var noOpen, force bool
	var team, environment, products, countryCodes, redirectURI string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Plaid Dashboard and fetch API keys",
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
	cmd.Flags().StringVar(&products, "products", "", "comma-separated Plaid Link products to write to config")
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
	fmt.Fprintf(stderr, "Plaid Dashboard OAuth URL: %s\n", authURL)
	if !opts.NoOpen && !state.json {
		if state.stdin == nil {
			return fmt.Errorf("stdin is required before opening a browser")
		}
		fmt.Fprintln(stderr, "Press Enter to open the browser.")
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
	fmt.Fprintf(stdout, "Plaid Dashboard login complete for team %s.\n", result.TeamID)
	switch result.CredentialAction {
	case "preserved_existing":
		fmt.Fprintf(stdout, "Plaid %s credentials already exist; preserved (use --force to overwrite).\n", result.Environment)
	default:
		fmt.Fprintf(stdout, "Plaid %s credentials written to %s.\n", result.Environment, result.EnvPath)
	}
	fmt.Fprintf(stdout, "Next: %s\n", result.NextCommand)
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
		exitCode = 2
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
		Use:   "logout",
		Short: "Remove stored Plaid Dashboard auth without deleting API keys",
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
				fmt.Fprintf(stdout, "Plaid Dashboard auth removed from %s.\n", authPath)
			} else {
				fmt.Fprintf(stdout, "Plaid Dashboard auth was not present at %s.\n", authPath)
			}
			fmt.Fprintf(stdout, "API keys remain in %s. To remove them, edit the file or run money providers configure plaid with new values.\n", meta.EnvPath)
			return nil
		},
	}
	return cmd
}

func newLinkCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var providerName, institutionID string
	var noOpen bool
	var additionalConsentedProducts, requiredIfSupportedProducts, optionalProducts string
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
					message:   providerName + " Link requires a browser flow; omit --json for the live link flow.",
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
	linkStore, ok := activeStore.(linking.Store)
	if !ok {
		return fmt.Errorf("active store cannot persist linked provider items")
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

	if !state.json && state.stderr != nil {
		fmt.Fprintf(state.stderr, "Plaid Link URL: %s\n", server.LinkURL())
	}
	if !opts.NoOpen && !state.json {
		if state.stdin == nil {
			return fmt.Errorf("stdin is required before opening a browser")
		}
		if state.stderr != nil {
			fmt.Fprintln(state.stderr, "Press Enter to open the browser.")
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
	result, err := linking.CompleteProviderLink(ctx, linkStore, provider, session, callback)
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
	fmt.Fprintf(stdout, "Linked %s Provider Item %s.\n", result.Provider, result.ProviderItemID)
	fmt.Fprintln(stdout, "No sync was run. Run `money sync` after linking.")
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
	linkStore, ok := activeStore.(linking.Store)
	if !ok {
		return fmt.Errorf("active store cannot persist linked provider items")
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
	result, err := linking.CompleteProviderLink(ctx, linkStore, provider, session, providers.LinkCallback{
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
	fmt.Fprintf(stdout, "Linked %s Sandbox Provider Item %s.\n", result.Provider, result.ProviderItemID)
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
	fmt.Fprintf(stdout, "Would create %s with balance %s %s\n", plan.AccountName, colorAmount(stdout, plan.SignedBalance), plan.Currency)
	return nil
}

func writeTransactionsPage(stdout io.Writer, state *runtimeState, command string, transactions []core.Transaction, limit int, offset int, verbose bool) error {
	if !state.json {
		table := tablewriter.NewWriter(stdout)
		table.SetHeader([]string{"DATE", "ACCOUNT", "MERCHANT", "AMOUNT", "CATEGORY", "STATUS"})
		table.SetBorder(false)
		table.SetAutoWrapText(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		for _, tx := range transactions {
			cat := ""
			if tx.CategoryName != nil {
				cat = *tx.CategoryName
			}
			status := ""
			if tx.Pending {
				status = "pending"
			}
			if tx.NeedsReview {
				if status != "" {
					status += ","
				}
				status += "review"
			}
			if tx.Removed {
				status = "removed"
			}
			merchant := tx.MerchantName
			if merchant == "" {
				merchant = tx.Name
			}
			accountName := tx.AccountName
			if accountName == "" {
				accountName = "-"
			}
			table.Append([]string{tx.Date, accountName, merchant, colorAmount(stdout, tx.Amount), cat, status})
		}
		table.Render()
		if verbose {
			for _, tx := range transactions {
				fmt.Fprintln(stdout)
				fmt.Fprintf(stdout, "  ID: %s\n", tx.ID)
				fmt.Fprintf(stdout, "  Account ID: %s\n", tx.AccountID)
				if tx.AuthorizedDate != nil {
					fmt.Fprintf(stdout, "  Authorized Date: %s\n", *tx.AuthorizedDate)
				}
				if tx.ProviderCategory != nil {
					fmt.Fprintf(stdout, "  Provider Category: %s\n", *tx.ProviderCategory)
				}
				if tx.ProviderSubcategory != nil {
					fmt.Fprintf(stdout, "  Provider Subcategory: %s\n", *tx.ProviderSubcategory)
				}
				if tx.Note != nil {
					fmt.Fprintf(stdout, "  Note: %s\n", *tx.Note)
				}
				if len(tx.Tags) > 0 {
					tagNames := make([]string, 0, len(tx.Tags))
					for _, tag := range tx.Tags {
						tagNames = append(tagNames, tag.Name)
					}
					fmt.Fprintf(stdout, "  Tags: %s\n", strings.Join(tagNames, ", "))
				}
				if tx.Source.Provider != nil {
					fmt.Fprintf(stdout, "  Source Provider: %s\n", *tx.Source.Provider)
				}
				if tx.Source.ProviderItemID != nil {
					fmt.Fprintf(stdout, "  Source Provider Item ID: %s\n", *tx.Source.ProviderItemID)
				}
				if tx.Source.ProviderAccountID != nil {
					fmt.Fprintf(stdout, "  Source Provider Account ID: %s\n", *tx.Source.ProviderAccountID)
				}
				if tx.Source.ProviderTransactionID != nil {
					fmt.Fprintf(stdout, "  Source Provider Transaction ID: %s\n", *tx.Source.ProviderTransactionID)
				}
				fmt.Fprintf(stdout, "  Last Changed: %s\n", tx.LastChangedAt)
			}
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
		cfg, err := config.Load(config.Options{ConfigPath: state.configPath, Profile: state.profile})
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

var noColorForced = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

func supportsColor(w io.Writer) bool {
	if noColorForced {
		return false
	}
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func colorAmount(w io.Writer, amount string) string {
	if !supportsColor(w) {
		return amount
	}
	if amount == "" || amount == "-" || amount == "0" || amount == "0.00" || amount == "-0.00" {
		return amount
	}
	if strings.HasPrefix(amount, "-") {
		return "\033[31m" + amount + "\033[0m"
	}
	return "\033[32m" + amount + "\033[0m"
}

func colorAmountFloat(w io.Writer, amount float64) string {
	return colorAmount(w, fmt.Sprintf("%.2f", amount))
}
