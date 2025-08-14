package technicalLag

import (
	"sbom-technical-lag/internal/semver"
	"slices"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

// createMockBOM creates a simple mock BOM for testing
func createMockBOM() *cdx.BOM {
	return &cdx.BOM{
		Metadata: &cdx.Metadata{
			Component: &cdx.Component{
				BOMRef: "pkg:test/project@1.0.0",
				Name:   "test-project",
			},
		},
		Components: &[]cdx.Component{},
		Dependencies: &[]cdx.Dependency{
			{
				Ref:          "pkg:test/project@1.0.0",
				Dependencies: &[]string{},
			},
		},
	}
}

func TestUpdateTechLagStats(t *testing.T) {
	stats := &TechLagStats{}

	component := cdx.Component{
		Name:    "test-component",
		Version: "1.0.0",
	}

	versionDistance := semver.VersionDistance{
		MissedReleases: 10,
		MissedMajor:    2,
		MissedMinor:    3,
		MissedPatch:    5,
	}

	technicalLag := TechnicalLag{
		Libdays:         100.5,
		VersionDistance: versionDistance,
	}

	componentLag := ComponentLag{
		Component:        component,
		TechnicalLag:     technicalLag,
		CriticalityScore: 0.5,
	}

	mockBOM := createMockBOM()
	componentMetrics := map[cdx.Component]TechnicalLag{component: technicalLag}
	updateTechLagStats(stats, technicalLag, component, componentLag, mockBOM, componentMetrics)

	// Test that all values were properly added
	if stats.TotalLibdays != 100.5 {
		t.Errorf("Expected TotalLibdays to be 100.5, got %f", stats.TotalLibdays)
	}

	if stats.TotalMissedMajor != 2 {
		t.Errorf("Expected TotalMissedMajor to be 2, got %d", stats.TotalMissedMajor)
	}

	if stats.TotalMissedMinor != 3 {
		t.Errorf("Expected TotalMissedMinor to be 3, got %d", stats.TotalMissedMinor)
	}

	if stats.TotalMissedPatch != 5 {
		t.Errorf("Expected TotalMissedPatch to be 5, got %d", stats.TotalMissedPatch)
	}

	if stats.TotalNumComponents != 1 {
		t.Errorf("Expected TotalNumComponents to be 1, got %d", stats.TotalNumComponents)
	}

	if stats.HighestLibdays != 100.5 {
		t.Errorf("Expected HighestLibdays to be 100.5, got %f", stats.HighestLibdays)
	}

	if stats.HighestMissedReleases != 10 {
		t.Errorf("Expected HighestMissedReleases to be 10, got %d", stats.HighestMissedReleases)
	}

	if stats.ComponentHighestLibdays.Name != "test-component" {
		t.Errorf("Expected ComponentHighestLibdays name to be 'test-component', got %s", stats.ComponentHighestLibdays.Name)
	}

	if stats.ComponentHighestMissedReleases.Name != "test-component" {
		t.Errorf("Expected ComponentHighestMissedReleases name to be 'test-component', got %s", stats.ComponentHighestMissedReleases.Name)
	}

	if stats.HighestCriticalityScore != 0.5 {
		t.Errorf("Expected HighestCriticalityScore to be 0.5, got %f", stats.HighestCriticalityScore)
	}

	if stats.ComponentHighestCriticalityScore.Name != "test-component" {
		t.Errorf("Expected ComponentHighestCriticalityScore name to be 'test-component', got %s", stats.ComponentHighestCriticalityScore.Name)
	}

	// Test that the component was added to the Components slice
	if len(stats.Components) != 1 {
		t.Errorf("Expected 1 component in Components slice, got %d", len(stats.Components))
	}

	if stats.Components[0].Component.Name != "test-component" {
		t.Errorf("Expected first component name to be 'test-component', got %s", stats.Components[0].Component.Name)
	}
}

func TestUpdateTechLagStatsMultipleComponents(t *testing.T) {
	stats := &TechLagStats{}

	// First component
	component1 := cdx.Component{
		Name:    "component-1",
		Version: "1.0.0",
	}

	versionDistance1 := semver.VersionDistance{
		MissedReleases: 5,
		MissedMajor:    1,
		MissedMinor:    2,
		MissedPatch:    2,
	}

	technicalLag1 := TechnicalLag{
		Libdays:         50.0,
		VersionDistance: versionDistance1,
	}

	// Second component
	component2 := cdx.Component{
		Name:    "component-2",
		Version: "2.0.0",
	}

	versionDistance2 := semver.VersionDistance{
		MissedReleases: 15,
		MissedMajor:    3,
		MissedMinor:    4,
		MissedPatch:    8,
	}

	technicalLag2 := TechnicalLag{
		Libdays:         75.5,
		VersionDistance: versionDistance2,
	}

	// Create ComponentLag instances
	componentLag1 := ComponentLag{
		Component:        component1,
		TechnicalLag:     technicalLag1,
		CriticalityScore: 0.3,
	}

	componentLag2 := ComponentLag{
		Component:        component2,
		TechnicalLag:     technicalLag2,
		CriticalityScore: 0.7,
	}

	// Update stats with both components
	mockBOM := createMockBOM()
	componentMetrics := map[cdx.Component]TechnicalLag{
		component1: technicalLag1,
		component2: technicalLag2,
	}
	updateTechLagStats(stats, technicalLag1, component1, componentLag1, mockBOM, componentMetrics)
	updateTechLagStats(stats, technicalLag2, component2, componentLag2, mockBOM, componentMetrics)

	// Test accumulated values
	if stats.TotalLibdays != 125.5 {
		t.Errorf("Expected TotalLibdays to be 125.5, got %f", stats.TotalLibdays)
	}

	if stats.TotalMissedMajor != 4 {
		t.Errorf("Expected TotalMissedMajor to be 4, got %d", stats.TotalMissedMajor)
	}

	if stats.TotalMissedMinor != 6 {
		t.Errorf("Expected TotalMissedMinor to be 6, got %d", stats.TotalMissedMinor)
	}

	if stats.TotalMissedPatch != 10 {
		t.Errorf("Expected TotalMissedPatch to be 10, got %d", stats.TotalMissedPatch)
	}

	if stats.TotalNumComponents != 2 {
		t.Errorf("Expected TotalNumComponents to be 2, got %d", stats.TotalNumComponents)
	}

	// Test that highest values are tracked correctly
	if stats.HighestLibdays != 75.5 {
		t.Errorf("Expected HighestLibdays to be 75.5, got %f", stats.HighestLibdays)
	}

	if stats.HighestMissedReleases != 15 {
		t.Errorf("Expected HighestMissedReleases to be 15, got %d", stats.HighestMissedReleases)
	}

	if stats.ComponentHighestLibdays.Name != "component-2" {
		t.Errorf("Expected ComponentHighestLibdays name to be 'component-2', got %s", stats.ComponentHighestLibdays.Name)
	}

	if stats.ComponentHighestMissedReleases.Name != "component-2" {
		t.Errorf("Expected ComponentHighestMissedReleases name to be 'component-2', got %s", stats.ComponentHighestMissedReleases.Name)
	}

	if stats.HighestCriticalityScore != 0.7 {
		t.Errorf("Expected HighestCriticalityScore to be 0.7, got %f", stats.HighestCriticalityScore)
	}

	if stats.ComponentHighestCriticalityScore.Name != "component-2" {
		t.Errorf("Expected ComponentHighestCriticalityScore name to be 'component-2', got %s", stats.ComponentHighestCriticalityScore.Name)
	}

	// Test that both components were added to the Components slice
	if len(stats.Components) != 2 {
		t.Errorf("Expected 2 components in Components slice, got %d", len(stats.Components))
	}

	if stats.Components[0].Component.Name != "component-1" {
		t.Errorf("Expected first component name to be 'component-1', got %s", stats.Components[0].Component.Name)
	}

	if stats.Components[1].Component.Name != "component-2" {
		t.Errorf("Expected second component name to be 'component-2', got %s", stats.Components[1].Component.Name)
	}
}

func TestComponentLagCreation(t *testing.T) {
	component := cdx.Component{
		Name:    "test-component",
		Version: "1.0.0",
	}

	versionDistance := semver.VersionDistance{
		MissedReleases: 12,
		MissedMajor:    2,
		MissedMinor:    3,
		MissedPatch:    7,
	}

	technicalLag := TechnicalLag{
		Libdays:         42.5,
		VersionDistance: versionDistance,
	}

	componentLag := ComponentLag{
		Component:        component,
		TechnicalLag:     technicalLag,
		CriticalityScore: 0.25,
	}

	if componentLag.Component.Name != "test-component" {
		t.Errorf("Expected component name to be 'test-component', got %s", componentLag.Component.Name)
	}

	if componentLag.Libdays() != 42.5 {
		t.Errorf("Expected Libdays to be 42.5, got %f", componentLag.Libdays())
	}

	if componentLag.MissedReleases() != 12 {
		t.Errorf("Expected MissedReleases to be 12, got %d", componentLag.MissedReleases())
	}

	if componentLag.MissedMajor() != 2 {
		t.Errorf("Expected MissedMajor to be 2, got %d", componentLag.MissedMajor())
	}

	if componentLag.MissedMinor() != 3 {
		t.Errorf("Expected MissedMinor to be 3, got %d", componentLag.MissedMinor())
	}

	if componentLag.MissedPatch() != 7 {
		t.Errorf("Expected MissedPatch to be 7, got %d", componentLag.MissedPatch())
	}

	if componentLag.CriticalityScore != 0.25 {
		t.Errorf("Expected CriticalityScore to be 0.25, got %f", componentLag.CriticalityScore)
	}
}

func TestTechLagStatsZeroValues(t *testing.T) {
	stats := &TechLagStats{}

	component := cdx.Component{
		Name:    "zero-component",
		Version: "1.0.0",
	}

	versionDistance := semver.VersionDistance{
		MissedReleases: 0,
		MissedMajor:    0,
		MissedMinor:    0,
		MissedPatch:    0,
	}

	technicalLag := TechnicalLag{
		Libdays:         0.0,
		VersionDistance: versionDistance,
	}

	componentLag := ComponentLag{
		Component:        component,
		TechnicalLag:     technicalLag,
		CriticalityScore: 0.0,
	}

	mockBOM := createMockBOM()
	componentMetrics := map[cdx.Component]TechnicalLag{component: technicalLag}
	updateTechLagStats(stats, technicalLag, component, componentLag, mockBOM, componentMetrics)

	// Test that all values are zero
	if stats.TotalLibdays != 0 {
		t.Errorf("Expected TotalLibdays to be 0, got %f", stats.TotalLibdays)
	}

	if stats.TotalMissedReleases != 0 {
		t.Errorf("Expected TotalMissedReleases to be 0, got %d", stats.TotalMissedReleases)
	}

	if stats.TotalMissedMajor != 0 {
		t.Errorf("Expected TotalMissedMajor to be 0, got %d", stats.TotalMissedMajor)
	}

	if stats.TotalMissedMinor != 0 {
		t.Errorf("Expected TotalMissedMinor to be 0, got %d", stats.TotalMissedMinor)
	}

	if stats.TotalMissedPatch != 0 {
		t.Errorf("Expected TotalMissedPatch to be 0, got %d", stats.TotalMissedPatch)
	}

	if stats.TotalNumComponents != 1 {
		t.Errorf("Expected TotalNumComponents to be 1, got %d", stats.TotalNumComponents)
	}

	if stats.HighestLibdays != 0.0 {
		t.Errorf("Expected HighestLibdays to be 0.0, got %f", stats.HighestLibdays)
	}

	if stats.HighestMissedReleases != 0 {
		t.Errorf("Expected HighestMissedReleases to be 0, got %d", stats.HighestMissedReleases)
	}

	if stats.HighestCriticalityScore != 0.0 {
		t.Errorf("Expected HighestCriticalityScore to be 0.0, got %f", stats.HighestCriticalityScore)
	}

	if stats.ComponentHighestCriticalityScore.Name != "" {
		t.Errorf("Expected ComponentHighestCriticalityScore name to be empty, got %s", stats.ComponentHighestCriticalityScore.Name)
	}

	// Test that the component was added to the Components slice
	if len(stats.Components) != 1 {
		t.Errorf("Expected 1 component in Components slice, got %d", len(stats.Components))
	}
}

func TestResultStringFormat(t *testing.T) {
	// Create test components with technical lag data
	prodComp1 := ComponentLag{
		Component: cdx.Component{Name: "prod1"},
		TechnicalLag: TechnicalLag{
			Libdays:         100.25,
			VersionDistance: semver.VersionDistance{MissedReleases: 10, MissedMajor: 2, MissedMinor: 3, MissedPatch: 5},
		},
	}
	prodComp2 := ComponentLag{
		Component: cdx.Component{Name: "prod2"},
		TechnicalLag: TechnicalLag{
			Libdays:         50.25,
			VersionDistance: semver.VersionDistance{MissedReleases: 5, MissedMajor: 1, MissedMinor: 2, MissedPatch: 2},
		},
	}
	optComp := ComponentLag{
		Component: cdx.Component{Name: "opt1"},
		TechnicalLag: TechnicalLag{
			Libdays:         50.25,
			VersionDistance: semver.VersionDistance{MissedReleases: 6, MissedMajor: 1, MissedMinor: 2, MissedPatch: 3},
		},
	}
	directProdComp := ComponentLag{
		Component: cdx.Component{Name: "direct1"},
		TechnicalLag: TechnicalLag{
			Libdays:         75.0,
			VersionDistance: semver.VersionDistance{MissedReleases: 4, MissedMajor: 2, MissedMinor: 1, MissedPatch: 1},
		},
	}

	result := Result{
		Production: TechLagStats{
			HighestLibdays:        100.25,
			HighestMissedReleases: 10,
			Components:            []ComponentLag{prodComp1, prodComp2},
		},
		Optional: TechLagStats{
			HighestLibdays:        50.25,
			HighestMissedReleases: 6,
			Components:            []ComponentLag{optComp},
		},
		DirectProduction: TechLagStats{
			HighestLibdays:        75.0,
			HighestMissedReleases: 4,
			Components:            []ComponentLag{directProdComp},
		},
		DirectOptional: TechLagStats{
			HighestLibdays:        0.0,
			HighestMissedReleases: 0,
			Components:            make([]ComponentLag, 0),
		},
	}

	// Compute totals for the manually created stats
	result.Production.computeTotals()
	result.Optional.computeTotals()
	result.DirectProduction.computeTotals()
	result.DirectOptional.computeTotals()

	output := result.String()

	// Just verify that the string contains expected sections and some key values
	if output == "" {
		t.Error("Expected non-empty string output")
	}

	// Check for section headers
	if !contains(output, "=== Technical Lag Analysis ===") {
		t.Error("Expected output to contain '=== Technical Lag Analysis ==='")
	}

	if !contains(output, "=== Summary ===") {
		t.Error("Expected output to contain '=== Summary ==='")
	}

	// Check for some key values (basic sanity check)
	if !contains(output, "150.50") { // Prod libdays (100.25 + 50.25)
		t.Error("Expected output to contain production libdays value")
	}

	if !contains(output, "50.25") { // Opt libdays
		t.Error("Expected output to contain optional libdays value")
	}
}

func TestSerializedComputedValues(t *testing.T) {
	// Create test components with technical lag data
	prodComp1 := ComponentLag{
		Component: cdx.Component{Name: "prod1"},
		TechnicalLag: TechnicalLag{
			Libdays:         100.25,
			VersionDistance: semver.VersionDistance{MissedReleases: 10, MissedMajor: 2, MissedMinor: 3, MissedPatch: 5},
		},
	}
	prodComp2 := ComponentLag{
		Component: cdx.Component{Name: "prod2"},
		TechnicalLag: TechnicalLag{
			Libdays:         50.25,
			VersionDistance: semver.VersionDistance{MissedReleases: 5, MissedMajor: 1, MissedMinor: 2, MissedPatch: 2},
		},
	}

	stats := TechLagStats{
		HighestLibdays:        100.25,
		HighestMissedReleases: 10,
		Components:            []ComponentLag{prodComp1, prodComp2},
	}

	// Compute totals to simulate serialization preparation
	stats.computeTotals()

	// Verify computed totals are correct

	// Verify expected values
	expectedLibdays := 150.5            // 100.25 + 50.25
	expectedMissedReleases := int64(15) // 10 + 5
	expectedMissedMajor := int64(3)     // 2 + 1
	expectedMissedMinor := int64(5)     // 3 + 2
	expectedMissedPatch := int64(7)     // 5 + 2
	expectedNumComponents := 2

	if stats.TotalLibdays != expectedLibdays {
		t.Errorf("Expected TotalLibdays %f, got %f", expectedLibdays, stats.TotalLibdays)
	}

	if stats.TotalMissedReleases != expectedMissedReleases {
		t.Errorf("Expected TotalMissedReleases %d, got %d", expectedMissedReleases, stats.TotalMissedReleases)
	}

	if stats.TotalMissedMajor != expectedMissedMajor {
		t.Errorf("Expected TotalMissedMajor %d, got %d", expectedMissedMajor, stats.TotalMissedMajor)
	}

	if stats.TotalMissedMinor != expectedMissedMinor {
		t.Errorf("Expected TotalMissedMinor %d, got %d", expectedMissedMinor, stats.TotalMissedMinor)
	}

	if stats.TotalMissedPatch != expectedMissedPatch {
		t.Errorf("Expected TotalMissedPatch %d, got %d", expectedMissedPatch, stats.TotalMissedPatch)
	}

	if stats.TotalNumComponents != expectedNumComponents {
		t.Errorf("Expected TotalNumComponents %d, got %d", expectedNumComponents, stats.TotalNumComponents)
	}
}

func TestComponentScopeSeparation(t *testing.T) {
	// Create mock components with different scopes
	prodComponent := cdx.Component{
		Name:    "prod-component",
		Version: "1.0.0",
		Scope:   "required",
	}

	optComponent := cdx.Component{
		Name:    "opt-component",
		Version: "2.0.0",
		Scope:   "optional",
	}

	directProdComponent := cdx.Component{
		Name:    "direct-prod",
		Version: "3.0.0",
		Scope:   "required",
	}

	directOptComponent := cdx.Component{
		Name:    "direct-opt",
		Version: "4.0.0",
		Scope:   "optional",
	}

	// Create component map
	cm := map[cdx.Component]TechnicalLag{
		prodComponent: {
			Libdays: 50.0,
			VersionDistance: semver.VersionDistance{
				MissedReleases: 5,
				MissedMajor:    1,
				MissedMinor:    2,
				MissedPatch:    2,
			},
		},
		optComponent: {
			Libdays: 30.0,
			VersionDistance: semver.VersionDistance{
				MissedReleases: 3,
				MissedMajor:    1,
				MissedMinor:    1,
				MissedPatch:    1,
			},
		},
		directProdComponent: {
			Libdays: 75.0,
			VersionDistance: semver.VersionDistance{
				MissedReleases: 7,
				MissedMajor:    2,
				MissedMinor:    2,
				MissedPatch:    3,
			},
		},
		directOptComponent: {
			Libdays: 25.0,
			VersionDistance: semver.VersionDistance{
				MissedReleases: 2,
				MissedMajor:    0,
				MissedMinor:    1,
				MissedPatch:    1,
			},
		},
	}

	// Mock the direct dependencies (simulate that directProdComponent and directOptComponent are direct)
	directDeps := []cdx.Component{directProdComponent, directOptComponent}

	// Create result manually to test component separation
	result := Result{
		Optional:         TechLagStats{Components: make([]ComponentLag, 0)},
		Production:       TechLagStats{Components: make([]ComponentLag, 0)},
		DirectOptional:   TechLagStats{Components: make([]ComponentLag, 0)},
		DirectProduction: TechLagStats{Components: make([]ComponentLag, 0)},
	}

	// Process all components (both direct and indirect)
	for k, v := range cm {
		componentLag := ComponentLag{
			Component:        k,
			TechnicalLag:     v,
			CriticalityScore: 0.1,
		}

		componentMetrics := map[cdx.Component]TechnicalLag{k: v}
		if k.Scope == "" || k.Scope == "required" {
			updateTechLagStats(&result.Production, v, k, componentLag, createMockBOM(), componentMetrics)
		} else {
			updateTechLagStats(&result.Optional, v, k, componentLag, createMockBOM(), componentMetrics)
		}
	}

	// Process direct dependencies separately
	for _, dep := range directDeps {
		tl := cm[dep]
		componentLag := ComponentLag{
			Component:        dep,
			TechnicalLag:     tl,
			CriticalityScore: 0.2,
		}

		componentMetrics := map[cdx.Component]TechnicalLag{dep: tl}
		if dep.Scope == "" || dep.Scope == "required" {
			updateTechLagStats(&result.DirectProduction, tl, dep, componentLag, createMockBOM(), componentMetrics)
		} else {
			updateTechLagStats(&result.DirectOptional, tl, dep, componentLag, createMockBOM(), componentMetrics)
		}
	}

	// Test Production components
	if len(result.Production.Components) != 2 {
		t.Errorf("Expected 2 production components, got %d", len(result.Production.Components))
	}

	prodNames := make([]string, len(result.Production.Components))
	for i, comp := range result.Production.Components {
		prodNames[i] = comp.Component.Name
	}

	if !containsString(prodNames, "prod-component") {
		t.Error("Expected prod-component in production components")
	}
	if !containsString(prodNames, "direct-prod") {
		t.Error("Expected direct-prod in production components")
	}

	// Test Optional components
	if len(result.Optional.Components) != 2 {
		t.Errorf("Expected 2 optional components, got %d", len(result.Optional.Components))
	}

	optNames := make([]string, len(result.Optional.Components))
	for i, comp := range result.Optional.Components {
		optNames[i] = comp.Component.Name
	}

	if !containsString(optNames, "opt-component") {
		t.Error("Expected opt-component in optional components")
	}
	if !containsString(optNames, "direct-opt") {
		t.Error("Expected direct-opt in optional components")
	}

	// Test Direct Production components
	if len(result.DirectProduction.Components) != 1 {
		t.Errorf("Expected 1 direct production component, got %d", len(result.DirectProduction.Components))
	}

	if result.DirectProduction.Components[0].Component.Name != "direct-prod" {
		t.Errorf("Expected direct-prod in direct production, got %s", result.DirectProduction.Components[0].Component.Name)
	}

	// Test Direct Optional components
	if len(result.DirectOptional.Components) != 1 {
		t.Errorf("Expected 1 direct optional component, got %d", len(result.DirectOptional.Components))
	}

	if result.DirectOptional.Components[0].Component.Name != "direct-opt" {
		t.Errorf("Expected direct-opt in direct optional, got %s", result.DirectOptional.Components[0].Component.Name)
	}

	// Test that statistics match the number of components
	if result.Production.TotalNumComponents != 2 {
		t.Errorf("Expected Production.TotalNumComponents to be 2, got %d", result.Production.TotalNumComponents)
	}
	if result.Optional.TotalNumComponents != 2 {
		t.Errorf("Expected Optional.TotalNumComponents to be 2, got %d", result.Optional.TotalNumComponents)
	}
	if result.DirectProduction.TotalNumComponents != 1 {
		t.Errorf("Expected DirectProduction.TotalNumComponents to be 1, got %d", result.DirectProduction.TotalNumComponents)
	}
	if result.DirectOptional.TotalNumComponents != 1 {
		t.Errorf("Expected DirectOptional.TotalNumComponents to be 1, got %d", result.DirectOptional.TotalNumComponents)
	}
}

// Helper function to check if a slice contains a string
func containsString(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}

// TestCriticalityScoreCalculation tests that criticality scores are properly calculated in CreateResult
func TestCriticalityScoreCalculation(t *testing.T) {
	// Create a simple BOM with dependencies
	bom := &cdx.BOM{
		Components: &[]cdx.Component{
			{
				BOMRef:     "pkg:npm/test-package@1.0.0",
				Name:       "test-package",
				Version:    "1.0.0",
				PackageURL: "pkg:npm/test-package@1.0.0",
				Scope:      "required",
			},
			{
				BOMRef:     "pkg:npm/dependency@2.0.0",
				Name:       "dependency",
				Version:    "2.0.0",
				PackageURL: "pkg:npm/dependency@2.0.0",
				Scope:      "required",
			},
		},
	}

	// Create component metrics
	componentMetrics := map[cdx.Component]TechnicalLag{
		(*bom.Components)[0]: {
			Libdays: 100.0,
			VersionDistance: semver.VersionDistance{
				MissedReleases: 5,
			},
		},
		(*bom.Components)[1]: {
			Libdays: 50.0,
			VersionDistance: semver.VersionDistance{
				MissedReleases: 2,
			},
		},
	}

	// Create result
	result, err := CreateResult(bom, componentMetrics)
	if err != nil {
		t.Fatalf("CreateResult failed: %v", err)
	}

	// Check that criticality scores are present and calculated
	if len(result.Production.Components) == 0 {
		t.Fatal("Expected at least one production component")
	}

	for _, comp := range result.Production.Components {
		if comp.CriticalityScore < 0 {
			t.Errorf("Expected CriticalityScore to be >= 0, got %f for component %s",
				comp.CriticalityScore, comp.Component.Name)
		}
	}
}

// TestHighestCriticalityScoreTracking tests that the highest criticality score is properly tracked
func TestHighestCriticalityScoreTracking(t *testing.T) {
	stats := &TechLagStats{}

	// Component with lower criticality score
	component1 := cdx.Component{
		Name:    "low-criticality",
		Version: "1.0.0",
	}

	technicalLag1 := TechnicalLag{
		Libdays: 25.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 3,
		},
	}

	componentLag1 := ComponentLag{
		Component:        component1,
		TechnicalLag:     technicalLag1,
		CriticalityScore: 0.2,
	}

	// Component with higher criticality score
	component2 := cdx.Component{
		Name:    "high-criticality",
		Version: "2.0.0",
	}

	technicalLag2 := TechnicalLag{
		Libdays: 50.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 7,
		},
	}

	componentLag2 := ComponentLag{
		Component:        component2,
		TechnicalLag:     technicalLag2,
		CriticalityScore: 0.8,
	}

	// Component with medium criticality score
	component3 := cdx.Component{
		Name:    "medium-criticality",
		Version: "3.0.0",
	}

	technicalLag3 := TechnicalLag{
		Libdays: 30.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 5,
		},
	}

	componentLag3 := ComponentLag{
		Component:        component3,
		TechnicalLag:     technicalLag3,
		CriticalityScore: 0.5,
	}

	// Update stats with all components
	mockBOM := createMockBOM()
	componentMetrics := map[cdx.Component]TechnicalLag{
		component1: technicalLag1,
		component2: technicalLag2,
		component3: technicalLag3,
	}
	updateTechLagStats(stats, technicalLag1, component1, componentLag1, mockBOM, componentMetrics)
	updateTechLagStats(stats, technicalLag2, component2, componentLag2, mockBOM, componentMetrics)
	updateTechLagStats(stats, technicalLag3, component3, componentLag3, mockBOM, componentMetrics)

	// Test that highest criticality score is tracked correctly
	if stats.HighestCriticalityScore != 0.8 {
		t.Errorf("Expected HighestCriticalityScore to be 0.8, got %f", stats.HighestCriticalityScore)
	}

	if stats.ComponentHighestCriticalityScore.Name != "high-criticality" {
		t.Errorf("Expected ComponentHighestCriticalityScore name to be 'high-criticality', got %s", stats.ComponentHighestCriticalityScore.Name)
	}

	// Verify other highest values for comparison
	if stats.HighestLibdays != 50.0 {
		t.Errorf("Expected HighestLibdays to be 50.0, got %f", stats.HighestLibdays)
	}

	if stats.ComponentHighestLibdays.Name != "high-criticality" {
		t.Errorf("Expected ComponentHighestLibdays name to be 'high-criticality', got %s", stats.ComponentHighestLibdays.Name)
	}

	if stats.HighestMissedReleases != 7 {
		t.Errorf("Expected HighestMissedReleases to be 7, got %d", stats.HighestMissedReleases)
	}

	if stats.ComponentHighestMissedReleases.Name != "high-criticality" {
		t.Errorf("Expected ComponentHighestMissedReleases name to be 'high-criticality', got %s", stats.ComponentHighestMissedReleases.Name)
	}

	// Test that all components were added
	if len(stats.Components) != 3 {
		t.Errorf("Expected 3 components in Components slice, got %d", len(stats.Components))
	}
}

func TestDependencyPathTracking(t *testing.T) {
	stats := &TechLagStats{}

	// Create a mock BOM with a dependency chain: direct-dep -> intermediate -> target
	directDep := cdx.Component{
		BOMRef:  "pkg:npm/direct-dep@1.0.0",
		Name:    "direct-dep",
		Version: "1.0.0",
		Scope:   cdx.ScopeRequired,
	}

	intermediate := cdx.Component{
		BOMRef:  "pkg:npm/intermediate@2.0.0",
		Name:    "intermediate",
		Version: "2.0.0",
		Scope:   cdx.ScopeRequired,
	}

	target := cdx.Component{
		BOMRef:  "pkg:npm/target@3.0.0",
		Name:    "target",
		Version: "3.0.0",
		Scope:   cdx.ScopeRequired,
	}

	mockBOM := &cdx.BOM{
		Metadata: &cdx.Metadata{
			Component: &cdx.Component{
				BOMRef: "pkg:npm/project@1.0.0",
				Name:   "test-project",
			},
		},
		Components: &[]cdx.Component{directDep, intermediate, target},
		Dependencies: &[]cdx.Dependency{
			{
				Ref:          "pkg:npm/project@1.0.0",
				Dependencies: &[]string{"pkg:npm/direct-dep@1.0.0"},
			},
			{
				Ref:          "pkg:npm/direct-dep@1.0.0",
				Dependencies: &[]string{"pkg:npm/intermediate@2.0.0"},
			},
			{
				Ref:          "pkg:npm/intermediate@2.0.0",
				Dependencies: &[]string{"pkg:npm/target@3.0.0"},
			},
			{
				Ref:          "pkg:npm/target@3.0.0",
				Dependencies: &[]string{},
			},
		},
	}

	// Create technical lag data for the target component (highest criticality)
	technicalLag := TechnicalLag{
		Libdays: 30.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 5,
			MissedMajor:    1,
			MissedMinor:    2,
			MissedPatch:    2,
		},
	}

	componentLag := ComponentLag{
		Component:        target,
		TechnicalLag:     technicalLag,
		CriticalityScore: 0.9, // Highest criticality score
	}

	// Update stats with the target component
	componentMetrics := map[cdx.Component]TechnicalLag{
		directDep:    {Libdays: 10.0},
		intermediate: {Libdays: 20.0},
		target:       technicalLag,
	}
	updateTechLagStats(stats, technicalLag, target, componentLag, mockBOM, componentMetrics)

	// Verify that the highest criticality component is set correctly
	if stats.ComponentHighestCriticalityScore.Name != "target" {
		t.Errorf("Expected ComponentHighestCriticalityScore name to be 'target', got %s", stats.ComponentHighestCriticalityScore.Name)
	}

	// Verify that the dependency path is populated and correct
	if stats.ComponentHighestCriticalityScorePath == nil {
		t.Fatalf("Expected ComponentHighestCriticalityScorePath to be populated, got nil")
	}

	expectedPathLength := 3 // direct-dep -> intermediate -> target
	if len(stats.ComponentHighestCriticalityScorePath) != expectedPathLength {
		t.Errorf("Expected dependency path length to be %d, got %d", expectedPathLength, len(stats.ComponentHighestCriticalityScorePath))
	}

	// Verify the path order: first element should be direct dependency, last should be target
	if len(stats.ComponentHighestCriticalityScorePath) >= 1 {
		firstComponent := stats.ComponentHighestCriticalityScorePath[0]
		if firstComponent.Component.Name != "direct-dep" {
			t.Errorf("Expected first component in path to be 'direct-dep', got %s", firstComponent.Component.Name)
		}
	}

	if len(stats.ComponentHighestCriticalityScorePath) >= 2 {
		secondComponent := stats.ComponentHighestCriticalityScorePath[1]
		if secondComponent.Component.Name != "intermediate" {
			t.Errorf("Expected second component in path to be 'intermediate', got %s", secondComponent.Component.Name)
		}
	}

	if len(stats.ComponentHighestCriticalityScorePath) >= 3 {
		lastComponent := stats.ComponentHighestCriticalityScorePath[2]
		if lastComponent.Component.Name != "target" {
			t.Errorf("Expected last component in path to be 'target', got %s", lastComponent.Component.Name)
		}
	}
}

func TestDependencyPathTrackingDirectDependency(t *testing.T) {
	stats := &TechLagStats{}

	// Create a mock BOM where the highest criticality component is a direct dependency
	directDep := cdx.Component{
		BOMRef:  "pkg:npm/direct-high-crit@1.0.0",
		Name:    "direct-high-crit",
		Version: "1.0.0",
		Scope:   cdx.ScopeRequired,
	}

	mockBOM := &cdx.BOM{
		Metadata: &cdx.Metadata{
			Component: &cdx.Component{
				BOMRef: "pkg:npm/project@1.0.0",
				Name:   "test-project",
			},
		},
		Components: &[]cdx.Component{directDep},
		Dependencies: &[]cdx.Dependency{
			{
				Ref:          "pkg:npm/project@1.0.0",
				Dependencies: &[]string{"pkg:npm/direct-high-crit@1.0.0"},
			},
			{
				Ref:          "pkg:npm/direct-high-crit@1.0.0",
				Dependencies: &[]string{},
			},
		},
	}

	technicalLag := TechnicalLag{
		Libdays: 20.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 3,
			MissedMajor:    1,
			MissedMinor:    1,
			MissedPatch:    1,
		},
	}

	componentLag := ComponentLag{
		Component:        directDep,
		TechnicalLag:     technicalLag,
		CriticalityScore: 0.8,
	}

	componentMetrics := map[cdx.Component]TechnicalLag{directDep: technicalLag}
	updateTechLagStats(stats, technicalLag, directDep, componentLag, mockBOM, componentMetrics)

	// Verify that the dependency path for a direct dependency contains only the direct dependency itself
	if stats.ComponentHighestCriticalityScorePath == nil {
		t.Fatalf("Expected ComponentHighestCriticalityScorePath to be populated, got nil")
	}

	if len(stats.ComponentHighestCriticalityScorePath) != 1 {
		t.Errorf("Expected dependency path length for direct dependency to be 1, got %d", len(stats.ComponentHighestCriticalityScorePath))
	}

	if len(stats.ComponentHighestCriticalityScorePath) >= 1 {
		pathComponent := stats.ComponentHighestCriticalityScorePath[0]
		if pathComponent.Component.Name != "direct-high-crit" {
			t.Errorf("Expected component in path to be 'direct-high-crit', got %s", pathComponent.Component.Name)
		}
	}
}

func TestDependencyPathContainsTechnicalLagData(t *testing.T) {
	stats := &TechLagStats{}

	// Create components with specific technical lag data
	directDep := cdx.Component{
		BOMRef:  "pkg:npm/direct-dep@1.0.0",
		Name:    "direct-dep",
		Version: "1.0.0",
		Scope:   cdx.ScopeRequired,
	}

	intermediate := cdx.Component{
		BOMRef:  "pkg:npm/intermediate@2.0.0",
		Name:    "intermediate",
		Version: "2.0.0",
		Scope:   cdx.ScopeRequired,
	}

	target := cdx.Component{
		BOMRef:  "pkg:npm/target@3.0.0",
		Name:    "target",
		Version: "3.0.0",
		Scope:   cdx.ScopeRequired,
	}

	// Create technical lag data for each component
	directDepLag := TechnicalLag{
		Libdays: 15.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 3,
			MissedMajor:    1,
			MissedMinor:    1,
			MissedPatch:    1,
		},
	}

	intermediateLag := TechnicalLag{
		Libdays: 25.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 4,
			MissedMajor:    1,
			MissedMinor:    2,
			MissedPatch:    1,
		},
	}

	targetLag := TechnicalLag{
		Libdays: 35.0,
		VersionDistance: semver.VersionDistance{
			MissedReleases: 6,
			MissedMajor:    2,
			MissedMinor:    2,
			MissedPatch:    2,
		},
	}

	mockBOM := &cdx.BOM{
		Metadata: &cdx.Metadata{
			Component: &cdx.Component{
				BOMRef: "pkg:npm/project@1.0.0",
				Name:   "test-project",
			},
		},
		Components: &[]cdx.Component{directDep, intermediate, target},
		Dependencies: &[]cdx.Dependency{
			{
				Ref:          "pkg:npm/project@1.0.0",
				Dependencies: &[]string{"pkg:npm/direct-dep@1.0.0"},
			},
			{
				Ref:          "pkg:npm/direct-dep@1.0.0",
				Dependencies: &[]string{"pkg:npm/intermediate@2.0.0"},
			},
			{
				Ref:          "pkg:npm/intermediate@2.0.0",
				Dependencies: &[]string{"pkg:npm/target@3.0.0"},
			},
			{
				Ref:          "pkg:npm/target@3.0.0",
				Dependencies: &[]string{},
			},
		},
	}

	componentMetrics := map[cdx.Component]TechnicalLag{
		directDep:    directDepLag,
		intermediate: intermediateLag,
		target:       targetLag,
	}

	targetComponentLag := ComponentLag{
		Component:        target,
		TechnicalLag:     targetLag,
		CriticalityScore: 0.9, // Highest criticality score
	}

	// Update stats with the target component
	updateTechLagStats(stats, targetLag, target, targetComponentLag, mockBOM, componentMetrics)

	// Verify that the dependency path is populated
	if stats.ComponentHighestCriticalityScorePath == nil {
		t.Fatalf("Expected ComponentHighestCriticalityScorePath to be populated, got nil")
	}

	expectedPathLength := 3
	if len(stats.ComponentHighestCriticalityScorePath) != expectedPathLength {
		t.Errorf("Expected dependency path length to be %d, got %d", expectedPathLength, len(stats.ComponentHighestCriticalityScorePath))
	}

	// Verify each ComponentLag in the path has correct technical lag data
	if len(stats.ComponentHighestCriticalityScorePath) >= 1 {
		directDepComponentLag := stats.ComponentHighestCriticalityScorePath[0]
		if directDepComponentLag.Component.Name != "direct-dep" {
			t.Errorf("Expected first component name to be 'direct-dep', got %s", directDepComponentLag.Component.Name)
		}
		if directDepComponentLag.TechnicalLag.Libdays != 15.0 {
			t.Errorf("Expected first component libdays to be 15.0, got %f", directDepComponentLag.TechnicalLag.Libdays)
		}
		if directDepComponentLag.TechnicalLag.VersionDistance.MissedReleases != 3 {
			t.Errorf("Expected first component missed releases to be 3, got %d", directDepComponentLag.TechnicalLag.VersionDistance.MissedReleases)
		}
		if directDepComponentLag.CriticalityScore <= 0 {
			t.Errorf("Expected first component to have a positive criticality score, got %f", directDepComponentLag.CriticalityScore)
		}
	}

	if len(stats.ComponentHighestCriticalityScorePath) >= 2 {
		intermediateComponentLag := stats.ComponentHighestCriticalityScorePath[1]
		if intermediateComponentLag.Component.Name != "intermediate" {
			t.Errorf("Expected second component name to be 'intermediate', got %s", intermediateComponentLag.Component.Name)
		}
		if intermediateComponentLag.TechnicalLag.Libdays != 25.0 {
			t.Errorf("Expected second component libdays to be 25.0, got %f", intermediateComponentLag.TechnicalLag.Libdays)
		}
		if intermediateComponentLag.TechnicalLag.VersionDistance.MissedReleases != 4 {
			t.Errorf("Expected second component missed releases to be 4, got %d", intermediateComponentLag.TechnicalLag.VersionDistance.MissedReleases)
		}
		if intermediateComponentLag.CriticalityScore <= 0 {
			t.Errorf("Expected second component to have a positive criticality score, got %f", intermediateComponentLag.CriticalityScore)
		}
	}

	if len(stats.ComponentHighestCriticalityScorePath) >= 3 {
		targetComponentLag := stats.ComponentHighestCriticalityScorePath[2]
		if targetComponentLag.Component.Name != "target" {
			t.Errorf("Expected third component name to be 'target', got %s", targetComponentLag.Component.Name)
		}
		if targetComponentLag.TechnicalLag.Libdays != 35.0 {
			t.Errorf("Expected third component libdays to be 35.0, got %f", targetComponentLag.TechnicalLag.Libdays)
		}
		if targetComponentLag.TechnicalLag.VersionDistance.MissedReleases != 6 {
			t.Errorf("Expected third component missed releases to be 6, got %d", targetComponentLag.TechnicalLag.VersionDistance.MissedReleases)
		}
		if targetComponentLag.CriticalityScore != 0.9 {
			t.Errorf("Expected third component criticality score to be 0.9, got %f", targetComponentLag.CriticalityScore)
		}
	}

	// Verify that ComponentLag convenience methods work on path elements
	if len(stats.ComponentHighestCriticalityScorePath) >= 1 {
		firstComponent := stats.ComponentHighestCriticalityScorePath[0]
		if firstComponent.Libdays() != 15.0 {
			t.Errorf("Expected first component Libdays() to return 15.0, got %f", firstComponent.Libdays())
		}
		if firstComponent.MissedReleases() != 3 {
			t.Errorf("Expected first component MissedReleases() to return 3, got %d", firstComponent.MissedReleases())
		}
	}
}
