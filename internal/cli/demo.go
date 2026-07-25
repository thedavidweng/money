package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/store"
)

func newFeedbackCommand(state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:     "feedback",
		Short:   "Open the project's GitHub issues page in a browser",
		Example: "  money feedback",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := "https://github.com/thedavidweng/money/issues"
			return render(stdout, state, "feedback", map[string]string{"url": url}, func() {
				_, _ = fmt.Fprintf(stdout, "Opening %s\n", url)
				if err := openBrowser(url); err != nil {
					_, _ = fmt.Fprintf(stdout, "Could not open browser automatically. Visit %s\n", url)
				}
			})
		},
	}
}

func newDemoCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo <command...>",
		Short: "Run a command against bundled non-persistent sample data",
		Example: `  money demo accounts list
  money demo transactions list --verbose
  money demo net-worth
  money demo categories list`,
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
