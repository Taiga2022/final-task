package core

import (
	"context"
	"errors"
	"unicode/utf8"
)

var ErrInvalidPage = errors.New("invalid pagination")

type Service struct{ reader Reader }

func NewService(reader Reader) *Service { return &Service{reader: reader} }

func (s *Service) List(ctx context.Context, roomID, actorID, targetID, after string, limit int) (Page, error) {
	if limit < 1 || limit > 100 || !utf8.ValidString(after) || utf8.RuneCountInString(after) > 64 {
		return Page{}, ErrInvalidPage
	}
	items, err := s.reader.Read(ctx, roomID, actorID, targetID, after, limit+1)
	if err != nil {
		return Page{}, err
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	result := Page{Items: items, HasMore: more}
	if more {
		result.LastID = items[len(items)-1].ID
	}
	return result, nil
}
