package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	historyadapter "github.com/Taiga2022/final-task/internal/history/adapter"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	csvPath   = "seed/data/agt_train_dataset_220501_220531_processed.csv"
	batchSize = 1000
)

var requiredColumns = [...]string{
	"first_pageview_time",
	"user_id",
	"item_type_cat",
	"watch_flag",
	"item_target_program_id",
}

func run(db *gorm.DB) error {
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open CSV file: %w", err)
	}
	defer file.Close()

	if err := importCSV(db, file); err != nil {
		return fmt.Errorf("import viewing histories: %w", err)
	}

	return nil
}

func importCSV(db *gorm.DB, source io.Reader) error {
	reader := csv.NewReader(source)

	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("CSV file is empty")
		}
		return fmt.Errorf("read CSV header: %w", err)
	}

	columns, err := columnIndexes(header)
	if err != nil {
		return err
	}

	histories := make([]historyadapter.HistoryModel, 0, batchSize)
	for line := 2; ; line++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read CSV line %d: %w", line, err)
		}

		if record[columns["watch_flag"]] != "1" || record[columns["item_type_cat"]] != "episode" {
			continue
		}

		history, err := historyFromRecord(record, columns)
		if err != nil {
			return fmt.Errorf("parse CSV line %d: %w", line, err)
		}

		histories = append(histories, history)
		if len(histories) == batchSize {
			if err := insertHistories(db, histories); err != nil {
				return fmt.Errorf("insert batch ending at CSV line %d: %w", line, err)
			}
			histories = histories[:0]
		}
	}

	if len(histories) > 0 {
		if err := insertHistories(db, histories); err != nil {
			return fmt.Errorf("insert final batch: %w", err)
		}
	}

	return nil
}

func columnIndexes(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	for index, name := range header {
		columns[name] = index
	}

	for _, name := range requiredColumns {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("required CSV column %q is missing", name)
		}
	}

	return columns, nil
}

func historyFromRecord(record []string, columns map[string]int) (historyadapter.HistoryModel, error) {
	userID := record[columns["user_id"]]
	if userID == "" {
		return historyadapter.HistoryModel{}, errors.New("user_id is empty")
	}

	episodeID := record[columns["item_target_program_id"]]
	if episodeID == "" {
		return historyadapter.HistoryModel{}, errors.New("item_target_program_id is empty")
	}

	unixSeconds, err := strconv.ParseInt(record[columns["first_pageview_time"]], 10, 64)
	if err != nil {
		return historyadapter.HistoryModel{}, fmt.Errorf("parse first_pageview_time: %w", err)
	}

	return historyadapter.HistoryModel{
		UserID:       userID,
		EpisodeID:    episodeID,
		LastViewedAt: time.Unix(unixSeconds, 0),
	}, nil
}

func insertHistories(db *gorm.DB, histories []historyadapter.HistoryModel) error {
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "episode_id"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"last_viewed_at": gorm.Expr("GREATEST(last_viewed_at, VALUES(last_viewed_at))"),
		}),
	}).Create(&histories)
	if result.Error != nil {
		return fmt.Errorf("upsert viewing histories: %w", result.Error)
	}

	return nil
}
