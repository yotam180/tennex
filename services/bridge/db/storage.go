package db

import (
	"context"
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage struct {
	db *gorm.DB
}

func NewStorage() (*Storage, error) {
	db, err := gorm.Open(postgres.Open(GetConnectionString()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&AccountConnection{})
	if err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

// SetAccountConnection stores or updates a connection between an account and an integration
func (storage *Storage) SetAccountConnection(ctx context.Context, accountID, integrationID, identifier string) error {
	connection := AccountConnection{
		AccountID:     accountID,
		IntegrationID: integrationID,
		Identifier:    identifier,
	}

	// Use GORM's Create with conflict resolution (upsert)
	result := storage.db.WithContext(ctx).
		Where("account_id = ? AND integration_id = ?", accountID, integrationID).
		Assign(AccountConnection{Identifier: identifier}).
		FirstOrCreate(&connection)

	return result.Error
}

// GetAccountConnection retrieves a connection for a specific account and integration
func (storage *Storage) GetAccountConnection(ctx context.Context, accountID, integrationID string) (*AccountConnection, error) {
	var connection AccountConnection
	result := storage.db.WithContext(ctx).
		Where("account_id = ? AND integration_id = ?", accountID, integrationID).
		First(&connection)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &connection, result.Error
}

// GetAccountConnections retrieves all connections for a specific account
func (storage *Storage) GetAccountConnections(ctx context.Context, accountID string) ([]AccountConnection, error) {
	var connections []AccountConnection
	result := storage.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Find(&connections)

	return connections, result.Error
}

// DeleteAccountConnection removes a connection between an account and an integration
func (storage *Storage) DeleteAccountConnection(ctx context.Context, accountID, integrationID string) error {
	result := storage.db.WithContext(ctx).
		Where("account_id = ? AND integration_id = ?", accountID, integrationID).
		Delete(&AccountConnection{})

	return result.Error
}

// SetJIDForAccount is a convenience method for setting WhatsApp JID for an account
func (storage *Storage) SetJIDForAccount(ctx context.Context, accountID, jid string) error {
	return storage.SetAccountConnection(ctx, accountID, "whatsapp", jid)
}

// GetJIDForAccount is a convenience method for getting WhatsApp JID for an account
func (storage *Storage) GetJIDForAccount(ctx context.Context, accountID string) (string, error) {
	connection, err := storage.GetAccountConnection(ctx, accountID, "whatsapp")
	if err != nil {
		return "", err
	}
	if connection == nil {
		return "", nil
	}
	return connection.Identifier, nil
}
