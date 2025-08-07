package sbom

import (
	"encoding/json"
	"os"
	"testing"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

func TestGetDirectDeps(t *testing.T) {
	// Load the SBOM from examples/sbom.json
	sbomFile, err := os.ReadFile("../../examples/sbom-npm-vuejs.json")
	if err != nil {
		t.Fatalf("Failed to read SBOM file: %v", err)
	}

	// Parse the SBOM
	var bom cdx.BOM
	err = json.Unmarshal(sbomFile, &bom)
	if err != nil {
		t.Fatalf("Failed to parse SBOM: %v", err)
	}

	// Get direct dependencies
	directDeps, err := GetDirectDeps(&bom)
	if err != nil {
		t.Fatalf("GetDirectDeps failed: %v", err)
	}

	// Verify the number of direct dependencies
	// The main project "spha-visualization@0.0.0" has 14 direct dependencies
	if len(directDeps) != 14 {
		t.Fatalf("Expected 14 direct dependencies, got %d", len(directDeps))
	}

	// Verify some of the direct dependencies are present
	expectedDeps := []string{
		"@popperjs/core@2.11.8",
		"bootstrap@5.3.7",
		"vue@3.5.17",
		"typescript@5.8.3",
		"vite@7.0.0",
	}

	// Create a map of found dependencies for easier lookup
	foundDeps := make(map[string]bool)
	for _, dep := range directDeps {
		foundDeps[dep.BOMRef] = true
	}

	// Check that each expected dependency is found
	for _, expectedDep := range expectedDeps {
		if !foundDeps[expectedDep] {
			t.Fatalf("Expected dependency not found: %s", expectedDep)
		}
	}
}
