package auth

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type GORMUserStore struct {
	db *gorm.DB
}

func NewGORMUserStore(db *gorm.DB) *GORMUserStore {
	return &GORMUserStore{db: db}
}

func (s *GORMUserStore) Exists(ctx context.Context, userID string) (bool, error) {
	var count int64
	result := s.db.WithContext(ctx).
		Table("user").
		Where("id = ?", userID).
		Count(&count)
	if result.Error != nil {
		return false, fmt.Errorf("check user existence: %w", result.Error)
	}

	return count > 0, nil
}
