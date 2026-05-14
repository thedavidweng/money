package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/core"
)

func newBudgetsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "budgets"}
	cmd.AddCommand(newBudgetsListCommand(ctx, state, stdout))
	cmd.AddCommand(newBudgetsCreateCommand(ctx, state, stdout))
	cmd.AddCommand(newBudgetsGetCommand(ctx, state, stdout))
	cmd.AddCommand(newBudgetsDeleteCommand(ctx, state, stdout))
	cmd.AddCommand(newBudgetCategoriesCommand(ctx, state, stdout))
	return cmd
}

func newBudgetsListCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List budgets",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			budgets, err := activeStore.ListBudgets(ctx)
			if err != nil {
				return err
			}
			if !state.json {
				table := tablewriter.NewWriter(stdout)
				if verbose {
					table.SetHeader([]string{"ID", "NAME", "PERIOD", "START", "END", "CURRENCY", "CATEGORIES"})
				} else {
					table.SetHeader([]string{"NAME", "PERIOD", "START", "END", "CURRENCY"})
				}
				table.SetBorder(false)
				for _, b := range budgets {
					if verbose {
						table.Append([]string{b.ID, b.Name, b.Period, b.StartDate, b.EndDate, b.Currency, strconv.Itoa(len(b.Categories))})
					} else {
						table.Append([]string{b.Name, b.Period, b.StartDate, b.EndDate, b.Currency})
					}
				}
				table.Render()
				return nil
			}
			env := contracts.NewSuccess("budgets.list", map[string]any{"budgets": budgets})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs and category counts")
	return cmd
}

func newBudgetsCreateCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var name, currency, period, startDate, endDate string
	var dryRun, confirm bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a budget",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.json && !dryRun && !confirm {
				return fmt.Errorf("JSON budget writes require --dry-run or --confirm")
			}
			if name == "" || startDate == "" || endDate == "" {
				return fmt.Errorf("budget requires --name, --start-date, and --end-date")
			}
			if period != "monthly" && period != "yearly" {
				return fmt.Errorf("--period must be monthly or yearly")
			}
			budget := core.Budget{
				Name:      name,
				Currency:  currency,
				Period:    period,
				StartDate: startDate,
				EndDate:   endDate,
			}
			if dryRun {
				if state.json {
					env := contracts.NewSuccess("budgets.create", map[string]any{"dry_run": true, "budget": budget})
					env.Meta.Demo = state.demo
					return contracts.WriteJSON(stdout, env)
				}
				fmt.Fprintf(stdout, "Would create budget %q (%s, %s to %s)\n", budget.Name, budget.Period, budget.StartDate, budget.EndDate)
				return nil
			}
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			created, err := activeStore.CreateBudget(ctx, budget)
			if err != nil {
				return err
			}
			if state.json {
				env := contracts.NewSuccess("budgets.create", map[string]any{"budget": created})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Created budget %s (%s)\n", created.Name, created.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "budget name")
	cmd.Flags().StringVar(&currency, "currency", "USD", "budget currency")
	cmd.Flags().StringVar(&period, "period", "monthly", "budget period: monthly or yearly")
	cmd.Flags().StringVar(&startDate, "start-date", "", "budget start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "budget end date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show write plan without saving")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "save the budget")
	return cmd
}

func newBudgetsGetCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a budget with its categories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			budget, err := activeStore.GetBudget(ctx, args[0])
			if err != nil {
				return err
			}
			if !state.json {
				fmt.Fprintf(stdout, "Budget: %s (%s)\n", budget.Name, budget.ID)
				fmt.Fprintf(stdout, "Period: %s\n", budget.Period)
				fmt.Fprintf(stdout, "Range: %s to %s\n", budget.StartDate, budget.EndDate)
				fmt.Fprintf(stdout, "Currency: %s\n", budget.Currency)
				if len(budget.Categories) > 0 {
					fmt.Fprintln(stdout, "Categories:")
					table := tablewriter.NewWriter(stdout)
					table.SetHeader([]string{"NAME", "LIMIT", "CATEGORY ID"})
					table.SetBorder(false)
					for _, bc := range budget.Categories {
						catID := "-"
						if bc.CategoryID != nil {
							catID = *bc.CategoryID
						}
						table.Append([]string{bc.Name, colorAmount(stdout, bc.Limit), catID})
					}
					table.Render()
				}
				return nil
			}
			env := contracts.NewSuccess("budgets.get", map[string]any{"budget": budget})
			env.Meta.Demo = state.demo
			return contracts.WriteJSON(stdout, env)
		},
	}
}

func newBudgetsDeleteCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a budget",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			if err := activeStore.DeleteBudget(ctx, args[0]); err != nil {
				return err
			}
			if state.json {
				env := contracts.NewSuccess("budgets.delete", map[string]string{"id": args[0]})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Deleted budget %s\n", args[0])
			return nil
		},
	}
}

func newBudgetCategoriesCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{Use: "categories"}
	cmd.AddCommand(newBudgetCategoriesCreateCommand(ctx, state, stdout))
	cmd.AddCommand(newBudgetCategoriesDeleteCommand(ctx, state, stdout))
	return cmd
}

func newBudgetCategoriesCreateCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var budgetID, name, categoryID, currency string
	var limitMinor int64
	var dryRun, confirm bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a budget category",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.json && !dryRun && !confirm {
				return fmt.Errorf("JSON budget category writes require --dry-run or --confirm")
			}
			if budgetID == "" || name == "" || limitMinor <= 0 {
				return fmt.Errorf("budget category requires --budget-id, --name, and --limit")
			}
			bc := core.BudgetCategory{
				BudgetID:        budgetID,
				CategoryID:      nil,
				Name:            name,
				LimitMinorUnits: limitMinor,
				Currency:        currency,
			}
			if categoryID != "" {
				bc.CategoryID = &categoryID
			}
			if dryRun {
				if state.json {
					env := contracts.NewSuccess("budgets.categories.create", map[string]any{"dry_run": true, "budget_category": bc})
					env.Meta.Demo = state.demo
					return contracts.WriteJSON(stdout, env)
				}
				fmt.Fprintf(stdout, "Would create budget category %q with limit %s %s\n", bc.Name, core.FormatMinorUnits(bc.LimitMinorUnits, bc.Currency), bc.Currency)
				return nil
			}
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			created, err := activeStore.CreateBudgetCategory(ctx, bc)
			if err != nil {
				return err
			}
			if state.json {
				env := contracts.NewSuccess("budgets.categories.create", map[string]any{"budget_category": created})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Created budget category %s (%s)\n", created.Name, created.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&budgetID, "budget-id", "", "parent budget ID")
	cmd.Flags().StringVar(&name, "name", "", "category name")
	cmd.Flags().StringVar(&categoryID, "category-id", "", "linked local category ID")
	cmd.Flags().Int64Var(&limitMinor, "limit", 0, "spending limit in minor units (e.g. 50000 for $500.00)")
	cmd.Flags().StringVar(&currency, "currency", "USD", "currency")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show write plan without saving")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "save the budget category")
	return cmd
}

func newBudgetCategoriesDeleteCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a budget category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			if err := activeStore.DeleteBudgetCategory(ctx, args[0]); err != nil {
				return err
			}
			if state.json {
				env := contracts.NewSuccess("budgets.categories.delete", map[string]string{"id": args[0]})
				env.Meta.Demo = state.demo
				return contracts.WriteJSON(stdout, env)
			}
			fmt.Fprintf(stdout, "Deleted budget category %s\n", args[0])
			return nil
		},
	}
}
