package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newInvestmentsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "investments",
		Short:   "Manage investment accounts",
		Example: "  money investments holdings\n  money investments securities --json",
	}
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
			return render(stdout, state, "investments.holdings", map[string]any{"holdings": holdings}, func() {
				rows := make([][]string, 0, len(holdings))
				for _, h := range holdings {
					rows = append(rows, []string{h.AccountID, h.SecurityID, fmt.Sprintf("%.4f", h.Quantity), colorAmountFloat(stdout, h.InstitutionPrice), colorAmountFloat(stdout, h.InstitutionValue), h.Currency})
				}
				renderTable(stdout, []string{"ACCOUNT", "SECURITY", "QUANTITY", "PRICE", "VALUE", "CURRENCY"}, rows)
			})
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
			return render(stdout, state, "investments.securities", map[string]any{"securities": securities}, func() {
				rows := make([][]string, 0, len(securities))
				for i := range securities {
					sec := &securities[i]
					ticker := "-"
					if sec.TickerSymbol != nil {
						ticker = *sec.TickerSymbol
					}
					rows = append(rows, []string{sec.SecurityID, sec.Name, ticker, sec.Type, colorAmountFloat(stdout, sec.ClosePrice), sec.Currency})
				}
				renderTable(stdout, []string{"SECURITY ID", "NAME", "TICKER", "TYPE", "CLOSE PRICE", "CURRENCY"}, rows)
			})
		},
	})
	return cmd
}

func newLiabilitiesCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "liabilities",
		Short:   "Manage liabilities",
		Example: "  money liabilities list\n  money liabilities list --json",
	}
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
			return render(stdout, state, "liabilities.list", map[string]any{"liabilities": liabilities}, func() {
				rows := make([][]string, 0, len(liabilities))
				for i := range liabilities {
					l := &liabilities[i]
					rows = append(rows, []string{l.AccountID, l.Type, l.Name, colorAmountFloat(stdout, l.CurrentBalance), l.Currency})
				}
				renderTable(stdout, []string{"ACCOUNT", "TYPE", "NAME", "BALANCE", "CURRENCY"}, rows)
			})
		},
	})
	return cmd
}
