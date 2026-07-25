package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"
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
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"ACCOUNT", "SECURITY", "QUANTITY", "PRICE", "VALUE", "CURRENCY"})
				table.SetBorder(false)
				for _, h := range holdings {
					table.Append([]string{h.AccountID, h.SecurityID, fmt.Sprintf("%.4f", h.Quantity), colorAmountFloat(stdout, h.InstitutionPrice), colorAmountFloat(stdout, h.InstitutionValue), h.Currency})
				}
				table.Render()
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
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"SECURITY ID", "NAME", "TICKER", "TYPE", "CLOSE PRICE", "CURRENCY"})
				table.SetBorder(false)
				for i := range securities {
					sec := &securities[i]
					ticker := "-"
					if sec.TickerSymbol != nil {
						ticker = *sec.TickerSymbol
					}
					table.Append([]string{sec.SecurityID, sec.Name, ticker, sec.Type, colorAmountFloat(stdout, sec.ClosePrice), sec.Currency})
				}
				table.Render()
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
				table := tablewriter.NewWriter(stdout)
				table.SetHeader([]string{"ACCOUNT", "TYPE", "NAME", "BALANCE", "CURRENCY"})
				table.SetBorder(false)
				for i := range liabilities {
					l := &liabilities[i]
					table.Append([]string{l.AccountID, l.Type, l.Name, colorAmountFloat(stdout, l.CurrentBalance), l.Currency})
				}
				table.Render()
			})
		},
	})
	return cmd
}
