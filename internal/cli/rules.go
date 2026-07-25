package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/core"
)

func newRulesCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rules",
		Short:   "Manage transaction categorization rules",
		Example: "  money rules list\n  money rules create --name \"Groceries\" --condition-field merchant_name --condition-op contains --condition-value \"whole foods\" --action-type set_category --action-value groceries --confirm\n  money rules apply\n  money rules delete <id>",
	}
	cmd.AddCommand(newRulesListCommand(ctx, state, stdout))
	cmd.AddCommand(newRulesCreateCommand(ctx, state, stdout))
	cmd.AddCommand(newRulesDeleteCommand(ctx, state, stdout))
	cmd.AddCommand(newRulesApplyCommand(ctx, state, stdout))
	return cmd
}

func newRulesListCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			rules, err := activeStore.ListRules(ctx)
			if err != nil {
				return err
			}
			return render(stdout, state, "rules.list", map[string]any{"rules": rules}, func() {
				rows := make([][]string, 0, len(rules))
				for i := range rules {
					r := &rules[i]
					condition := fmt.Sprintf("%s %s %q", r.ConditionField, r.ConditionOp, r.ConditionValue)
					action := fmt.Sprintf("%s %q", r.ActionType, r.ActionValue)
					rows = append(rows, []string{r.Name, condition, action, fmt.Sprintf("%d", r.Priority)})
				}
				renderTable(stdout, []string{"NAME", "IF", "THEN", "PRIORITY"}, rows)
			})
		},
	}
}

func newRulesCreateCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var name, conditionField, conditionOp, conditionValue, actionType, actionValue string
	var priority int
	var dryRun, confirm bool
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a rule",
		Example: "  money rules create --name \"Mark Uber\" --condition-field merchant_name --condition-op contains --condition-value uber --action-type set_category --action-value transport --confirm\n  money rules create --name \"Add note\" --condition-field name --condition-op equals --condition-value \"Starbucks\" --action-type set_note --action-value \"coffee\" --dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state.json && !dryRun && !confirm {
				return &cliError{
					command:   "rules.create",
					code:      "CONFIRMATION_REQUIRED",
					message:   "JSON rule writes require --dry-run or --confirm",
					category:  contracts.CategoryValidation,
					retryable: false,
					exitCode:  7,
				}
			}
			if name == "" || conditionField == "" || conditionOp == "" || conditionValue == "" || actionType == "" || actionValue == "" {
				return fmt.Errorf("rule requires --name, --condition-field, --condition-op, --condition-value, --action-type, and --action-value")
			}
			rule := core.Rule{
				Name:           name,
				ConditionField: conditionField,
				ConditionOp:    conditionOp,
				ConditionValue: conditionValue,
				ActionType:     actionType,
				ActionValue:    actionValue,
				Priority:       priority,
				Enabled:        true,
			}
			if dryRun {
				return render(stdout, state, "rules.create", map[string]any{"dry_run": true, "rule": rule}, func() {
					_, _ = fmt.Fprintf(stdout, "Would create rule %q: if %s %s %q then %s %q (priority %d)\n",
						rule.Name, rule.ConditionField, rule.ConditionOp, rule.ConditionValue, rule.ActionType, rule.ActionValue, rule.Priority)
				})
			}
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			created, err := activeStore.CreateRule(ctx, &rule)
			if err != nil {
				return err
			}
			return render(stdout, state, "rules.create", map[string]any{"rule": created}, func() {
				_, _ = fmt.Fprintf(stdout, "Created rule %s (%s)\n", created.Name, created.ID)
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "rule name")
	cmd.Flags().StringVar(&conditionField, "condition-field", "", "field to match: merchant_name or name")
	_ = cmd.RegisterFlagCompletionFunc("condition-field", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"merchant_name", "name"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&conditionOp, "condition-op", "", "operator: contains or equals")
	_ = cmd.RegisterFlagCompletionFunc("condition-op", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"contains", "equals"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&conditionValue, "condition-value", "", "value to match")
	cmd.Flags().StringVar(&actionType, "action-type", "", "action: set_category or set_note")
	_ = cmd.RegisterFlagCompletionFunc("action-type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"set_category", "set_note"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&actionValue, "action-value", "", "action value")
	cmd.Flags().IntVar(&priority, "priority", 0, "rule priority (higher first)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show write plan without saving")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "save the rule")
	return cmd
}

func newRulesDeleteCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a rule",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			activeStore, err := requireStore(state)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			rules, err := activeStore.ListRules(ctx)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var ids []string
			for i := range rules {
				ids = append(ids, rules[i].ID+"\t"+rules[i].Name)
			}
			return ids, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			if err := activeStore.DeleteRule(ctx, args[0]); err != nil {
				return err
			}
			return render(stdout, state, "rules.delete", map[string]string{"id": args[0]}, func() {
				_, _ = fmt.Fprintf(stdout, "Deleted rule %s\n", args[0])
			})
		},
	}
}

func newRulesApplyCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "apply",
		Short: "Apply all enabled rules to transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			result, err := activeStore.ApplyRules(ctx)
			if err != nil {
				return err
			}
			return render(stdout, state, "rules.apply", map[string]any{"result": result}, func() {
				_, _ = fmt.Fprintf(stdout, "Updated %d transactions\n", result.TransactionsUpdated)
			})
		},
	}
}
