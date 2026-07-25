package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/store"
)

func newCategoriesCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:     "categories",
		Short:   "Manage transaction categories",
		Example: "  money categories list\n  money categories list --verbose --json",
	}
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List categories",
		Example: "  money categories list\n  money categories list --verbose",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			categories, err := activeStore.ListCategories(ctx)
			if err != nil {
				return err
			}
			return render(stdout, state, "categories.list", map[string]any{"categories": categories}, func() {
				headers := []string{"NAME", "GROUP", "HIDDEN"}
				if verbose {
					headers = []string{"ID", "NAME", "GROUP", "HIDDEN"}
				}
				rows := make([][]string, 0, len(categories))
				for _, c := range categories {
					group := "-"
					if c.GroupName != nil {
						group = *c.GroupName
					}
					hidden := ""
					if c.Hidden {
						hidden = "yes"
					}
					if verbose {
						rows = append(rows, []string{c.ID, c.Name, group, hidden})
					} else {
						rows = append(rows, []string{c.Name, group, hidden})
					}
				}
				renderTable(stdout, headers, rows)
			})
		},
	}
	listCmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs")
	cmd.AddCommand(listCmd)
	return cmd
}

func newTagsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:     "tags",
		Short:   "Manage transaction tags",
		Example: "  money tags list\n  money tags list --verbose",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			tags, err := activeStore.ListTags(ctx)
			if err != nil {
				return err
			}
			return render(stdout, state, "tags.list", map[string]any{"tags": tags}, func() {
				headers := []string{"NAME"}
				if verbose {
					headers = []string{"ID", "NAME"}
				}
				rows := make([][]string, 0, len(tags))
				for _, t := range tags {
					if verbose {
						rows = append(rows, []string{t.ID, t.Name})
					} else {
						rows = append(rows, []string{t.Name})
					}
				}
				renderTable(stdout, headers, rows)
			})
		},
	}
	listCmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs")
	cmd.AddCommand(listCmd)
	return cmd
}

func newItemsCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "items",
		Short:   "Manage linked provider items",
		Example: "  money items list\n  money items get <id>\n  money items rename <id> \"My Bank\"\n  money items remove <id>",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List linked provider items",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			items, err := activeStore.ListProviderItems(ctx, store.ProviderItemQuery{})
			if err != nil {
				return err
			}
			return render(stdout, state, "items.list", map[string]any{"items": items}, func() {
				rows := make([][]string, 0, len(items))
				for i := range items {
					item := &items[i]
					alias := item.Alias
					if alias == "" {
						alias = "-"
					}
					rows = append(rows, []string{item.ID, item.Provider, item.InstitutionID, alias, item.Status, strings.Join(item.Products, ",")})
				}
				renderTable(stdout, []string{"ID", "PROVIDER", "INSTITUTION", "ALIAS", "STATUS", "PRODUCTS"}, rows)
			})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get <id>",
		Short: "Get a linked provider item",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			activeStore, err := requireStore(state)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			items, err := activeStore.ListProviderItems(ctx, store.ProviderItemQuery{})
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var ids []string
			for i := range items {
				ids = append(ids, items[i].ID+"\t"+items[i].Provider+": "+items[i].InstitutionID)
			}
			return ids, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			item, err := activeStore.GetProviderItem(ctx, args[0])
			if err != nil {
				return err
			}
			return render(stdout, state, "items.get", map[string]any{"item": item}, func() {
				alias := item.Alias
				if alias == "" {
					alias = "-"
				}
				renderTable(stdout, []string{"ID", "PROVIDER", "INSTITUTION", "ALIAS", "STATUS", "PRODUCTS"}, [][]string{{item.ID, item.Provider, item.InstitutionID, alias, item.Status, strings.Join(item.Products, ",")}})
			})
		},
	})
	renameCmd := &cobra.Command{
		Use:   "rename <id> <name>",
		Short: "Rename a linked provider item alias",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				activeStore, err := requireStore(state)
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				items, err := activeStore.ListProviderItems(ctx, store.ProviderItemQuery{})
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				var ids []string
				for i := range items {
					ids = append(ids, items[i].ID+"\t"+items[i].Provider+": "+items[i].InstitutionID)
				}
				return ids, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			if err := activeStore.UpdateProviderItemName(ctx, args[0], args[1]); err != nil {
				return err
			}
			return render(stdout, state, "items.rename", map[string]string{"id": args[0], "alias": args[1]}, func() {
				_, _ = fmt.Fprintf(stdout, "Renamed %s to %s\n", args[0], args[1])
			})
		},
	}
	cmd.AddCommand(renameCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a linked provider item",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			activeStore, err := requireStore(state)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			items, err := activeStore.ListProviderItems(ctx, store.ProviderItemQuery{})
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var ids []string
			for i := range items {
				ids = append(ids, items[i].ID+"\t"+items[i].Provider+": "+items[i].InstitutionID)
			}
			return ids, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			if err := activeStore.RemoveProviderItem(ctx, args[0]); err != nil {
				return err
			}
			return render(stdout, state, "items.remove", map[string]string{"id": args[0]}, func() {
				_, _ = fmt.Fprintf(stdout, "Removed %s\n", args[0])
			})
		},
	})
	return cmd
}

func newRecurringCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:     "recurring",
		Short:   "Manage recurring transactions",
		Example: "  money recurring list\n  money recurring list --verbose --json",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recurring transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			recurringItems, err := activeStore.ListRecurring(ctx)
			if err != nil {
				return err
			}
			return render(stdout, state, "recurring.list", map[string]any{"recurring": recurringItems}, func() {
				headers := []string{"MERCHANT", "AMOUNT", "FREQUENCY", "NEXT DATE"}
				if verbose {
					headers = []string{"ID", "ACCOUNT", "MERCHANT", "AMOUNT", "FREQUENCY", "NEXT DATE"}
				}
				rows := make([][]string, 0, len(recurringItems))
				for i := range recurringItems {
					r := &recurringItems[i]
					nextDate := "-"
					if r.NextDate != nil {
						nextDate = *r.NextDate
					}
					if verbose {
						rows = append(rows, []string{r.ID, r.AccountID, r.MerchantName, colorAmount(stdout, r.AverageAmount), r.Frequency, nextDate})
					} else {
						rows = append(rows, []string{r.MerchantName, colorAmount(stdout, r.AverageAmount), r.Frequency, nextDate})
					}
				}
				renderTable(stdout, headers, rows)
			})
		},
	}
	listCmd.Flags().BoolVar(&verbose, "verbose", false, "show local IDs and account IDs")
	cmd.AddCommand(listCmd)
	return cmd
}
