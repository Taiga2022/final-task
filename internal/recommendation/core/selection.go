package core

import (
	"slices"
	"strings"
	"unicode/utf8"
)

type Selection struct{ ids []string }

func NewSelection(ids []string) (Selection, error) {
	if ids == nil {
		return Selection{}, ErrInvalidSelectionInput
	}
	copyIDs := slices.Clone(ids)
	for _, id := range copyIDs {
		if strings.TrimSpace(id) == "" || !utf8.ValidString(id) || utf8.RuneCountInString(id) > 64 {
			return Selection{}, ErrInvalidSelectionInput
		}
	}
	slices.Sort(copyIDs)
	if len(slices.Compact(slices.Clone(copyIDs))) != len(copyIDs) {
		return Selection{}, ErrInvalidSelectionInput
	}
	return Selection{ids: copyIDs}, nil
}
func (s Selection) IDs() []string { return slices.Clone(s.ids) }
