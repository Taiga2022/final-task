package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/Taiga2022/final-task/internal/room/core"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, roomID, userID string, roomName core.RoomName, passwordHash core.PasswordHash) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		room := roomModel{
			ID:           roomID,
			RoomName:     roomName.String(),
			PasswordHash: passwordHash.String(),
			OwnerUserID:  userID,
		}
		if err := tx.Create(&room).Error; err != nil {
			return err
		}
		return tx.Create(&roomMemberModel{RoomID: roomID, UserID: userID}).Error
	})
	if isDuplicate(err) {
		return core.ErrRoomNameAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	return nil
}

func (r *Repository) FindByName(ctx context.Context, roomName core.RoomName) (core.Room, error) {
	var room roomModel
	err := r.db.WithContext(ctx).Where("room_name = ?", roomName.String()).First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return core.Room{}, core.ErrRoomNotFound
	}
	if err != nil {
		return core.Room{}, fmt.Errorf("find room by name: %w", err)
	}
	passwordHash, err := core.NewPasswordHash(room.PasswordHash)
	if err != nil {
		return core.Room{}, fmt.Errorf("restore room password hash: %w", err)
	}
	return core.Room{ID: room.ID, PasswordHash: passwordHash}, nil
}

func (r *Repository) Join(ctx context.Context, roomID, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var room roomModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", roomID).First(&room).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return core.ErrRoomNotFound
			}
			return err
		}

		var existing int64
		if err := tx.Model(&roomMemberModel{}).Where("room_id = ? AND user_id = ?", roomID, userID).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			if err := tx.Model(&roomMemberModel{}).Where("room_id = ?", roomID).Count(&count).Error; err != nil {
				return err
			}
			if err := (core.Room{MemberCount: count}).CanJoin(); err != nil {
				return err
			}
			if err := tx.Create(&roomMemberModel{RoomID: roomID, UserID: userID}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&roomMemberModel{}).Where("room_id = ?", roomID).Count(&count).Error
	})
	if err != nil {
		if errors.Is(err, core.ErrRoomNotFound) || errors.Is(err, core.ErrRoomFull) {
			return 0, err
		}
		return 0, fmt.Errorf("join room: %w", err)
	}
	return count, nil
}

func (r *Repository) Leave(ctx context.Context, roomID, userID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var room roomModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", roomID).First(&room).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return core.ErrRoomNotFound
			}
			return fmt.Errorf("check room: %w", err)
		}
		result := tx.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&roomMemberModel{})
		if result.Error != nil {
			return fmt.Errorf("leave room: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return core.ErrRoomNotFound
		}
		var remaining []roomMemberModel
		if err := tx.Where("room_id = ?", roomID).Order("user_id").Find(&remaining).Error; err != nil {
			return fmt.Errorf("find remaining members: %w", err)
		}
		if len(remaining) == 0 {
			if err := tx.Where("id = ?", roomID).Delete(&roomModel{}).Error; err != nil {
				return fmt.Errorf("delete empty room: %w", err)
			}
		} else if room.OwnerUserID == userID {
			if err := tx.Model(&roomModel{}).Where("id = ?", roomID).
				Updates(map[string]any{"owner_user_id": remaining[0].UserID, "weight": 0.5}).Error; err != nil {
				return fmt.Errorf("transfer room ownership: %w", err)
			}
		}
		return nil
	})
}
