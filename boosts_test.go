package soccer_test

import (
	"math/rand"
	"testing"
	"time"

	soccer "github.com/stein-f/oink-soccer-common"
)

func TestGetBoost(t *testing.T) {
	tests := []struct {
		name  string
		boost soccer.Boost
	}{
		{"in range", soccer.Boost{MinBoost: 5, MaxBoost: 10}},
		{"handles min 0", soccer.Boost{MinBoost: 0, MaxBoost: 1}},
		{"handles negative", soccer.Boost{MinBoost: -10, MaxBoost: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.boost.GetBoost(rand.New(rand.NewSource(time.Now().UnixNano())))
			if got < tt.boost.MinBoost || got > tt.boost.MaxBoost {
				t.Errorf("GetBoost() = %v, want range [%v, %v]", got, tt.boost.MinBoost, tt.boost.MaxBoost)
			}
		})
	}
}
