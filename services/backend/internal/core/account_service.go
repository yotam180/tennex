package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/tennex/backend/internal/repo"
	"github.com/tennex/pkg/events"
)

// AccountService handles account business logic
type AccountService struct {
	accountRepo repo.AccountRepository
	logger      *zap.Logger
}

// NewAccountService creates a new account service
func NewAccountService(accountRepo repo.AccountRepository, logger *zap.Logger) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		logger:      logger.Named("account_service"),
	}
}

// UpsertAccount creates or updates an account
func (s *AccountService) UpsertAccount(ctx context.Context, id, waJid, displayName, avatarUrl, status string, lastSeen *time.Time) (*repo.Account, error) {
	s.logger.Debug("Upserting account",
		zap.String("id", id),
		zap.String("wa_jid", waJid),
		zap.String("display_name", displayName),
		zap.String("status", status))

	var waJidNull sql.NullString
	if waJid != "" {
		waJidNull = sql.NullString{String: waJid, Valid: true}
	}

	var displayNameNull sql.NullString
	if displayName != "" {
		displayNameNull = sql.NullString{String: displayName, Valid: true}
	}

	var avatarUrlNull sql.NullString
	if avatarUrl != "" {
		avatarUrlNull = sql.NullString{String: avatarUrl, Valid: true}
	}

	var lastSeenNull sql.NullTime
	if lastSeen != nil {
		lastSeenNull = sql.NullTime{Time: *lastSeen, Valid: true}
	}

	account, err := s.accountRepo.UpsertAccount(ctx, repo.UpsertAccountParams{
		ID:          id,
		WaJid:       waJidNull,
		DisplayName: displayNameNull,
		AvatarUrl:   avatarUrlNull,
		Status:      status,
		LastSeen:    lastSeenNull,
	})
	if err != nil {
		s.logger.Error("Failed to upsert account", zap.Error(err))
		return nil, fmt.Errorf("failed to upsert account: %w", err)
	}

	s.logger.Info("Account upserted",
		zap.String("id", account.ID),
		zap.String("status", account.Status))

	return &account, nil
}

// GetAccount retrieves an account by ID
func (s *AccountService) GetAccount(ctx context.Context, id string) (*repo.Account, error) {
	account, err := s.accountRepo.GetAccount(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return &account, nil
}

// GetAccountByWAJID retrieves an account by WhatsApp JID
func (s *AccountService) GetAccountByWAJID(ctx context.Context, waJid string) (*repo.Account, error) {
	account, err := s.accountRepo.GetAccountByWAJID(ctx, waJid)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by WA JID: %w", err)
	}
	return &account, nil
}

// UpdateAccountStatus updates the status of an account
func (s *AccountService) UpdateAccountStatus(ctx context.Context, id, status string, lastSeen *time.Time) error {
	s.logger.Debug("Updating account status",
		zap.String("id", id),
		zap.String("status", status))

	var lastSeenNull sql.NullTime
	if lastSeen != nil {
		lastSeenNull = sql.NullTime{Time: *lastSeen, Valid: true}
	}

	err := s.accountRepo.UpdateAccountStatus(ctx, repo.UpdateAccountStatusParams{
		ID:       id,
		Status:   status,
		LastSeen: lastSeenNull,
	})
	if err != nil {
		s.logger.Error("Failed to update account status", zap.Error(err))
		return fmt.Errorf("failed to update account status: %w", err)
	}

	s.logger.Info("Account status updated",
		zap.String("id", id),
		zap.String("status", status))

	return nil
}

// ListAccounts retrieves a list of accounts with pagination
func (s *AccountService) ListAccounts(ctx context.Context, limit, offset int32) ([]repo.Account, error) {
	accounts, err := s.accountRepo.ListAccounts(ctx, repo.ListAccountsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	s.logger.Debug("Listed accounts", zap.Int("count", len(accounts)))
	return accounts, nil
}

// GetConnectedAccounts retrieves all connected accounts
func (s *AccountService) GetConnectedAccounts(ctx context.Context) ([]repo.Account, error) {
	accounts, err := s.accountRepo.GetConnectedAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connected accounts: %w", err)
	}

	s.logger.Debug("Retrieved connected accounts", zap.Int("count", len(accounts)))
	return accounts, nil
}

// SetAccountConnected marks an account as connected
func (s *AccountService) SetAccountConnected(ctx context.Context, id, waJid, displayName string) error {
	now := time.Now()
	_, err := s.UpsertAccount(ctx, id, waJid, displayName, "", events.AccountStatusConnected, &now)
	return err
}

// SetAccountDisconnected marks an account as disconnected
func (s *AccountService) SetAccountDisconnected(ctx context.Context, id string) error {
	return s.UpdateAccountStatus(ctx, id, events.AccountStatusDisconnected, nil)
}

// SetAccountError marks an account as having an error
func (s *AccountService) SetAccountError(ctx context.Context, id string) error {
	return s.UpdateAccountStatus(ctx, id, events.AccountStatusError, nil)
}
