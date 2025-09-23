package db

import "time"

// AccountConnection represents a connection between an account and an integration
type AccountConnection struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	AccountID     string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_account_integration" json:"account_id"`
	IntegrationID string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_account_integration" json:"integration_id"`
	Identifier    string    `gorm:"type:text;not null" json:"identifier"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName sets the table name for the AccountConnection model
func (AccountConnection) TableName() string {
	return "account_connections"
}
