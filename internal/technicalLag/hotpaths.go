package technicalLag

import (
	"sort"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

// HotPathComponent represents a component that contributes significantly to technical lag
type HotPathComponent struct {
	Component         cdx.Component `json:"component"`
	Contribution      float64       `json:"contribution"`      // Actual lag contribution
	PercentageOfTotal float64       `json:"percentageOfTotal"` // Percentage of total lag
	CumulativePercent float64       `json:"cumulativePercent"` // Cumulative percentage when sorted
}

// HotPath represents hotpath analysis results for a specific metric and scope
type HotPath struct {
	Metric               string             `json:"metric"`               // "libyears" or "versionDistance"
	Scope                string             `json:"scope"`                // e.g., "production", "optional", "directProduction", etc.
	TotalLag             float64            `json:"totalLag"`             // Total lag for this scope
	HotPathThreshold     float64            `json:"hotPathThreshold"`     // Threshold for hotpath (50% of total)
	HotPathComponents    []HotPathComponent `json:"hotPathComponents"`    // Components contributing >50%
	TopContributors      []HotPathComponent `json:"topContributors"`      // Top 10 contributors regardless of threshold
	NumHotPathComponents int                `json:"numHotPathComponents"` // Number of components in hotpath
	HotPathCoverage      float64            `json:"hotPathCoverage"`      // Actual coverage percentage of hotpath components
}

// HotPathAnalysis contains all hotpath analyses
type HotPathAnalysis struct {
	LibyearsHotPaths        []HotPath      `json:"libyearsHotPaths"`
	VersionDistanceHotPaths []HotPath      `json:"versionDistanceHotPaths"`
	Summary                 HotPathSummary `json:"summary"`
}

// HotPathSummary provides high-level hotpath insights
type HotPathSummary struct {
	MostCriticalScope    string  `json:"mostCriticalScope"`    // Scope with highest concentration
	HighestConcentration float64 `json:"highestConcentration"` // Highest single component contribution percentage
	MostFragmentedScope  string  `json:"mostFragmentedScope"`  // Scope requiring most components for hotpath
}

// AnalyzeLibyearsHotPaths analyzes hotpaths for libyears across all scopes
func AnalyzeLibyearsHotPaths(result Result) []HotPath {
	var hotPaths []HotPath

	// Analyze production scope
	if result.Production.NumComponents > 0 {
		hotPath := analyzeLibyearsHotPath(result.Production, "production")
		hotPaths = append(hotPaths, hotPath)
	}

	// Analyze optional scope
	if result.Optional.NumComponents > 0 {
		hotPath := analyzeLibyearsHotPath(result.Optional, "optional")
		hotPaths = append(hotPaths, hotPath)
	}

	// Analyze direct production scope
	if result.DirectProduction.NumComponents > 0 {
		hotPath := analyzeLibyearsHotPath(result.DirectProduction, "directProduction")
		hotPaths = append(hotPaths, hotPath)
	}

	// Analyze direct optional scope
	if result.DirectOptional.NumComponents > 0 {
		hotPath := analyzeLibyearsHotPath(result.DirectOptional, "directOptional")
		hotPaths = append(hotPaths, hotPath)
	}

	return hotPaths
}

// analyzeLibyearsHotPath analyzes hotpath for libyears in a specific scope
func analyzeLibyearsHotPath(stats TechLagStats, scope string) HotPath {
	totalLibyears := stats.Libdays
	threshold := totalLibyears * 0.5 // 50% threshold

	// Create component contributions and sort by libyears (descending)
	var contributors []HotPathComponent
	for _, comp := range stats.Components {
		if comp.Libdays > 0 { // Only include components with actual lag
			percentage := (comp.Libdays / totalLibyears) * 100
			contributors = append(contributors, HotPathComponent{
				Component:         comp.Component,
				Contribution:      comp.Libdays,
				PercentageOfTotal: percentage,
			})
		}
	}

	// Sort by contribution (descending)
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Contribution > contributors[j].Contribution
	})

	// Calculate cumulative percentages and identify hotpath components
	var hotPathComponents []HotPathComponent
	var cumulativeContribution float64
	var cumulativePercent float64

	for i := range contributors {
		cumulativeContribution += contributors[i].Contribution
		cumulativePercent = (cumulativeContribution / totalLibyears) * 100
		contributors[i].CumulativePercent = cumulativePercent

		// Add component to hotpath
		hotPathComponents = append(hotPathComponents, contributors[i])

		// Stop if we've reached or exceeded the threshold
		if cumulativeContribution >= threshold {
			break
		}
	}

	// Get top 10 contributors
	topContributors := contributors
	if len(topContributors) > 10 {
		topContributors = topContributors[:10]
	}

	// Calculate actual hotpath coverage
	var hotPathCoverage float64
	if len(hotPathComponents) > 0 {
		var hotPathTotal float64
		for _, comp := range hotPathComponents {
			hotPathTotal += comp.Contribution
		}
		hotPathCoverage = (hotPathTotal / totalLibyears) * 100
	}

	return HotPath{
		Metric:               "libyears",
		Scope:                scope,
		TotalLag:             totalLibyears,
		HotPathThreshold:     threshold,
		HotPathComponents:    hotPathComponents,
		TopContributors:      topContributors,
		NumHotPathComponents: len(hotPathComponents),
		HotPathCoverage:      hotPathCoverage,
	}
}

// AnalyzeVersionDistanceHotPaths analyzes hotpaths for version distance across all scopes
func AnalyzeVersionDistanceHotPaths(result Result) []HotPath {
	var hotPaths []HotPath

	// Analyze production scope
	if result.Production.NumComponents > 0 {
		hotPath := analyzeVersionDistanceHotPath(result.Production, "production")
		hotPaths = append(hotPaths, hotPath)
	}

	// Analyze optional scope
	if result.Optional.NumComponents > 0 {
		hotPath := analyzeVersionDistanceHotPath(result.Optional, "optional")
		hotPaths = append(hotPaths, hotPath)
	}

	// Analyze direct production scope
	if result.DirectProduction.NumComponents > 0 {
		hotPath := analyzeVersionDistanceHotPath(result.DirectProduction, "directProduction")
		hotPaths = append(hotPaths, hotPath)
	}

	// Analyze direct optional scope
	if result.DirectOptional.NumComponents > 0 {
		hotPath := analyzeVersionDistanceHotPath(result.DirectOptional, "directOptional")
		hotPaths = append(hotPaths, hotPath)
	}

	return hotPaths
}

// analyzeVersionDistanceHotPath analyzes hotpath for version distance in a specific scope
func analyzeVersionDistanceHotPath(stats TechLagStats, scope string) HotPath {
	totalMissedReleases := float64(stats.MissedReleases)
	threshold := totalMissedReleases * 0.5 // 50% threshold

	// Create component contributions and sort by missed releases (descending)
	var contributors []HotPathComponent
	for _, comp := range stats.Components {
		if comp.MissedReleases > 0 { // Only include components with actual lag
			percentage := (float64(comp.MissedReleases) / totalMissedReleases) * 100
			contributors = append(contributors, HotPathComponent{
				Component:         comp.Component,
				Contribution:      float64(comp.MissedReleases),
				PercentageOfTotal: percentage,
			})
		}
	}

	// Sort by contribution (descending)
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].Contribution > contributors[j].Contribution
	})

	// Calculate cumulative percentages and identify hotpath components
	var hotPathComponents []HotPathComponent
	var cumulativeContribution float64
	var cumulativePercent float64

	for i := range contributors {
		cumulativeContribution += contributors[i].Contribution
		cumulativePercent = (cumulativeContribution / totalMissedReleases) * 100
		contributors[i].CumulativePercent = cumulativePercent

		// Add component to hotpath
		hotPathComponents = append(hotPathComponents, contributors[i])

		// Stop if we've reached or exceeded the threshold
		if cumulativeContribution >= threshold {
			break
		}
	}

	// Get top 10 contributors
	topContributors := contributors
	if len(topContributors) > 10 {
		topContributors = topContributors[:10]
	}

	// Calculate actual hotpath coverage
	var hotPathCoverage float64
	if len(hotPathComponents) > 0 {
		var hotPathTotal float64
		for _, comp := range hotPathComponents {
			hotPathTotal += comp.Contribution
		}
		hotPathCoverage = (hotPathTotal / totalMissedReleases) * 100
	}

	return HotPath{
		Metric:               "versionDistance",
		Scope:                scope,
		TotalLag:             totalMissedReleases,
		HotPathThreshold:     threshold,
		HotPathComponents:    hotPathComponents,
		TopContributors:      topContributors,
		NumHotPathComponents: len(hotPathComponents),
		HotPathCoverage:      hotPathCoverage,
	}
}

// CreateHotPathAnalysis performs comprehensive hotpath analysis
func CreateHotPathAnalysis(result Result) HotPathAnalysis {
	libyearsHotPaths := AnalyzeLibyearsHotPaths(result)
	versionDistanceHotPaths := AnalyzeVersionDistanceHotPaths(result)

	summary := createHotPathSummary(libyearsHotPaths, versionDistanceHotPaths)

	return HotPathAnalysis{
		LibyearsHotPaths:        libyearsHotPaths,
		VersionDistanceHotPaths: versionDistanceHotPaths,
		Summary:                 summary,
	}
}

// createHotPathSummary creates a summary of hotpath analysis insights
func createHotPathSummary(libyearsHotPaths, versionDistanceHotPaths []HotPath) HotPathSummary {
	var mostCriticalScope string
	var highestConcentration float64
	var mostFragmentedScope string
	var maxComponents int

	// Find highest single component contribution
	allHotPaths := append(libyearsHotPaths, versionDistanceHotPaths...)
	for _, hotPath := range allHotPaths {
		if len(hotPath.HotPathComponents) > 0 {
			firstComponentPercent := hotPath.HotPathComponents[0].PercentageOfTotal
			if firstComponentPercent > highestConcentration {
				highestConcentration = firstComponentPercent
				mostCriticalScope = hotPath.Scope + " (" + hotPath.Metric + ")"
			}
		}

		// Find most fragmented scope (most components needed for hotpath)
		if hotPath.NumHotPathComponents > maxComponents {
			maxComponents = hotPath.NumHotPathComponents
			mostFragmentedScope = hotPath.Scope + " (" + hotPath.Metric + ")"
		}
	}

	return HotPathSummary{
		MostCriticalScope:    mostCriticalScope,
		HighestConcentration: highestConcentration,
		MostFragmentedScope:  mostFragmentedScope,
	}
}
