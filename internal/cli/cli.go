package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/prompt"
	"github.com/thedavidweng/money/internal/store"
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

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	state := &runtimeState{stdin: stdin, stderr: stderr}
	defer func() {
		if closer, ok := state.store.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()
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
					_, _ = fmt.Fprintln(stderr, writeErr)
				}
			} else {
				_, _ = fmt.Fprintln(stderr, cliErr.message)
			}
			return cliErr.exitCode
		}
		if state.json {
			env := contracts.NewError(commandName(root), "COMMAND_FAILED", err.Error(), contracts.CategoryInternal, false)
			if writeErr := contracts.WriteJSON(stdout, env); writeErr != nil {
				_, _ = fmt.Fprintln(stderr, writeErr)
			}
		} else {
			_, _ = fmt.Fprintln(stderr, err)
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
		Version: fmt.Sprintf("%s (commit %s)", Version, Commit),
		Example: `  money setup
  money link "Chase"
  money sync
  money transactions list --category groceries
  money accounts list --json
  money demo transactions list`,
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

	completionCmd := newCompletionCommand(stdout)
	completionCmd.GroupID = "utils"
	root.AddCommand(completionCmd)

	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Print version",
		Example: "  money version\n  money version --json",
		GroupID: "utils",
		RunE: func(cmd *cobra.Command, args []string) error {
			return render(stdout, state, "version", map[string]string{"version": Version, "commit": Commit}, func() {
				_, _ = fmt.Fprintf(stdout, "money %s (commit %s)\n", Version, Commit)
			})
		},
	}
	root.AddCommand(versionCmd)

	setupCmd := newSetupCommand(ctx, state, stdout)
	setupCmd.GroupID = "start"
	root.AddCommand(setupCmd)

	doctorCmd := newDoctorCommand(ctx, state, stdout)
	doctorCmd.GroupID = "start"
	root.AddCommand(doctorCmd)

	demoCmd := newDemoCommand(ctx, state, stdout)
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

func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

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
