package cli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/store"
)

func newTransactionsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transactions",
		Aliases: []string{"transaction"},
		Short:   "Manage transactions",
		Example: "  money transactions list\n  money transactions search \"grocery\"\n  money tx list --account acc_123 --date-from 2024-01-01",
	}
	cmd.AddCommand(newTransactionsListCommand(ctx, state, stdout))
	var searchLimit int
	var searchVerbose bool
	searchCmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Search transactions",
		Example: "  money transactions search \"whole foods\"\n  money transactions search uber --limit 10",
		Args:    cobra.ExactArgs(1),
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
		Use:     "list",
		Short:   "List transactions",
		Example: "  money transactions list\n  money transactions list --category groceries --date-from 2024-01-01\n  money transactions list --merchant amazon --limit 20 --json",
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
	_ = cmd.RegisterFlagCompletionFunc("removed", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"exclude", "include", "only"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&accountID, "account", "", "filter by account id")
	cmd.Flags().StringVar(&categoryID, "category", "", "filter by category id")
	cmd.Flags().StringVar(&merchant, "merchant", "", "filter by merchant or transaction name")
	cmd.Flags().StringVar(&tagID, "tag", "", "filter by tag id")
	cmd.Flags().StringVar(&dateFrom, "date-from", "", "filter by inclusive start date")
	cmd.Flags().StringVar(&dateTo, "date-to", "", "filter by inclusive end date")
	cmd.Flags().StringVar(&needsReview, "needs-review", "", "filter by review state: true or false")
	_ = cmd.RegisterFlagCompletionFunc("needs-review", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&pending, "pending", "", "filter by pending state: true or false")
	_ = cmd.RegisterFlagCompletionFunc("pending", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&recurring, "recurring", "", "filter by recurring state: true or false")
	_ = cmd.RegisterFlagCompletionFunc("recurring", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum transactions to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "transactions to skip")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs, source provenance, notes, tags, and provider categories")
	return cmd
}
