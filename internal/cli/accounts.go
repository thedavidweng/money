package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/core"
)

func newAccountsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:     "accounts",
		Short:   "Manage accounts",
		Example: "  money accounts list\n  money accounts list --verbose --json\n  money accounts create-manual --name Savings --type depository --balance 1000.00 --currency USD --confirm",
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List accounts",
		Example: "  money accounts list\n  money accounts list --verbose\n  money accounts list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			accounts, err := activeStore.ListAccounts(ctx)
			if err != nil {
				return err
			}
			return render(stdout, state, "accounts.list", map[string]any{"accounts": accounts}, func() {
				table := tablewriter.NewWriter(stdout)
				if verbose {
					table.SetHeader([]string{"ID", "NAME", "TYPE", "BALANCE", "AVAILABLE", "AVAILABLE CREDIT", "CURRENCY", "SOURCE", "PROVIDER", "PROVIDER ACCOUNT ID", "UPDATED"})
				} else {
					table.SetHeader([]string{"NAME", "TYPE", "BALANCE", "CURRENCY", "SOURCE"})
				}
				table.SetBorder(false)
				table.SetAutoWrapText(false)
				for i := range accounts {
					a := &accounts[i]
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
			})
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
		Use:     "create-manual",
		Short:   "Create a local manual account",
		Example: "  money accounts create-manual --name Savings --type depository --balance 5000.00 --currency USD --confirm\n  money accounts create-manual --name \"Credit Card\" --type credit --balance 500.00 --dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.json && !dryRun && !confirm {
				return &cliError{
					command:   "accounts.create_manual",
					code:      "CONFIRMATION_REQUIRED",
					message:   "JSON manual account writes require --dry-run or --confirm",
					category:  contracts.CategoryValidation,
					retryable: false,
					exitCode:  7,
				}
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
				return writeManualPlan(stdout, state, &plan)
			}
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			account, err := activeStore.CreateManualAccount(ctx, &core.Account{
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
			return render(stdout, state, "accounts.create_manual", map[string]any{"account": account, "plan": plan}, func() {
				_, _ = fmt.Fprintf(stdout, "Created %s with balance %s %s\n", account.DisplayName, colorAmount(stdout, account.CurrentBalance), account.Currency)
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "account name")
	cmd.Flags().StringVar(&accountType, "type", "", "account type")
	_ = cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"depository", "credit", "investment", "loan", "property", "vehicle", "other_asset", "other_liability"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&subtype, "subtype", "", "account subtype")
	cmd.Flags().StringVar(&balance, "balance", "", "unsigned balance")
	cmd.Flags().StringVar(&currency, "currency", "USD", "currency")
	cmd.Flags().StringVar(&alias, "alias", "", "local account alias")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show write plan without saving")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "save the manual account")
	return cmd
}

type manualAccountPlan struct {
	WillWrite         bool   `json:"will_write"`
	AccountName       string `json:"account_name"`
	AccountType       string `json:"account_type"`
	SignedBalance     string `json:"signed_balance"`
	Currency          string `json:"currency"`
	FinancialPosition string `json:"financial_position"`
}
