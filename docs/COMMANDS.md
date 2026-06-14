# Commands

## Getting Started

```text
money setup                         Initialize configuration and encrypted database
money doctor                        Check configuration and system health (--fix, --dry-run)
money demo <command>                Run against non-persistent sample data
```

## Data Queries

```text
money accounts list                 List financial accounts
money accounts create-manual        Create a local manual account
money transactions list             List transactions with filters
money transactions search           Search transactions by text
money tx                            Alias for transactions
money categories list               List transaction categories
money tags list                     List transaction tags
money recurring list                List recurring transactions
money investments holdings          List investment holdings
money investments securities        List investment securities
money liabilities list              List liabilities
money items list                    List linked provider items
money items get <id>                Get a linked provider item
money items rename <id> <name>      Rename a provider item alias
money items remove <id>             Remove a linked provider item with cascade delete
money import <source> <file>        Import data from external sources (source: monarch, csv)
money cashflow                      Show cashflow summary by period
money net-worth                     Show net worth breakdown
```

## Budgets and Rules

```text
money budgets list                  List budgets
money budgets create                Create a budget
money budgets get <id>              Get budget details with categories
money budgets delete <id>           Delete a budget
money budgets categories create     Add a category to a budget
money budgets categories delete     Remove a category from a budget
money rules list                    List transaction rules
money rules create                  Create a transaction rule
money rules delete <id>             Delete a transaction rule
money rules apply                   Apply rules to transactions
```

## Provider Management

```text
money link                          Link a financial institution
money providers configure <provider> Configure provider credentials
money plaid login                    Sign in to Plaid Dashboard and fetch API keys
money plaid logout                   Remove Plaid Dashboard auth; keep API keys
money plaid sandbox link             Create and store a Plaid Sandbox Provider Item
money providers plaid link          Link a Plaid Provider Item
money providers bridge link         Link a Bridge Provider Item
money sync                          Sync linked provider data (supports --start-date/--end-date)
```

## Utilities

```text
money feedback                      Open the project GitHub issues page
money version                       Print version
money completion                    Generate shell completions
```

## Global Flags

```text
--config string            config file path
--profile string           configuration profile (default "default")
-j, --json                 write a JSON envelope to stdout
```

Read commands and provisional sync diagnostics support `--json` for machine-readable output. Manual write operations require `--dry-run` or `--confirm`.
