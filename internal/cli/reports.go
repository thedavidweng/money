package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newCashflowCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var fromDate, toDate, period, currency string
	cmd := &cobra.Command{
		Use:     "cashflow",
		Short:   "Show income and expenses over time",
		Example: "  money cashflow --from 2024-01-01 --to 2024-12-31\n  money cashflow --from 2024-01-01 --to 2024-12-31 --period yearly --json",
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
			return render(stdout, state, "cashflow", map[string]any{"periods": periods}, func() {
				rows := make([][]string, 0, len(periods))
				for _, p := range periods {
					rows = append(rows, []string{p.Period, colorAmount(stdout, p.Income), colorAmount(stdout, p.Expenses), colorAmount(stdout, p.Net)})
				}
				renderTable(stdout, []string{"PERIOD", "INCOME", "EXPENSES", "NET"}, rows)
			})
		},
	}
	cmd.Flags().StringVar(&fromDate, "from", "", "start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&toDate, "to", "", "end date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&period, "period", "monthly", "grouping period: monthly or yearly")
	_ = cmd.RegisterFlagCompletionFunc("period", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"monthly", "yearly"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&currency, "currency", "USD", "currency to report")
	return cmd
}

func newNetWorthCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "net-worth",
		Short:   "Show current net worth across all visible accounts",
		Example: "  money net-worth\n  money net-worth --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			nw, err := activeStore.NetWorth(ctx)
			if err != nil {
				return err
			}
			return render(stdout, state, "net_worth", map[string]any{"net_worth": nw}, func() {
				_, _ = fmt.Fprintf(stdout, "Net worth: %s %s\n", colorAmount(stdout, nw.Total), nw.Currency)
			})
		},
	}
	return cmd
}
