package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	historycore "github.com/Taiga2022/final-task/internal/history/core"
)

type recordingHistoryWriter struct {
	histories []historycore.ViewingHistory
	err       error
}

func (writer *recordingHistoryWriter) UpsertHistories(histories []historycore.ViewingHistory) error {
	writer.histories = append(writer.histories, histories...)
	return writer.err
}

func TestImportCSV(t *testing.T) {
	t.Parallel()

	const source = `watch_flag,item_target_program_id,first_pageview_time,item_type_cat,user_id
1,episode-1,1651363200,episode,user-1
0,episode-2,1651363201,episode,user-2
1,series-1,1651363202,series,user-3
`

	writer := &recordingHistoryWriter{}
	if err := importCSV(writer, strings.NewReader(source)); err != nil {
		t.Fatalf("importCSV() error = %v", err)
	}

	want := []historycore.ViewingHistory{{
		UserID:       "user-1",
		EpisodeID:    "episode-1",
		LastViewedAt: time.Unix(1651363200, 0),
	}}
	if len(writer.histories) != len(want) || writer.histories[0] != want[0] {
		t.Fatalf("written histories = %#v, want %#v", writer.histories, want)
	}
}

func TestImportCSVRejectsMissingColumn(t *testing.T) {
	t.Parallel()

	writer := &recordingHistoryWriter{}
	err := importCSV(writer, strings.NewReader("user_id,watch_flag\nuser-1,1\n"))
	if err == nil || !strings.Contains(err.Error(), `required CSV column "first_pageview_time" is missing`) {
		t.Fatalf("importCSV() error = %v, want missing-column error", err)
	}
}

func TestImportCSVRejectsDuplicateColumn(t *testing.T) {
	t.Parallel()

	writer := &recordingHistoryWriter{}
	const source = `first_pageview_time,user_id,item_type_cat,watch_flag,item_target_program_id,user_id
1651363200,user-1,episode,1,episode-1,user-2
`

	err := importCSV(writer, strings.NewReader(source))
	if err == nil || !strings.Contains(err.Error(), `CSV column "user_id" is duplicated`) {
		t.Fatalf("importCSV() error = %v, want duplicate-column error", err)
	}
}

func TestImportCSVPropagatesWriterError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("database unavailable")
	writer := &recordingHistoryWriter{err: wantErr}
	const source = `first_pageview_time,user_id,item_type_cat,watch_flag,item_target_program_id
1651363200,user-1,episode,1,episode-1
`

	err := importCSV(writer, strings.NewReader(source))
	if !errors.Is(err, wantErr) {
		t.Fatalf("importCSV() error = %v, want error wrapping %v", err, wantErr)
	}
}
