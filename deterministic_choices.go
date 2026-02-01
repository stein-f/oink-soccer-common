package soccer

import (
	"sort"

	"github.com/mroth/weightedrand"
)

// BuildChoicesFromMapNumberKeys generalizes building deterministic choices from maps with
// string-like keys and integer-like weights (int/uint variants). Keys are sorted lexicographically.
func BuildChoicesFromMapNumberKeys[K ~string, V ~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64](m map[K]V) []weightedrand.Choice {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	choices := make([]weightedrand.Choice, 0, len(keys))
	for _, ks := range keys {
		k := K(ks)
		choices = append(choices, weightedrand.Choice{Item: k, Weight: uint(m[k])})
	}
	return choices
}
