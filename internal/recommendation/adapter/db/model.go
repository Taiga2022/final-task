package db

type roomModel struct {
	ID          string
	OwnerUserID string
	Weight      float64
}

func (roomModel) TableName() string { return "rooms" }

type seriesModel struct {
	ID      string
	Title   string
	GenreID string
}
