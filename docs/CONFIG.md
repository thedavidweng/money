# Configuration Loading

`money` configuration is explicit. It does not silently read donor config paths, cwd `.env`, or provider environment variables that are not referenced by config.

## Config Shape

Secret references use YAML objects with exactly one `env` key:

```yaml
database:
  path: ~/.money/data/money.db
  encryption_key:
    env: MONEY_DB_ENCRYPTION_KEY

providers:
  plaid:
    client_id:
      env: PLAID_CLIENT_ID
    secret:
      env: PLAID_SECRET
    environment: sandbox
    products: [transactions]
    country_codes: [US]
    additional_consented_products: [investments]
    required_if_supported_products: [liabilities]
    optional_products: [auth]
    redirect_uri:
      env: PLAID_REDIRECT_URI
  bridge:
    client_id:
      env: BRIDGE_CLIENT_ID
    client_secret:
      env: BRIDGE_CLIENT_SECRET
```

Direct scalar values are allowed only for non-secrets such as `database.path`, `providers.plaid.environment`, `products`, `country_codes`, and Plaid Link consent product lists. Direct scalar secrets are accepted for manually edited files, but config loading emits a structured warning recommending `.env` references.

## Path Resolution

1. If `--config <path>` is present, use it.
2. Else if `MONEY_CONFIG` is set, use it.
3. Else use the `--profile` flag to determine the default config path:
   - `default` (or not specified): `~/.money/config.yaml`
   - Any other profile name: `~/.money/profiles/<name>/config.yaml`
4. If config has `env_file`, resolve it relative to the config file's directory when it is not absolute.
5. Else the env companion is `.env` in the config file's directory.
6. Never load cwd `.env` implicitly.

Profile names must be alphanumeric, hyphen, or underscore only; path traversal characters are rejected.

`~` expansion is supported only at the start of configured paths.

## Load Order

1. Resolve the config path.
2. Read config YAML if it exists. Missing config is a config error for real commands unless the command is `setup`, `doctor --fix`, or demo mode.
3. Determine the env companion path from config or default rules.
4. Load key/value pairs from that env file if it exists.
5. Overlay process environment values over env-file values for the same variable names.
6. Resolve each `env:` reference by looking up the named variable in the merged env map.
7. Validate required fields for the command being executed.
8. Return config values plus structured warnings, such as direct secrets in YAML or broad env-file permissions.

Environment variables complete explicit references; they do not form a magic override chain. For example, `PLAID_SECRET` is used only when config says `secret: { env: PLAID_SECRET }` or a setup/configure command writes that reference.

Plaid Dashboard OAuth bootstrap state is stored outside YAML at `plaid-dashboard-auth.json` beside the resolved config file. It is local bootstrap state for `money plaid login`, written `0600`, and may include Dashboard access/refresh tokens plus selected `team_id` and `client_id`. Provider API credentials still use the normal `.env` plus YAML `env:` references model.

## Pseudocode

```text
load(flags, process_env):
  config_path = flags.config || process_env["MONEY_CONFIG"] || "~/.money/config.yaml"
  config_path = expand_home(config_path)
  raw = read_yaml(config_path)

  env_path = raw.env_file
    ? resolve_relative_to_config_dir(raw.env_file)
    : join(dirname(config_path), ".env")

  file_env = parse_dotenv(env_path)
  merged_env = file_env overlaid by process_env

  resolved.database.path = expand_home(required_scalar(raw.database.path))
  resolved.database.encryption_key = resolve_secret(raw.database.encryption_key, merged_env)
  resolved.read_only = raw.read_only || process_env["MONEY_READ_ONLY"] == "1"

  for provider in raw.providers:
    resolve explicit secret refs from merged_env
    copy non-secret scalar/list values from YAML
    collect warnings for direct secret scalars

  validate command-specific requirements
  return resolved config, config paths, warnings
```

## Database Encryption Key

Generated keys are exactly 32 cryptographically random bytes, encoded with base64url without padding into `MONEY_DB_ENCRYPTION_KEY`. That is 256 bits of key material.

Config loading decodes the base64url string into 32 bytes. Any malformed base64url value or decoded length other than 32 bytes is a config error. The store adapter converts those bytes into the selected SQLite encryption driver's key input format without printing or logging the key.

## Demo Mode

`money demo <command...>` does not require `MONEY_DB_ENCRYPTION_KEY`, does not open the real database, and does not run real migrations against disk. Demo mode opens an in-memory SQLite database, runs the same migration SQL without encryption, seeds deterministic fixtures, marks JSON envelopes with `meta.demo: true`, and blocks real Provider network flows.

This is not a plaintext fallback for real data because demo never reads or writes real financial records.
