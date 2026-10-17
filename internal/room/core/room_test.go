package core

import (
	"errors"
	"testing"
)

func TestRoomCanJoin(t *testing.T) {
	for _, count := range []int64{0, 1, 2, 3} {
		err := (Room{MemberCount: count}).CanJoin()
		if count >= 2 {
			if !errors.Is(err, ErrRoomFull) {
				t.Errorf("count %d: error = %v", count, err)
			}
		} else if err != nil {
			t.Errorf("count %d: error = %v", count, err)
		}
	}
}
