package fifacsv

// Player datasets, relative to the repo root. Both use the sofifa FIFA 22
// column layout (ConvertFC27 rewrites EA's FC 27 export into it), so
// LoadCandidates and the lost-pigs importer read either unchanged.
const (
	FIFA22Dataset = "cmd/allocation/fifa_players_22.csv"
	FC27Dataset   = "cmd/allocation/fifa_players_fc27.csv"
)

// FC27FirstSeason is the first season allocated from the FC 27 dataset.
//
// Every season is pinned to the dataset it was drawn from. The candidate pool
// is an input to the block-seeded draw, so swapping the file under a past
// season would change its re-verifiable output. lost-pigs mirrors this rule
// in domain.FifaDatasetKey; the two must move together.
const FC27FirstSeason = 19

// DatasetPath returns the dataset the given season is allocated from,
// relative to the repo root.
func DatasetPath(season int) string {
	if season >= FC27FirstSeason {
		return FC27Dataset
	}
	return FIFA22Dataset
}
