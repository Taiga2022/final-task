package core

import "math"

type Weight struct{ value float64 }

func NewWeight(value float64) (Weight, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
		return Weight{}, ErrInvalidWeight
	}
	// Match rooms.weight DECIMAL(5,4), including the in-memory snapshot.
	return Weight{value: math.Round(value*10000) / 10000}, nil
}
func (w Weight) Value() float64 { return w.value }
