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

	updateTechLagStats(stats, technicalLag.Libdays, technicalLag.VersionDistance, component)

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

	// Update stats with both components
	updateTechLagStats(stats, technicalLag1.Libdays, technicalLag1.VersionDistance, component1)
	updateTechLagStats(stats, technicalLag2.Libdays, technicalLag2.VersionDistance, component2)

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

	updateTechLagStats(stats, technicalLag.Libdays, technicalLag.VersionDistance, component)

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
}

func TestResultStringFormat(t *testing.T) {
	result := Result{
		Prod: TechLagStats{
			NumComponents:         2,
			Libdays:               150.5,
			MissedReleases:        15,
			MissedMajor:           3,
			MissedMinor:           5,
			MissedPatch:           7,
			HighestLibdays:        100.0,
			HighestMissedReleases: 10,
		},
		Opt: TechLagStats{
			NumComponents:         1,
			Libdays:               50.25,
			MissedReleases:        6,
			MissedMajor:           1,
			MissedMinor:           2,
			MissedPatch:           3,
			HighestLibdays:        50.25,
			HighestMissedReleases: 6,
		},
		DirectProd: TechLagStats{
			NumComponents:         1,
			Libdays:               75.0,
			MissedReleases:        4,
			MissedMajor:           2,
			MissedMinor:           1,
			MissedPatch:           1,
			HighestLibdays:        75.0,
			HighestMissedReleases: 4,
		},
		DirectOpt: TechLagStats{
			NumComponents:         0,
			Libdays:               0.0,
			MissedReleases:        0,
			MissedMajor:           0,
			MissedMinor:           0,
			MissedPatch:           0,
			HighestLibdays:        0.0,
			HighestMissedReleases: 0,
		},
	}

	output := result.String()

	// Just verify that the string contains expected sections and some key values
	if output == "" {
		t.Error("Expected non-empty string output")
	}

	// Check for section headers
	if !contains(output, "--- Overall ---") {
		t.Error("Expected output to contain '--- Overall ---'")
	}

	if !contains(output, "--- Direct ---") {
		t.Error("Expected output to contain '--- Direct ---'")
	}

	// Check for some key values (basic sanity check)
	if !contains(output, "150.50") { // Prod libdays
		t.Error("Expected output to contain production libdays value")
	}

	if !contains(output, "50.25") { // Opt libdays
		t.Error("Expected output to contain optional libdays value")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}
