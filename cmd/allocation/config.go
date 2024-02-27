package allocation

import (
	_ "embed"
	"encoding/json"
	"math/rand"

	"github.com/hashicorp/go-retryablehttp"
	soccer "github.com/stein-f/oink-soccer-common"
)

//go:embed config.json
var configJSON []byte

type Context struct {
	FifaPlayerRepository    FifaPlayerRepository
	EligibleAssetRepository SoccerEligibleAssetsRepository
	Config                  SeasonConfig
	Rand                    *rand.Rand
}

type config struct {
	CurrentSeason int            `json:"current_season"`
	Seasons       []SeasonConfig `json:"seasons"`
}

type SeasonConfig struct {
	Season     int    `json:"season"`
	Collab     string `json:"collab"`
	AssetsPath string `json:"assets_path"`
	Round      uint64 `json:"round"`
}

func GetContext() (Context, error) {
	var cfg config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return Context{}, err
	}
	var seasonCfg SeasonConfig
	for _, s := range cfg.Seasons {
		if s.Season == cfg.CurrentSeason {
			seasonCfg = s
			break
		}
	}

	httpCli := retryablehttp.NewClient().StandardClient()
	randSource := soccer.CreateRandomSourceFromAlgorandBlockHash(httpCli, seasonCfg.Round)

	return Context{
		FifaPlayerRepository:    FifaPlayersRepository{RandSource: randSource},
		EligibleAssetRepository: EligibleAssetsRepository{},
		Config:                  seasonCfg,
		Rand:                    randSource,
	}, nil
}
