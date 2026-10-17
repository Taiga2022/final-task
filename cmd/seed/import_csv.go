package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	historycore "github.com/Taiga2022/final-task/internal/history/core"
)

const (
	csvPath   = "seed/data/agt_train_dataset_220501_220531_processed.csv"
	batchSize = 1000
)

type historyBatchWriter interface {
	UpsertHistories([]historycore.ViewingHistory) error
}

type csvColumns struct {
	firstPageviewTime int
	userID            int
	itemType          int
	watchFlag         int
	targetProgramID   int
}

func run(writer historyBatchWriter) (returnErr error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close CSV file: %w", err))
		}
	}()

	if err := importCSV(writer, file); err != nil {
		return fmt.Errorf("import viewing histories: %w", err)
	}

	return nil
}

func importCSV(writer historyBatchWriter, source io.Reader) error {
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

	histories := make([]historycore.ViewingHistory, 0, batchSize)
	for line := 2; ; line++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read CSV line %d: %w", line, err)
		}

		if record[columns.watchFlag] != "1" || record[columns.itemType] != "episode" {
			continue
		}

		history, err := historyFromRecord(record, columns)
		if err != nil {
			return fmt.Errorf("parse CSV line %d: %w", line, err)
		}

		histories = append(histories, history)
		if len(histories) == batchSize {
			if err := writer.UpsertHistories(histories); err != nil {
				return fmt.Errorf("insert batch ending at CSV line %d: %w", line, err)
			}
			histories = histories[:0]
		}
	}

	if len(histories) > 0 {
		if err := writer.UpsertHistories(histories); err != nil {
			return fmt.Errorf("insert final batch: %w", err)
		}
	}

	return nil
}

func columnIndexes(header []string) (csvColumns, error) {
	indexes := make(map[string]int, len(header))
	for index, name := range header {
		if _, exists := indexes[name]; exists {
			return csvColumns{}, fmt.Errorf("CSV column %q is duplicated", name)
		}
		indexes[name] = index
	}

	required := [...]string{
		"first_pageview_time",
		"user_id",
		"item_type_cat",
		"watch_flag",
		"item_target_program_id",
	}
	for _, name := range required {
		if _, ok := indexes[name]; !ok {
			return csvColumns{}, fmt.Errorf("required CSV column %q is missing", name)
		}
	}

	return csvColumns{
		firstPageviewTime: indexes["first_pageview_time"],
		userID:            indexes["user_id"],
		itemType:          indexes["item_type_cat"],
		watchFlag:         indexes["watch_flag"],
		targetProgramID:   indexes["item_target_program_id"],
	}, nil
}

func historyFromRecord(record []string, columns csvColumns) (historycore.ViewingHistory, error) {
	userID := record[columns.userID]
	if userID == "" {
		return historycore.ViewingHistory{}, errors.New("user_id is empty")
	}

	episodeID := record[columns.targetProgramID]
	if episodeID == "" {
		return historycore.ViewingHistory{}, errors.New("item_target_program_id is empty")
	}

	unixSeconds, err := strconv.ParseInt(record[columns.firstPageviewTime], 10, 64)
	if err != nil {
		return historycore.ViewingHistory{}, fmt.Errorf("parse first_pageview_time: %w", err)
	}

	return historycore.ViewingHistory{
		UserID:       userID,
		EpisodeID:    episodeID,
		LastViewedAt: time.Unix(unixSeconds, 0),
	}, nil
}
