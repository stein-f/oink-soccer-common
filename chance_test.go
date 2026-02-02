package soccer_test

import (
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/lang"
	"github.com/stretchr/testify/assert"
)

func TestDetermineChangeType(t *testing.T) {
	randSource := rand.NewSource(1)

	cases := map[string]struct {
		gotPreviousChanceType *soccer.ChanceType
		gotRandSource         *rand.Rand
		wantChanceType        soccer.ChanceType
	}{
		"selects a change type": {
			gotPreviousChanceType: lang.ToPtr(soccer.ChanceTypeCross),
			gotRandSource:         rand.New(randSource),
			wantChanceType:        soccer.ChanceTypeOpenPlay,
		},
		"selects a different change type if same as previous": {
			gotPreviousChanceType: lang.ToPtr(soccer.ChanceTypeOpenPlay),
			gotRandSource:         rand.New(randSource),
			wantChanceType:        soccer.ChanceTypeLongRange,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			gotChanceType, err := soccer.DetermineChanceType(c.gotPreviousChanceType, c.gotRandSource)

			assert.NoError(t, err)
			assert.Equal(t, c.wantChanceType, gotChanceType)
		})
	}
}
