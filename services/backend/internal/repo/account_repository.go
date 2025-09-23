package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type accountRepository struct {
	db *pgxpool.Pool
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db *pgxpool.Pool) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) UpsertAccount(ctx context.Context, params UpsertAccountParams) (Account, error) {
	query := `
		INSERT INTO accounts (id, wa_jid, display_name, avatar_url, status, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			wa_jid = EXCLUDED.wa_jid,
			display_name = EXCLUDED.display_name,
			avatar_url = EXCLUDED.avatar_url,
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen,
			updated_at = NOW()
		RETURNING id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at`

	var account Account
	err := r.db.QueryRow(ctx, query,
		params.ID,
		params.WaJid,
		params.DisplayName,
		params.AvatarUrl,
		params.Status,
		params.LastSeen,
	).Scan(
		&account.ID,
		&account.WaJid,
		&account.DisplayName,
		&account.AvatarUrl,
		&account.Status,
		&account.LastSeen,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return Account{}, fmt.Errorf("failed to upsert account: %w", err)
	}

	return account, nil
}

func (r *accountRepository) GetAccount(ctx context.Context, id string) (Account, error) {
	query := `
		SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
		FROM accounts 
		WHERE id = $1`

	var account Account
	err := r.db.QueryRow(ctx, query, id).Scan(
		&account.ID,
		&account.WaJid,
		&account.DisplayName,
		&account.AvatarUrl,
		&account.Status,
		&account.LastSeen,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return Account{}, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

func (r *accountRepository) GetAccountByWAJID(ctx context.Context, waJid string) (Account, error) {
	query := `
		SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
		FROM accounts 
		WHERE wa_jid = $1`

	var account Account
	err := r.db.QueryRow(ctx, query, waJid).Scan(
		&account.ID,
		&account.WaJid,
		&account.DisplayName,
		&account.AvatarUrl,
		&account.Status,
		&account.LastSeen,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return Account{}, fmt.Errorf("failed to get account by WA JID: %w", err)
	}

	return account, nil
}

func (r *accountRepository) UpdateAccountStatus(ctx context.Context, params UpdateAccountStatusParams) error {
	query := `
		UPDATE accounts 
		SET status = $2, last_seen = $3, updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query,
		params.ID,
		params.Status,
		params.LastSeen,
	)
	if err != nil {
		return fmt.Errorf("failed to update account status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("account not found: %s", params.ID)
	}

	return nil
}

func (r *accountRepository) ListAccounts(ctx context.Context, params ListAccountsParams) ([]Account, error) {
	query := `
		SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
		FROM accounts 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var account Account
		err := rows.Scan(
			&account.ID,
			&account.WaJid,
			&account.DisplayName,
			&account.AvatarUrl,
			&account.Status,
			&account.LastSeen,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return accounts, nil
}

func (r *accountRepository) GetConnectedAccounts(ctx context.Context) ([]Account, error) {
	query := `
		SELECT id, wa_jid, display_name, avatar_url, status, last_seen, created_at, updated_at
		FROM accounts 
		WHERE status = 'connected'
		ORDER BY last_seen DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var account Account
		err := rows.Scan(
			&account.ID,
			&account.WaJid,
			&account.DisplayName,
			&account.AvatarUrl,
			&account.Status,
			&account.LastSeen,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return accounts, nil
}
