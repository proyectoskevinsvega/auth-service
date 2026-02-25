package domain

import (
	"time"

	"github.com/google/uuid"
)

type BackupCode struct {
	ID        string
	TenantID  string
	UserID    string
	CodeHash  string
	Used      bool
	UsedAt    *time.Time
	CreatedAt time.Time
}

func NewBackupCode(tenantID, userID, codeHash string) *BackupCode {
	return &BackupCode{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		UserID:    userID,
		CodeHash:  codeHash,
		Used:      false,
		CreatedAt: time.Now(),
	}
}
