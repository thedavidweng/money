package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/thedavidweng/money/internal/core"
)

type LinkedProviderItem struct {
	Institution LinkedInstitution
	Item        LinkedItem
}

type LinkedInstitution struct {
	ID                    string
	Name                  string
	Provider              string
	ProviderInstitutionID string
}

type LinkedItem struct {
	ID                     string
	Provider               string
	InstitutionID          string
	ProviderExternalItemID string
	Alias                  string
	AccessToken            string
	EncryptedAccessToken   []byte
	Status                 string
	Products               []string
	TransactionCursor      string
	ExternalUserID         string
}

type ProviderItemQuery struct {
	Provider       string
	ProviderItemID string
}

func (s *SQLiteStore) StoreLinkedProviderItem(ctx context.Context, linked LinkedProviderItem) error {
	now := time.Now().UTC().Format(time.RFC3339)
	products, err := json.Marshal(linked.Item.Products)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO institutions (id, name, provider, provider_institution_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, provider_institution_id) DO UPDATE SET
  name = excluded.name,
  updated_at = excluded.updated_at`,
		linked.Institution.ID, linked.Institution.Name, linked.Institution.Provider, linked.Institution.ProviderInstitutionID, now, now)
	if err != nil {
		tx.Rollback()
		return err
	}

	token := linked.Item.EncryptedAccessToken
	if len(token) == 0 {
		token = []byte(linked.Item.AccessToken)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO provider_items (
  id, provider, institution_id, provider_external_item_id, encrypted_access_token,
  external_user_id, status, products_json, transaction_cursor, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?)
ON CONFLICT(provider, provider_external_item_id) DO UPDATE SET
  institution_id = excluded.institution_id,
  encrypted_access_token = excluded.encrypted_access_token,
  external_user_id = excluded.external_user_id,
  status = excluded.status,
  products_json = excluded.products_json,
  transaction_cursor = excluded.transaction_cursor,
  updated_at = excluded.updated_at`,
		linked.Item.ID, linked.Item.Provider, linked.Item.InstitutionID, linked.Item.ProviderExternalItemID, token,
		linked.Item.ExternalUserID, linked.Item.Status, string(products), linked.Item.TransactionCursor, now, now)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListProviderItems(ctx context.Context, query ProviderItemQuery) ([]LinkedItem, error) {
	sqlText := `
SELECT id, provider, institution_id, provider_external_item_id, COALESCE(alias, ''),
       encrypted_access_token, external_user_id, status, products_json, transaction_cursor
FROM provider_items
WHERE (? = '' OR provider = ?)
  AND (? = '' OR id = ?)
ORDER BY provider, id`
	rows, err := s.db.QueryContext(ctx, sqlText, query.Provider, query.Provider, query.ProviderItemID, query.ProviderItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []LinkedItem
	for rows.Next() {
		var item LinkedItem
		var externalUserID, cursor sql.NullString
		var productsJSON string
		if err := rows.Scan(
			&item.ID, &item.Provider, &item.InstitutionID, &item.ProviderExternalItemID, &item.Alias,
			&item.EncryptedAccessToken, &externalUserID, &item.Status, &productsJSON, &cursor,
		); err != nil {
			return nil, err
		}
		item.ExternalUserID = externalUserID.String
		item.TransactionCursor = cursor.String
		if err := json.Unmarshal([]byte(productsJSON), &item.Products); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLiteStore) GetProviderItem(ctx context.Context, id string) (LinkedItem, error) {
	var item LinkedItem
	var externalUserID, cursor sql.NullString
	var productsJSON string
	err := s.db.QueryRowContext(ctx, `
SELECT id, provider, institution_id, provider_external_item_id, COALESCE(alias, ''),
       encrypted_access_token, external_user_id, status, products_json, transaction_cursor
FROM provider_items
WHERE id = ?`, id).Scan(
		&item.ID, &item.Provider, &item.InstitutionID, &item.ProviderExternalItemID, &item.Alias,
		&item.EncryptedAccessToken, &externalUserID, &item.Status, &productsJSON, &cursor,
	)
	if err != nil {
		return LinkedItem{}, err
	}
	item.ExternalUserID = externalUserID.String
	item.TransactionCursor = cursor.String
	if err := json.Unmarshal([]byte(productsJSON), &item.Products); err != nil {
		return LinkedItem{}, err
	}
	return item, nil
}

func (s *SQLiteStore) UpsertInstitution(ctx context.Context, institution core.Institution) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO institutions (id, name, provider, provider_institution_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, provider_institution_id) DO UPDATE SET
  name = excluded.name,
  updated_at = excluded.updated_at`,
		institution.ID, institution.Name, institution.Provider, institution.ProviderInstitutionID, now, now)
	return err
}

func (s *SQLiteStore) UpdateProviderItemName(ctx context.Context, id string, name string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE provider_items
SET alias = ?, updated_at = ?
WHERE id = ?`, name, now, id)
	return err
}

func (s *SQLiteStore) RemoveProviderItem(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Cascade-delete in dependency order:
	// transaction_tags → transactions → recurring → sync_runs → accounts → provider_items
	// holdings and liabilities have ON DELETE CASCADE in schema.
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM transaction_tags WHERE transaction_id IN (
			SELECT id FROM transactions WHERE provider_item_id = ?
		)`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM transactions WHERE provider_item_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recurring WHERE provider_item_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_runs WHERE provider_item_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM accounts WHERE provider_item_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM provider_items WHERE id = ?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteStore) UpsertProviderItem(ctx context.Context, item core.ProviderItem) error {
	now := time.Now().UTC().Format(time.RFC3339)
	products, err := json.Marshal(item.Products)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO provider_items (
  id, provider, institution_id, provider_external_item_id, encrypted_access_token,
  external_user_id, status, products_json, transaction_cursor, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?)
ON CONFLICT(provider, provider_external_item_id) DO UPDATE SET
  institution_id = excluded.institution_id,
  encrypted_access_token = excluded.encrypted_access_token,
  external_user_id = excluded.external_user_id,
  status = excluded.status,
  products_json = excluded.products_json,
  transaction_cursor = excluded.transaction_cursor,
  updated_at = excluded.updated_at`,
		item.ID, item.Provider, item.InstitutionID, item.ProviderExternalItemID, item.EncryptedAccessToken,
		item.ExternalUserID, item.Status, string(products), item.TransactionCursor, now, now)
	return err
}
