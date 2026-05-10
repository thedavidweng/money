# money Context

`money` is the local-first personal finance backend context for deterministic finance data access. This context exists to keep the project's language independent from donor project names while preserving the useful finance boundaries learned from them.

## Language

**money**:
A local-first personal finance backend that exposes deterministic finance primitives through local command contracts.
_Avoid_: OpenRay, Ray Finance, hosted finance app

**OpenRay Finance**:
A prior extraction concept from Ray Finance used as donor/reference language, not the product identity for this project.
_Avoid_: money, current binary name, current config name

**Provider**:
A networked financial data integration source, such as Plaid, Bridge, MX, or Finicity, that connects institution data to `money`.
_Avoid_: API, bank API, model provider

**Institution**:
A bank or financial institution that can be connected through one or more Providers.
_Avoid_: Provider, account, API

**Provider Item**:
A single linked connection between an Institution and a Provider.
_Avoid_: Institution, Financial Account, merged connection

**Financial Account**:
A bank, credit, loan, investment, property, or manual account tracked by `money`.
_Avoid_: Provider Item, user account, login account

**Manual Account**:
A user-created local Financial Account not linked to a Provider.
_Avoid_: Provider, Import Source, synthetic Provider Item

**Demo Environment**:
An isolated non-persistent `money` runtime with synthetic data that resets to its initial state on each run.
_Avoid_: Demo seed in real store, sample Provider, persisted demo data

**Account Balance**:
The signed current value of a Financial Account as it contributes to local financial position, with assets positive and liabilities negative.
_Avoid_: Credit limit, available credit, provider-native balance

**Transaction Amount**:
A signed money movement where positive means inflow and negative means outflow.
_Avoid_: Plaid amount semantics, provider-native sign, unsigned spend

**Pending Transaction**:
A transaction reported by a Provider before final posting.
_Avoid_: Posted transaction, ignored transaction

**Local Annotation**:
A user-controlled local addition to financial data, such as tags, notes, review state, or other custom fields.
_Avoid_: Provider field, synced source data, AI memory

**Account Alias**:
A local user-defined display name for a Financial Account.
_Avoid_: Provider account name, institution name

**Tag**:
A local annotation label attached to transactions.
_Avoid_: Provider category, Plaid personal finance category

**Provider Category**:
A transaction classification supplied by a Provider.
_Avoid_: Local category, user override

**Local Category**:
A user-controlled transaction classification stored locally by `money`.
_Avoid_: Provider category, raw category

**Review Workflow**:
An optional local annotation workflow that marks transactions for human or agent review.
_Avoid_: Provider status, required workflow, default inbox

**Credit Limit**:
The maximum credit a lender makes available on a credit account.
_Avoid_: Account Balance, cash, available money

**Available Balance**:
Money currently available inside a cash-like Financial Account.
_Avoid_: Available credit, credit limit, borrowing capacity

**Merge Review**:
A user-reviewed operation that compares duplicate-looking Provider Items or Financial Accounts before any deduplication or merge is applied.
_Avoid_: Automatic merge, silent dedupe, fallback matching

**Import Source**:
A file-based or migration data source, such as CSV, Apple Card export, or Monarch export, that brings financial data into `money` without a live Provider sync.
_Avoid_: Provider, API, fallback

**Command Contract**:
A stable machine-readable promise for a `money` CLI command's input, output envelope, field names, sorting, pagination, and errors.
_Avoid_: API contract, terminal formatting, helper endpoint

**Monarch Compatibility**:
Command semantics that preserve the agent-facing habits of `monarchmoney-cli` while using `money` as the local backend.
_Avoid_: Monarch API dependency, exact remote schema clone, donor fallback

**Link**:
A user-initiated operation that connects an Institution through a selected Provider.
_Avoid_: Sync, import, account creation

**Sync**:
A user-initiated or scheduled operation that refreshes linked Provider Items into the local store.
_Avoid_: Link, import, background magic

**Removed Transaction**:
A transaction the Provider reports as deleted or no longer present, retained locally as removed until explicit cleanup.
_Avoid_: Hard delete, invisible deletion

**Warning**:
A non-fatal diagnostic that tells the caller about risky or discouraged configuration while allowing the command to proceed.
_Avoid_: Error, stderr noise in JSON mode

**Encrypted Store**:
The user-owned local SQLite database transparently encrypted at rest with an app-managed local encryption key, using the implementation best suited to `money`'s Go architecture.
_Avoid_: Plain SQLite, best-effort encryption, token-only encryption, password vault

**Local Configuration**:
The user-controlled `money` configuration rooted at `~/.money/config.yaml`, with values completed from `~/.money/.env`, environment variables, or explicit setup commands.
_Avoid_: Ray config, managed account settings, hidden donor fallback, override chain

**Setup**:
An explicit command flow that writes Local Configuration and explains any skipped capabilities.
_Avoid_: Managed onboarding, account registration, hidden defaults

**Secret Reference**:
A configuration value that explicitly names where a secret should be read from using an `env:` reference loaded directly or through `.env`.
_Avoid_: Implicit override, magic env lookup

## Relationships

- **money** may borrow domain boundaries from **OpenRay Finance**, but it owns its own product identity.
- **OpenRay Finance** terms should be translated into **money** terms before becoming command names, config names, or stable contracts.
- A **Provider** connects external financial data to **money**; it is not a user-facing **money** API and is not ranked above another Provider by default.
- Users start **Link** from an **Institution**; if multiple **Providers** support that **Institution**, the user chooses the Provider.
- An **Institution** can have multiple **Provider Items** when it is linked through multiple Providers.
- A **Financial Account** belongs to one **Provider Item** unless a future **Merge Review** explicitly reconciles duplicates.
- A **Manual Account** is local financial data and does not require a Provider or Import Source.
- A **Demo Environment** never writes to the real Encrypted Store and resets when the demo run ends.
- **Account Balance** and **Credit Limit** are separate; credit capacity is not cash and must not be merged into cash balances.
- Credit card and loan debt are negative **Account Balances**.
- **Available Balance** is for cash-like accounts; credit borrowing capacity must use credit-specific fields.
- A **Transaction Amount** follows Monarch-style signed semantics; Provider adapters must translate provider-native signs into this canonical form.
- Read commands include **Pending Transactions** by default and expose pending status explicitly.
- Transaction search defaults to deterministic ordering: newest date first, pending before posted on the same date, then stable ID order.
- **Tags** and other **Local Annotations** live in the local database and are not supplied by Providers.
- **Account Alias** is a **Local Annotation** and must not be overwritten by Sync.
- **Provider Category** and **Local Category** are separate; contracts may expose an effective category with its source.
- **Sync** may update Provider-owned fields but must preserve **Local Annotations**.
- **Removed Transactions** are soft-deleted by Sync and can be purged only by an explicit user-confirmed cleanup.
- **Review Workflow** is optional and disabled by default for agent-first usage.
- **Merge Review** requires user comparison and approval because deduplication can change financial history.
- Read commands show all unmerged **Financial Accounts** and preserve Provider provenance.
- An **Import Source** brings data into **money** without being a networked **Provider**.
- A **Command Contract** is the current external interface for agents, scripts, and cron jobs.
- Canonical transaction commands use Monarch-compatible `transactions`; `tx` is a CLI alias only.
- A **Warning** appears inside the JSON envelope for JSON **Command Contracts** and on stderr for human output.
- **money** uses a BYOK model: users provide financial **Provider** credentials locally and do not need a `money` account, subscription, or AI key.
- **Monarch Compatibility** guides command semantics when it does not conflict with local-first ownership and provider neutrality.
- **money** stores financial data in an **Encrypted Store** and must fail explicitly instead of opening a plaintext database.
- **Local Configuration** provides the database key, database path, and Provider credentials through explicit values and **Secret References** without falling back to Ray or donor config paths.
- **Setup** may skip Provider credentials, but must clearly explain that sync will not work until credentials are added.
- **Sync** defaults to all linked Provider Items and can be narrowed by Provider or Provider Item.

## Example dialogue

> **Dev:** "Should we call the new provider command `openray sync` because the donor docs use OpenRay?"
> **Domain expert:** "No. **OpenRay Finance** is reference language. The project is **money**, so the command should use `money` naming."
>
> **Dev:** "Should Plaid, Finicity, and MX be grouped under APIs?"
> **Domain expert:** "No. They are **Providers**. `money` does not currently plan a local API surface."
>
> **Dev:** "Is a Monarch export a Provider?"
> **Domain expert:** "No. Monarch export is an **Import Source**. Plaid and Bridge are the first **Providers**."
>
> **Dev:** "Should setup ask users to choose Plaid or Bridge first?"
> **Domain expert:** "No. Users choose the **Institution** they want to connect. If multiple **Providers** support it, `money` asks which Provider to use."
>
> **Dev:** "Can scripts still link through a specific Provider?"
> **Domain expert:** "Yes. `money link` is the canonical institution-first flow, while provider-specific link commands remain explicit tools."
>
> **Dev:** "If Chase is linked through two Providers, should `money` merge the accounts automatically?"
> **Domain expert:** "No. Store separate **Provider Items** first. Any later merge requires **Merge Review**."
>
> **Dev:** "Should account listing hide duplicate-looking accounts before merge review?"
> **Domain expert:** "No. It should show all **Financial Accounts** with Provider provenance."
>
> **Dev:** "Plaid treats spending as positive, but Monarch treats it as negative. Which sign should `money` expose?"
> **Domain expert:** "**Transaction Amount** uses Monarch-style signs: income positive, spending negative. Provider adapters normalize their native signs."
>
> **Dev:** "Should transaction search hide pending card charges?"
> **Domain expert:** "No. Include **Pending Transactions** by default and mark them clearly."
>
> **Dev:** "How are transaction search results ordered?"
> **Domain expert:** "By newest date first, pending before posted on the same date, then stable ID order."
>
> **Dev:** "Does Plaid provide transaction tags?"
> **Domain expert:** "No. **Tags** are **Local Annotations** controlled by `money` or imported from a user-owned source."
>
> **Dev:** "How does a user distinguish two similar checking accounts?"
> **Domain expert:** "Provider mask can help, and the user can set an **Account Alias** locally."
>
> **Dev:** "Should demo data be inserted into the user's real database?"
> **Domain expert:** "No. Use a **Demo Environment** that is isolated and non-persistent."
>
> **Dev:** "If Plaid resyncs a transaction, can it erase local notes or tags?"
> **Domain expert:** "No. **Sync** preserves **Local Annotations**."
>
> **Dev:** "If a user recategorizes a Provider transaction, where does that go?"
> **Domain expert:** "Into **Local Category**. **Provider Category** remains available and may be refreshed by Sync."
>
> **Dev:** "Should every newly synced transaction need review?"
> **Domain expert:** "No. **Review Workflow** is optional, asked during setup, and defaults off."
>
> **Dev:** "Should a credit card's available credit be counted like cash?"
> **Domain expert:** "No. **Credit Limit** and available credit are separate from **Account Balance**."
>
> **Dev:** "If a credit card has a $500 balance owed, what is its **Account Balance**?"
> **Domain expert:** "`-500`; debt is a negative contribution to local financial position."
>
> **Dev:** "Can credit card available credit be returned as **Available Balance**?"
> **Domain expert:** "No. **Available Balance** means available cash-like money, not borrowing capacity."
>
> **Dev:** "How should an external agent call `money`?"
> **Domain expert:** "Through **Command Contracts** exposed by the CLI. Cron jobs also call the CLI."
>
> **Dev:** "Should `money tx search` accept an empty query to mean recent transactions?"
> **Domain expert:** "No. Preserve **Monarch Compatibility**: list is for recent/filter-only results; search requires a text query."
>
> **Dev:** "Should contracts use `tx.search` because the CLI has a short alias?"
> **Domain expert:** "No. `tx` is only an alias. The canonical command contract is `transactions.search`."
>
> **Dev:** "Can we start with plaintext SQLite and add encryption later?"
> **Domain expert:** "No. The local database is an **Encrypted Store** from the first provider-backed version."
>
> **Dev:** "Does `money` need a user password to unlock a vault?"
> **Domain expert:** "No. The **Encrypted Store** uses app-managed local encryption like the relevant BYOK path in Ray Finance, not a zero-knowledge password vault."
>
> **Dev:** "Can users configure `money` by editing files instead of an interactive setup?"
> **Domain expert:** "Yes. **Local Configuration** can be changed through config files, `.env`, environment variables, or setup commands."
>
> **Dev:** "Can setup finish without Provider credentials?"
> **Domain expert:** "Yes, but **Setup** must warn that sync is unavailable and show the command for adding credentials later."
>
> **Dev:** "What does `money sync` do without flags?"
> **Domain expert:** "It syncs all linked **Provider Items**. Flags can narrow it to one Provider or Provider Item."
>
> **Dev:** "If a Provider says a transaction was removed, should `money` delete the local row?"
> **Domain expert:** "No. Mark it as a **Removed Transaction**. Hard cleanup requires an explicit confirmed command."
>
> **Dev:** "If `~/.money/config.yaml` points Plaid's secret at `PLAID_SECRET`, is `.env` overriding config?"
> **Domain expert:** "No. That is a **Secret Reference** written with `env: PLAID_SECRET`. The config says where to read the secret; `.env` supplies the value."
>
> **Dev:** "Should direct credentials in `config.yaml` be rejected?"
> **Domain expert:** "No. They can work, but `money` should warn that `env:` references are preferred for secrets."
>
> **Dev:** "Where does that warning go in JSON mode?"
> **Domain expert:** "Inside the **Command Contract** envelope's `warnings` array, so stdout remains one parseable JSON document."

## Flagged ambiguities

- "OpenRay" appeared in implementation plans as if it might still be the target product name. Resolved: **money** is the canonical project identity; **OpenRay Finance** is donor/reference language only.
- "Monarch-compatible" could imply depending on Monarch's API or copying its remote schema. Resolved: **Monarch Compatibility** means preserving useful command semantics while replacing Monarch with a local backend.
- "API" was used ambiguously for financial integrations and a possible local interface. Resolved: Plaid, Finicity, and MX are **Providers**; a local API is not part of the current plan.
- "Data source" could mean a live financial integration or a file/migration path. Resolved: **Provider** is a networked sync source; **Import Source** is file-based or migration input.
- "Provider choice" could imply one default Provider is preferred globally. Resolved: Providers are peers; choice depends on **Institution** support and explicit user selection.
- "Merge" could mean automatic deduplication after linking. Resolved: duplicate-looking Provider data remains separate until a user-approved **Merge Review**.
- "Amount" conflicts across donors and Providers. Resolved: canonical **Transaction Amount** is positive for inflow and negative for outflow, matching Monarch-style agent workflows.
- "Tag" could be confused with Provider category data. Resolved: **Tag** is a **Local Annotation**, separate from Provider categories.
- "Category" could mean Provider-supplied classification or user-controlled classification. Resolved: **Provider Category** and **Local Category** are separate, with effective category exposed intentionally.
- "Interface" previously included a possible local API. Resolved: the current planned external interface is **Command Contract** over CLI only.
- "SQLite encryption" could mean only encrypting provider tokens. Resolved: **Encrypted Store** means the SQLite database itself is encrypted at rest; provider credentials still require careful handling inside that store.
- "Secure local storage" could imply a master-password vault or hosted account. Resolved: `money` is BYOK-only and uses app-managed local encryption, with no registered account, subscription, or required AI key.
- "Config" could imply Ray's `~/.ray`, managed account setup, donor fallback, precedence-based override behavior, cwd `.env`, or magic environment lookup. Resolved: **Local Configuration** is owned by `money`, uses `~/.money/config.yaml`, `~/.money/.env`, environment variables, and setup commands only, and may complete secret values through explicit `env:` **Secret References**. Direct secret values are allowed but should produce a warning recommending `env:`.
