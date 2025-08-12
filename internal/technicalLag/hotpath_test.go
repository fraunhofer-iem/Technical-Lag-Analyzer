package technicalLag

import (
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

func TestAnalyzeLibyearsHotPath(t *testing.T) {
	// Create test data with known distribution
	testStats := TechLagStats{
		Libdays:       100.0, // Total libyears
		NumComponents: 4,
		Components: []ComponentLag{
			{
				Component: cdx.Component{Name: "critical", Version: "1.0.0"},
				Libdays:   60.0, // 60% of total - should be hotpath alone
			},
			{
				Component: cdx.Component{Name: "moderate", Version: "2.0.0"},
				Libdays:   25.0, // 25% of total
			},
			{
				Component: cdx.Component{Name: "minor", Version: "3.0.0"},
				Libdays:   10.0, // 10% of total
			},
			{
				Component: cdx.Component{Name: "minimal", Version: "4.0.0"},
				Libdays:   5.0, // 5% of total
			},
		},
	}

	hotPath := analyzeLibyearsHotPath(testStats, "test")

	// Verify basic properties
	if hotPath.Metric != "libyears" {
		t.Errorf("Expected metric 'libyears', got '%s'", hotPath.Metric)
	}
	if hotPath.Scope != "test" {
		t.Errorf("Expected scope 'test', got '%s'", hotPath.Scope)
	}
	if hotPath.TotalLag != 100.0 {
		t.Errorf("Expected total lag 100.0, got %f", hotPath.TotalLag)
	}
	if hotPath.HotPathThreshold != 50.0 {
		t.Errorf("Expected threshold 50.0, got %f", hotPath.HotPathThreshold)
	}

	// Verify hotpath components - only the critical component should be in hotpath
	if len(hotPath.HotPathComponents) != 1 {
		t.Errorf("Expected 1 hotpath component, got %d", len(hotPath.HotPathComponents))
	}
	if len(hotPath.HotPathComponents) > 0 {
		criticalComp := hotPath.HotPathComponents[0]
		if criticalComp.Component.Name != "critical" {
			t.Errorf("Expected critical component in hotpath, got %s", criticalComp.Component.Name)
		}
		if criticalComp.PercentageOfTotal != 60.0 {
			t.Errorf("Expected 60%% contribution, got %f%%", criticalComp.PercentageOfTotal)
		}
		if criticalComp.CumulativePercent != 60.0 {
			t.Errorf("Expected 60%% cumulative, got %f%%", criticalComp.CumulativePercent)
		}
	}

	// Verify hotpath coverage
	if hotPath.HotPathCoverage != 60.0 {
		t.Errorf("Expected hotpath coverage 60%%, got %f%%", hotPath.HotPathCoverage)
	}

	// Verify top contributors
	if len(hotPath.TopContributors) != 4 {
		t.Errorf("Expected 4 top contributors, got %d", len(hotPath.TopContributors))
	}

	// Verify sorting (descending by contribution)
	if len(hotPath.TopContributors) >= 2 {
		if hotPath.TopContributors[0].Contribution < hotPath.TopContributors[1].Contribution {
			t.Error("Top contributors not sorted in descending order")
		}
	}
}

func TestAnalyzeLibyearsHotPathMultipleComponents(t *testing.T) {
	// Test case where multiple components are needed to reach 50%
	testStats := TechLagStats{
		Libdays:       100.0,
		NumComponents: 3,
		Components: []ComponentLag{
			{
				Component: cdx.Component{Name: "comp1", Version: "1.0.0"},
				Libdays:   30.0, // 30%
			},
			{
				Component: cdx.Component{Name: "comp2", Version: "2.0.0"},
				Libdays:   25.0, // 25%
			},
			{
				Component: cdx.Component{Name: "comp3", Version: "3.0.0"},
				Libdays:   45.0, // 45%
			},
		},
	}

	hotPath := analyzeLibyearsHotPath(testStats, "test")

	// Should have 2 components: comp3 (45%) + comp1 (30%) = 75% > 50%
	if len(hotPath.HotPathComponents) != 2 {
		t.Errorf("Expected 2 hotpath components, got %d", len(hotPath.HotPathComponents))
	}

	// Verify components are sorted by contribution
	if len(hotPath.HotPathComponents) >= 2 {
		if hotPath.HotPathComponents[0].Component.Name != "comp3" {
			t.Errorf("Expected comp3 first, got %s", hotPath.HotPathComponents[0].Component.Name)
		}
		if hotPath.HotPathComponents[1].Component.Name != "comp1" {
			t.Errorf("Expected comp1 second, got %s", hotPath.HotPathComponents[1].Component.Name)
		}
	}

	// Verify coverage is around 75%
	expectedCoverage := 75.0
	if hotPath.HotPathCoverage != expectedCoverage {
		t.Errorf("Expected hotpath coverage %.1f%%, got %.1f%%", expectedCoverage, hotPath.HotPathCoverage)
	}
}

func TestAnalyzeVersionDistanceHotPath(t *testing.T) {
	// Create test data for version distance analysis
	testStats := TechLagStats{
		MissedReleases: 20, // Total missed releases
		NumComponents:  3,
		Components: []ComponentLag{
			{
				Component:      cdx.Component{Name: "outdated", Version: "1.0.0"},
				MissedReleases: 12, // 60% of total
			},
			{
				Component:      cdx.Component{Name: "somewhat", Version: "2.0.0"},
				MissedReleases: 5, // 25% of total
			},
			{
				Component:      cdx.Component{Name: "recent", Version: "3.0.0"},
				MissedReleases: 3, // 15% of total
			},
		},
	}

	hotPath := analyzeVersionDistanceHotPath(testStats, "test")

	// Verify basic properties
	if hotPath.Metric != "versionDistance" {
		t.Errorf("Expected metric 'versionDistance', got '%s'", hotPath.Metric)
	}
	if hotPath.TotalLag != 20.0 {
		t.Errorf("Expected total lag 20.0, got %f", hotPath.TotalLag)
	}
	if hotPath.HotPathThreshold != 10.0 {
		t.Errorf("Expected threshold 10.0, got %f", hotPath.HotPathThreshold)
	}

	// Should have 1 component (outdated with 60%)
	if len(hotPath.HotPathComponents) != 1 {
		t.Errorf("Expected 1 hotpath component, got %d", len(hotPath.HotPathComponents))
	}

	if len(hotPath.HotPathComponents) > 0 {
		comp := hotPath.HotPathComponents[0]
		if comp.Component.Name != "outdated" {
			t.Errorf("Expected 'outdated' component, got %s", comp.Component.Name)
		}
		if comp.PercentageOfTotal != 60.0 {
			t.Errorf("Expected 60%% contribution, got %f%%", comp.PercentageOfTotal)
		}
	}
}

func TestAnalyzeLibyearsHotPaths(t *testing.T) {
	// Create a full result with multiple scopes
	result := Result{
		Production: TechLagStats{
			Libdays:       50.0,
			NumComponents: 2,
			Components: []ComponentLag{
				{Component: cdx.Component{Name: "prod1"}, Libdays: 30.0},
				{Component: cdx.Component{Name: "prod2"}, Libdays: 20.0},
			},
		},
		Optional: TechLagStats{
			Libdays:       30.0,
			NumComponents: 1,
			Components: []ComponentLag{
				{Component: cdx.Component{Name: "opt1"}, Libdays: 30.0},
			},
		},
		DirectProduction: TechLagStats{
			Libdays:       25.0,
			NumComponents: 1,
			Components: []ComponentLag{
				{Component: cdx.Component{Name: "direct1"}, Libdays: 25.0},
			},
		},
		DirectOptional: TechLagStats{
			NumComponents: 0, // Should be skipped
		},
	}

	hotPaths := AnalyzeLibyearsHotPaths(result)

	// Should have 3 hotpaths (skipping DirectOptional with 0 components)
	if len(hotPaths) != 3 {
		t.Errorf("Expected 3 hotpaths, got %d", len(hotPaths))
	}

	// Verify scopes
	expectedScopes := map[string]bool{
		"production":       false,
		"optional":         false,
		"directProduction": false,
	}

	for _, hotPath := range hotPaths {
		if _, exists := expectedScopes[hotPath.Scope]; !exists {
			t.Errorf("Unexpected scope: %s", hotPath.Scope)
		}
		expectedScopes[hotPath.Scope] = true
	}

	for scope, found := range expectedScopes {
		if !found {
			t.Errorf("Missing scope: %s", scope)
		}
	}
}

func TestCreateHotPathAnalysis(t *testing.T) {
	result := Result{
		Production: TechLagStats{
			Libdays:        100.0,
			MissedReleases: 50,
			NumComponents:  2,
			Components: []ComponentLag{
				{
					Component:      cdx.Component{Name: "critical", Version: "1.0.0"},
					Libdays:        80.0,
					MissedReleases: 40,
				},
				{
					Component:      cdx.Component{Name: "minor", Version: "2.0.0"},
					Libdays:        20.0,
					MissedReleases: 10,
				},
			},
		},
	}

	analysis := CreateHotPathAnalysis(result)

	// Should have both libyears and version distance hotpaths
	if len(analysis.LibyearsHotPaths) == 0 {
		t.Error("Expected libyears hotpaths")
	}
	if len(analysis.VersionDistanceHotPaths) == 0 {
		t.Error("Expected version distance hotpaths")
	}

	// Verify summary contains meaningful data
	if analysis.Summary.MostCriticalScope == "" {
		t.Error("Expected most critical scope to be identified")
	}
	if analysis.Summary.HighestConcentration <= 0 {
		t.Error("Expected highest concentration to be positive")
	}
	if analysis.Summary.MostFragmentedScope == "" {
		t.Error("Expected most fragmented scope to be identified")
	}
}

func TestHotPathWithZeroComponents(t *testing.T) {
	// Test edge case with no components
	testStats := TechLagStats{
		NumComponents: 0,
		Components:    []ComponentLag{},
	}

	hotPath := analyzeLibyearsHotPath(testStats, "empty")

	if len(hotPath.HotPathComponents) != 0 {
		t.Errorf("Expected 0 hotpath components for empty stats, got %d", len(hotPath.HotPathComponents))
	}
	if len(hotPath.TopContributors) != 0 {
		t.Errorf("Expected 0 top contributors for empty stats, got %d", len(hotPath.TopContributors))
	}
	if hotPath.HotPathCoverage != 0 {
		t.Errorf("Expected 0 coverage for empty stats, got %f", hotPath.HotPathCoverage)
	}
}

func TestHotPathWithZeroLag(t *testing.T) {
	// Test with components that have zero lag
	testStats := TechLagStats{
		Libdays:       0.0,
		NumComponents: 2,
		Components: []ComponentLag{
			{Component: cdx.Component{Name: "comp1"}, Libdays: 0.0},
			{Component: cdx.Component{Name: "comp2"}, Libdays: 0.0},
		},
	}

	hotPath := analyzeLibyearsHotPath(testStats, "zero")

	if len(hotPath.HotPathComponents) != 0 {
		t.Errorf("Expected 0 hotpath components for zero lag, got %d", len(hotPath.HotPathComponents))
	}
	if len(hotPath.TopContributors) != 0 {
		t.Errorf("Expected 0 top contributors for zero lag, got %d", len(hotPath.TopContributors))
	}
}
