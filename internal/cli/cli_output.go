package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"

	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/core"
)

type renderOpt func(*contracts.Envelope)

func withPagination(p *contracts.Pagination) renderOpt {
	return func(env *contracts.Envelope) { env.Meta.Pagination = p }
}

func render(stdout io.Writer, state *runtimeState, command string, data any, table func(), opts ...renderOpt) error {
	if !state.json {
		table()
		return nil
	}
	env := contracts.NewSuccess(command, data)
	env.Meta.Demo = state.demo
	for _, opt := range opts {
		opt(&env)
	}
	return contracts.WriteJSON(stdout, env)
}

func writeManualPlan(stdout io.Writer, state *runtimeState, plan manualAccountPlan) error {
	return render(stdout, state, "accounts.create_manual", map[string]any{"plan": plan}, func() {
		_, _ = fmt.Fprintf(stdout, "Would create %s with balance %s %s\n", plan.AccountName, colorAmount(stdout, plan.SignedBalance), plan.Currency)
	})
}

func writeTransactionsPage(stdout io.Writer, state *runtimeState, command string, transactions []core.Transaction, limit int, offset int, verbose bool) error {
	page := &contracts.Pagination{Limit: limit, Offset: offset, HasMore: len(transactions) == limit}
	return render(stdout, state, command, map[string]any{"transactions": transactions}, func() {
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
		if !verbose {
			return
		}
		for _, tx := range transactions {
			_, _ = fmt.Fprintln(stdout)
			_, _ = fmt.Fprintf(stdout, "  ID: %s\n", tx.ID)
			_, _ = fmt.Fprintf(stdout, "  Account ID: %s\n", tx.AccountID)
			if tx.AuthorizedDate != nil {
				_, _ = fmt.Fprintf(stdout, "  Authorized Date: %s\n", *tx.AuthorizedDate)
			}
			if tx.ProviderCategory != nil {
				_, _ = fmt.Fprintf(stdout, "  Provider Category: %s\n", *tx.ProviderCategory)
			}
			if tx.ProviderSubcategory != nil {
				_, _ = fmt.Fprintf(stdout, "  Provider Subcategory: %s\n", *tx.ProviderSubcategory)
			}
			if tx.Note != nil {
				_, _ = fmt.Fprintf(stdout, "  Note: %s\n", *tx.Note)
			}
			if len(tx.Tags) > 0 {
				tagNames := make([]string, 0, len(tx.Tags))
				for _, tag := range tx.Tags {
					tagNames = append(tagNames, tag.Name)
				}
				_, _ = fmt.Fprintf(stdout, "  Tags: %s\n", strings.Join(tagNames, ", "))
			}
			if tx.Source.Provider != nil {
				_, _ = fmt.Fprintf(stdout, "  Source Provider: %s\n", *tx.Source.Provider)
			}
			if tx.Source.ProviderItemID != nil {
				_, _ = fmt.Fprintf(stdout, "  Source Provider Item ID: %s\n", *tx.Source.ProviderItemID)
			}
			if tx.Source.ProviderAccountID != nil {
				_, _ = fmt.Fprintf(stdout, "  Source Provider Account ID: %s\n", *tx.Source.ProviderAccountID)
			}
			if tx.Source.ProviderTransactionID != nil {
				_, _ = fmt.Fprintf(stdout, "  Source Provider Transaction ID: %s\n", *tx.Source.ProviderTransactionID)
			}
			_, _ = fmt.Fprintf(stdout, "  Last Changed: %s\n", tx.LastChangedAt)
		}
	}, withPagination(page))
}
