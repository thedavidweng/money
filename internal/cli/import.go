package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/money/internal/contracts"
	"github.com/thedavidweng/money/internal/importsource"
)

func newImportCommand(ctx context.Context, state *runtimeState, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Import accounts and transactions from external sources",
		Example: "  money import monarch transactions.csv\n  money import csv transactions.csv --dry-run\n  money import monarch data.csv --batch-id 20240101 --confirm",
	}
	registry := importsource.DefaultRegistry()
	for _, name := range registry.Names() {
		sourceName := name
		var batchID string
		var dryRun, confirm bool
		sourceCmd := &cobra.Command{
			Use:   sourceName + " <file>",
			Short: "Import accounts and transactions from " + sourceName,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if state.json && !dryRun && !confirm {
					return &cliError{
						command:   "import." + sourceName,
						code:      "CONFIRMATION_REQUIRED",
						message:   "JSON import writes require --dry-run or --confirm",
						category:  contracts.CategoryValidation,
						retryable: false,
						exitCode:  7,
					}
				}
				if batchID == "" {
					batchID = time.Now().UTC().Format("20060102T150405Z")
				}
				source, ok := registry.Get(sourceName)
				if !ok {
					return fmt.Errorf("import source %q is not registered", sourceName)
				}
				if dryRun {
					return render(stdout, state, "import."+sourceName, map[string]any{"dry_run": true, "file": args[0], "batch_id": batchID}, func() {
						_, _ = fmt.Fprintf(stdout, "Would import %s from %s (batch %s)\n", sourceName, args[0], batchID)
					})
				}
				activeStore, err := requireStore(state)
				if err != nil {
					return err
				}
				f, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				result, err := source.Import(ctx, activeStore, batchID, f)
				if err != nil {
					var importErr importsource.ImportError
					if errors.As(err, &importErr) {
						return &cliError{
							command:   "import." + sourceName,
							code:      importErr.Code,
							message:   importErr.Message,
							category:  contracts.CategoryValidation,
							retryable: false,
							exitCode:  7,
						}
					}
					return err
				}
				return render(stdout, state, "import."+sourceName, map[string]any{"result": result, "file": args[0], "batch_id": batchID}, func() {
					_, _ = fmt.Fprintf(stdout, "Imported %d accounts and %d transactions from %s.\n", result.AccountsImported, result.TransactionsImported, args[0])
					if result.DuplicatesSkipped > 0 {
						_, _ = fmt.Fprintf(stdout, "Skipped %d duplicate rows.\n", result.DuplicatesSkipped)
					}
					if len(result.PossibleDuplicates) > 0 {
						_, _ = fmt.Fprintf(stdout, "Possible duplicates across sources: %d\n", len(result.PossibleDuplicates))
					}
				})
			},
		}
		sourceCmd.Flags().StringVar(&batchID, "batch-id", "", "import batch id (default: timestamp)")
		sourceCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show import plan without writing")
		sourceCmd.Flags().BoolVar(&confirm, "confirm", false, "confirm import")
		cmd.AddCommand(sourceCmd)
	}
	return cmd
}
