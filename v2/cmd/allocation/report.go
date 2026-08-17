package main

import (
	"fmt"
	"sort"
	"strings"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
)

// printReport writes the post-allocation report to stdout: the tier × level
// table with per-tier percentages, plus the integrity checks that would
// otherwise be run by hand after every allocation (rehearsal, dress or live).
// Eyeball it every run — this is where a wrong weights table or a silently
// missing collection shows up as a shifted column.
func printReport(assignments []allocation.Assignment) {
	tierOrder := []allocation.AssetTier{
		allocation.AssetTierS,
		allocation.AssetTierA,
		allocation.AssetTierB,
		allocation.AssetTierC,
		allocation.AssetTierSpecialist,
		allocation.AssetTierAggressive,
	}
	levelOrder := []soccer.PlayerLevel{
		soccer.PlayerLevelLegendary,
		soccer.PlayerLevelWorldClass,
		soccer.PlayerLevelProfessional,
		soccer.PlayerLevelSemiProfessional,
		soccer.PlayerLevelAmateur,
	}

	type tierStats struct {
		n       int
		byLevel map[soccer.PlayerLevel]int
		elite   int // real (non-legend-card) players rated 87+
		maxOvr  int
	}
	stats := map[allocation.AssetTier]*tierStats{}
	get := func(t allocation.AssetTier) *tierStats {
		if stats[t] == nil {
			stats[t] = &tierStats{byLevel: map[soccer.PlayerLevel]int{}}
		}
		return stats[t]
	}

	assetSeen := map[string]int{}
	legendSeen := map[string]int{}
	for _, a := range assignments {
		s := get(a.Asset.Tier)
		s.n++
		lvl := a.Player.Attributes.PlayerLevel
		s.byLevel[lvl]++
		assetSeen[a.Asset.ID]++
		isLegendCard := lvl == soccer.PlayerLevelLegendary
		if isLegendCard {
			legendSeen[a.Player.ID]++
		}
		if ovr := a.Player.Attributes.OverallRating; ovr > s.maxOvr {
			s.maxOvr = ovr
		}
		if !isLegendCard && a.Player.Attributes.OverallRating >= 87 {
			s.elite++
		}
	}

	// Any tier outside the known order still gets a row (a typo'd tier in the
	// eligible-assets CSV should be loud, not invisible).
	for t := range stats {
		known := false
		for _, k := range tierOrder {
			if t == k {
				known = true
				break
			}
		}
		if !known {
			tierOrder = append(tierOrder, t)
		}
	}

	var b strings.Builder
	b.WriteString("\n=== Post-allocation report ===\n\n")
	fmt.Fprintf(&b, "%-18s %7s", "tier", "n")
	for _, l := range levelOrder {
		fmt.Fprintf(&b, " %22s", l)
	}
	fmt.Fprintf(&b, " %10s %8s\n", "87+ reals", "max ovr")
	for _, t := range tierOrder {
		s := stats[t]
		if s == nil {
			continue
		}
		fmt.Fprintf(&b, "%-18s %7d", t, s.n)
		for _, l := range levelOrder {
			c := s.byLevel[l]
			fmt.Fprintf(&b, " %12d (%6.2f%%)", c, float64(c)/float64(s.n)*100)
		}
		fmt.Fprintf(&b, " %10d %8d\n", s.elite, s.maxOvr)
	}

	// Integrity checks.
	dupAssets, dupLegends := 0, 0
	for _, c := range assetSeen {
		if c > 1 {
			dupAssets++
		}
	}
	for _, c := range legendSeen {
		if c > 1 {
			dupLegends++
		}
	}
	b.WriteString("\nchecks:\n")
	fmt.Fprintf(&b, "  assets assigned:        %d (%d unique", len(assignments), len(assetSeen))
	if dupAssets > 0 {
		fmt.Fprintf(&b, " — %d DUPLICATE ASSET IDS", dupAssets)
	}
	b.WriteString(")\n")
	legendIDs := make([]string, 0, len(legendSeen))
	for id := range legendSeen {
		legendIDs = append(legendIDs, id)
	}
	sort.Strings(legendIDs)
	fmt.Fprintf(&b, "  legend cards seeded:    %d", len(legendSeen))
	if dupLegends > 0 {
		fmt.Fprintf(&b, " — %d SEEDED MORE THAN ONCE", dupLegends)
	}
	b.WriteString("\n")
	if len(legendIDs) > 0 {
		fmt.Fprintf(&b, "  legend id range:        %s … %s\n", legendIDs[0], legendIDs[len(legendIDs)-1])
	}
	fmt.Print(b.String())
}
