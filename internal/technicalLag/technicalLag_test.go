package technicalLag

import (
	"sbom-technical-lag/internal/semver"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

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
		Component:      component,
		Libdays:        technicalLag.Libdays,
		MissedReleases: technicalLag.VersionDistance.MissedReleases,
		MissedMajor:    technicalLag.VersionDistance.MissedMajor,
		MissedMinor:    technicalLag.VersionDistance.MissedMinor,
		MissedPatch:    technicalLag.VersionDistance.MissedPatch,
	}

	updateTechLagStats(stats, technicalLag, component, componentLag)

	// Test that all values were properly added
	if stats.Libdays != 100.5 {
		t.Errorf("Expected Libdays to be 100.5, got %f", stats.Libdays)
	}

	if stats.MissedMajor != 2 {
		t.Errorf("Expected MissedMajor to be 2, got %d", stats.MissedMajor)
	}

	if stats.MissedMinor != 3 {
		t.Errorf("Expected MissedMinor to be 3, got %d", stats.MissedMinor)
	}

	if stats.MissedPatch != 5 {
		t.Errorf("Expected MissedPatch to be 5, got %d", stats.MissedPatch)
	}

	if stats.NumComponents != 1 {
		t.Errorf("Expected NumComponents to be 1, got %d", stats.NumComponents)
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
		Component:      component1,
		Libdays:        technicalLag1.Libdays,
		MissedReleases: technicalLag1.VersionDistance.MissedReleases,
		MissedMajor:    technicalLag1.VersionDistance.MissedMajor,
		MissedMinor:    technicalLag1.VersionDistance.MissedMinor,
		MissedPatch:    technicalLag1.VersionDistance.MissedPatch,
	}

	componentLag2 := ComponentLag{
		Component:      component2,
		Libdays:        technicalLag2.Libdays,
		MissedReleases: technicalLag2.VersionDistance.MissedReleases,
		MissedMajor:    technicalLag2.VersionDistance.MissedMajor,
		MissedMinor:    technicalLag2.VersionDistance.MissedMinor,
		MissedPatch:    technicalLag2.VersionDistance.MissedPatch,
	}

	// Update stats with both components
	updateTechLagStats(stats, technicalLag1, component1, componentLag1)
	updateTechLagStats(stats, technicalLag2, component2, componentLag2)

	// Test accumulated values
	if stats.Libdays != 125.5 {
		t.Errorf("Expected Libdays to be 125.5, got %f", stats.Libdays)
	}

	if stats.MissedMajor != 4 {
		t.Errorf("Expected MissedMajor to be 4, got %d", stats.MissedMajor)
	}

	if stats.MissedMinor != 6 {
		t.Errorf("Expected MissedMinor to be 6, got %d", stats.MissedMinor)
	}

	if stats.MissedPatch != 10 {
		t.Errorf("Expected MissedPatch to be 10, got %d", stats.MissedPatch)
	}

	if stats.NumComponents != 2 {
		t.Errorf("Expected NumComponents to be 2, got %d", stats.NumComponents)
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
		Libdays:         200.75,
		VersionDistance: versionDistance,
	}

	componentLag := ComponentLag{
		Component:      component,
		Libdays:        technicalLag.Libdays,
		MissedReleases: technicalLag.VersionDistance.MissedReleases,
		MissedMajor:    technicalLag.VersionDistance.MissedMajor,
		MissedMinor:    technicalLag.VersionDistance.MissedMinor,
		MissedPatch:    technicalLag.VersionDistance.MissedPatch,
	}

	if componentLag.Component.Name != "test-component" {
		t.Errorf("Expected component name to be 'test-component', got %s", componentLag.Component.Name)
	}

	if componentLag.Libdays != 200.75 {
		t.Errorf("Expected Libdays to be 200.75, got %f", componentLag.Libdays)
	}

	if componentLag.MissedMajor != 2 {
		t.Errorf("Expected MissedMajor to be 2, got %d", componentLag.MissedMajor)
	}

	if componentLag.MissedMinor != 3 {
		t.Errorf("Expected MissedMinor to be 3, got %d", componentLag.MissedMinor)
	}

	if componentLag.MissedPatch != 7 {
		t.Errorf("Expected MissedPatch to be 7, got %d", componentLag.MissedPatch)
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
		Component:      component,
		Libdays:        technicalLag.Libdays,
		MissedReleases: technicalLag.VersionDistance.MissedReleases,
		MissedMajor:    technicalLag.VersionDistance.MissedMajor,
		MissedMinor:    technicalLag.VersionDistance.MissedMinor,
		MissedPatch:    technicalLag.VersionDistance.MissedPatch,
	}

	updateTechLagStats(stats, technicalLag, component, componentLag)

	if stats.Libdays != 0.0 {
		t.Errorf("Expected Libdays to be 0.0, got %f", stats.Libdays)
	}

	if stats.MissedMajor != 0 {
		t.Errorf("Expected MissedMajor to be 0, got %d", stats.MissedMajor)
	}

	if stats.MissedMinor != 0 {
		t.Errorf("Expected MissedMinor to be 0, got %d", stats.MissedMinor)
	}

	if stats.MissedPatch != 0 {
		t.Errorf("Expected MissedPatch to be 0, got %d", stats.MissedPatch)
	}

	if stats.NumComponents != 1 {
		t.Errorf("Expected NumComponents to be 1, got %d", stats.NumComponents)
	}

	if stats.HighestLibdays != 0.0 {
		t.Errorf("Expected HighestLibdays to be 0.0, got %f", stats.HighestLibdays)
	}

	if stats.HighestMissedReleases != 0 {
		t.Errorf("Expected HighestMissedReleases to be 0, got %d", stats.HighestMissedReleases)
	}

	// Test that the component was added to the Components slice
	if len(stats.Components) != 1 {
		t.Errorf("Expected 1 component in Components slice, got %d", len(stats.Components))
	}
}

func TestResultStringFormat(t *testing.T) {
	result := Result{
		Production: TechLagStats{
			NumComponents:         2,
			Libdays:               150.5,
			MissedReleases:        15,
			MissedMajor:           3,
			MissedMinor:           5,
			MissedPatch:           7,
			HighestLibdays:        100.0,
			HighestMissedReleases: 10,
			Components:            make([]ComponentLag, 0),
		},
		Optional: TechLagStats{
			NumComponents:         1,
			Libdays:               50.25,
			MissedReleases:        6,
			MissedMajor:           1,
			MissedMinor:           2,
			MissedPatch:           3,
			HighestLibdays:        50.25,
			HighestMissedReleases: 6,
			Components:            make([]ComponentLag, 0),
		},
		DirectProduction: TechLagStats{
			NumComponents:         1,
			Libdays:               75.0,
			MissedReleases:        4,
			MissedMajor:           2,
			MissedMinor:           1,
			MissedPatch:           1,
			HighestLibdays:        75.0,
			HighestMissedReleases: 4,
			Components:            make([]ComponentLag, 0),
		},
		DirectOptional: TechLagStats{
			NumComponents:         0,
			Libdays:               0.0,
			MissedReleases:        0,
			MissedMajor:           0,
			MissedMinor:           0,
			MissedPatch:           0,
			HighestLibdays:        0.0,
			HighestMissedReleases: 0,
			Components:            make([]ComponentLag, 0),
		},
	}

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
	if !contains(output, "150.50") { // Prod libdays
		t.Error("Expected output to contain production libdays value")
	}

	if !contains(output, "50.25") { // Opt libdays
		t.Error("Expected output to contain optional libdays value")
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
			Component:      k,
			Libdays:        v.Libdays,
			MissedReleases: v.VersionDistance.MissedReleases,
			MissedMajor:    v.VersionDistance.MissedMajor,
			MissedMinor:    v.VersionDistance.MissedMinor,
			MissedPatch:    v.VersionDistance.MissedPatch,
		}

		if k.Scope == "" || k.Scope == "required" {
			updateTechLagStats(&result.Production, v, k, componentLag)
		} else {
			updateTechLagStats(&result.Optional, v, k, componentLag)
		}
	}

	// Process direct dependencies separately
	for _, dep := range directDeps {
		tl := cm[dep]
		componentLag := ComponentLag{
			Component:      dep,
			Libdays:        tl.Libdays,
			MissedReleases: tl.VersionDistance.MissedReleases,
			MissedMajor:    tl.VersionDistance.MissedMajor,
			MissedMinor:    tl.VersionDistance.MissedMinor,
			MissedPatch:    tl.VersionDistance.MissedPatch,
		}

		if dep.Scope == "" || dep.Scope == "required" {
			updateTechLagStats(&result.DirectProduction, tl, dep, componentLag)
		} else {
			updateTechLagStats(&result.DirectOptional, tl, dep, componentLag)
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
	if result.Production.NumComponents != 2 {
		t.Errorf("Expected Production.NumComponents to be 2, got %d", result.Production.NumComponents)
	}
	if result.Optional.NumComponents != 2 {
		t.Errorf("Expected Optional.NumComponents to be 2, got %d", result.Optional.NumComponents)
	}
	if result.DirectProduction.NumComponents != 1 {
		t.Errorf("Expected DirectProduction.NumComponents to be 1, got %d", result.DirectProduction.NumComponents)
	}
	if result.DirectOptional.NumComponents != 1 {
		t.Errorf("Expected DirectOptional.NumComponents to be 1, got %d", result.DirectOptional.NumComponents)
	}
}

// Helper function to check if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}
