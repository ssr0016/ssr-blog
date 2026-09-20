package repository

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPageOffset(t *testing.T) {
	tests := []struct {
		name        string
		page, limit int
		want        uint64
		ok          bool
	}{
		{"first page", 1, 20, 0, true},
		{"third page", 3, 20, 40, true},
		{"largest fitting page", math.MaxInt64/20 + 1, 20, uint64(math.MaxInt64/20) * 20, true},
		{"one past largest fitting page", math.MaxInt64/20 + 2, 20, 0, false},
		{"max int page", math.MaxInt64, 20, 0, false},
		{"max int page limit 1 still fits", math.MaxInt64, 1, math.MaxInt64 - 1, true},
		{"page below 1", 0, 20, 0, false},
		{"limit below 1", 1, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pageOffset(tt.page, tt.limit)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
