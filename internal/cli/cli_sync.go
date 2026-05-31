package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/syncer"
)

func newSyncCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	var providerName, providerItemID, startDate, endDate string
	var verbose bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync linked Provider Items",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeStore, err := requireStore(state)
			if err != nil {
				return err
			}
			cfg, err := config.Load(config.Options{ConfigPath: state.configPath, Profile: state.profile})
			if err != nil {
				return err
			}
			result, err := syncer.Sync(ctx, activeStore, providers.NewRegistry(cfg), syncer.Options{
				Provider:       providerName,
				ProviderItemID: providerItemID,
				StartDate:      startDate,
				EndDate:        endDate,
			})
			if state.json {
				return writeSyncJSON(stdout, result, err)
			}
			writeSyncHuman(stdout, result, verbose)
			return err
		},
	}
	cmd.Flags().StringVar(&providerName, "provider", "", "sync only one provider")
	cmd.Flags().StringVar(&providerItemID, "provider-item", "", "sync only one provider item")
	cmd.Flags().StringVar(&startDate, "start-date", "", "backfill transactions from this date (YYYY-MM-DD); requires --end-date")
	cmd.Flags().StringVar(&endDate, "end-date", "", "backfill transactions until this date (YYYY-MM-DD); requires --start-date")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "show per-provider-item sync details")
	return cmd
}

func writeSyncJSON(stdout io.Writer, result syncer.Result, err error) error {
	if err == nil {
		return contracts.WriteJSON(stdout, contracts.NewSuccess("sync", result))
	}
	var partial syncer.PartialFailure
	if errors.As(err, &partial) {
		env := contracts.NewError("sync", "SYNC_PARTIAL_FAILURE", err.Error(), contracts.CategoryAPI, true)
		env.Data = result
		if writeErr := contracts.WriteJSON(stdout, env); writeErr != nil {
			return writeErr
		}
		return cliExit{exitCode: 6}
	}
	return err
}

func writeSyncHuman(stdout io.Writer, result syncer.Result, verbose bool) {
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "warning\t%s\t%s\n", warning.Code, warning.Message)
	}
	if len(result.Items) == 0 {
		return
	}
	var okCount, errorCount int
	for _, item := range result.Items {
		if item.Status == "ok" {
			okCount++
		} else {
			errorCount++
		}
		if verbose {
			fmt.Fprintf(stdout, "%s\t%s\t%s\taccounts=%d\tadded=%d\tmodified=%d\tremoved=%d\n",
				item.Provider, item.ProviderItemID, item.Status, item.AccountsSeen,
				item.TransactionsAdded, item.TransactionsModified, item.TransactionsRemoved)
		}
	}
	if !verbose {
		fmt.Fprintf(stdout, "synced\tok=%d\terrors=%d\n", okCount, errorCount)
	}
}
