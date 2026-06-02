package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SeasonConfig mirrors one entry in cmd/allocation/config.json.
type SeasonConfig struct {
	Season     int    `json:"season"`
	Collab     string `json:"collab"`
	AssetsPath string `json:"assets_path"`
	Round      uint64 `json:"round"`
}

type allocationConfig struct {
	CurrentSeason int            `json:"current_season"`
	Seasons       []SeasonConfig `json:"seasons"`
}

// loadCurrentSeason reads cmd/allocation/config.json under dataRoot and returns
// the entry matching its current_season. The config file is shared with the v1
// runner so a single edit drives both pipelines.
func loadCurrentSeason(dataRoot string) (SeasonConfig, error) {
	raw, err := os.ReadFile(filepath.Join(dataRoot, "cmd/allocation/config.json"))
	if err != nil {
		return SeasonConfig{}, err
	}
	var cfg allocationConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return SeasonConfig{}, err
	}
	for _, s := range cfg.Seasons {
		if s.Season == cfg.CurrentSeason {
			return s, nil
		}
	}
	return SeasonConfig{}, fmt.Errorf("config.json has no entry for current_season %d", cfg.CurrentSeason)
}

// findDataRoot walks up from start until it finds the directory holding
// cmd/allocation/config.json. This lets the tool run from the repo root or from
// the v2/ module directory (the natural cwd for `go run ./cmd/allocation`)
// without anyone having to pass -root.
func findDataRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "cmd/allocation/config.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate cmd/allocation/config.json above %s", start)
		}
		dir = parent
	}
}
