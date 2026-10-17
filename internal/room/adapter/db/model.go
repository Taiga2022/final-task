package db

type roomModel struct {
	ID           string `gorm:"column:id"`
	RoomName     string `gorm:"column:room_name"`
	PasswordHash string `gorm:"column:password_hash"`
	OwnerUserID  string `gorm:"column:owner_user_id"`
}

func (roomModel) TableName() string { return "rooms" }

type roomMemberModel struct {
	RoomID string `gorm:"column:room_id"`
	UserID string `gorm:"column:user_id"`
}

func (roomMemberModel) TableName() string { return "room_members" }
