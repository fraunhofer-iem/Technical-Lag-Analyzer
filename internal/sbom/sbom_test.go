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

func TestGetDependencyPath(t *testing.T) {
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

	// Test finding path to target component
	path, err := GetDependencyPath(mockBOM, "pkg:npm/target@3.0.0")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if path == nil {
		t.Fatalf("Expected path to be found, got nil")
	}

	expectedPathLength := 3 // direct-dep -> intermediate -> target
	if len(path) != expectedPathLength {
		t.Errorf("Expected path length to be %d, got %d", expectedPathLength, len(path))
	}

	// Verify path order
	if len(path) >= 1 && path[0].Name != "direct-dep" {
		t.Errorf("Expected first component in path to be 'direct-dep', got %s", path[0].Name)
	}
	if len(path) >= 2 && path[1].Name != "intermediate" {
		t.Errorf("Expected second component in path to be 'intermediate', got %s", path[1].Name)
	}
	if len(path) >= 3 && path[2].Name != "target" {
		t.Errorf("Expected third component in path to be 'target', got %s", path[2].Name)
	}
}

func TestGetDependencyPathDirectDependency(t *testing.T) {
	// Test when target is a direct dependency
	directDep := cdx.Component{
		BOMRef:  "pkg:npm/direct-dep@1.0.0",
		Name:    "direct-dep",
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
				Dependencies: &[]string{"pkg:npm/direct-dep@1.0.0"},
			},
			{
				Ref:          "pkg:npm/direct-dep@1.0.0",
				Dependencies: &[]string{},
			},
		},
	}

	path, err := GetDependencyPath(mockBOM, "pkg:npm/direct-dep@1.0.0")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if path == nil {
		t.Fatalf("Expected path to be found, got nil")
	}

	if len(path) != 1 {
		t.Errorf("Expected path length for direct dependency to be 1, got %d", len(path))
	}

	if len(path) >= 1 && path[0].Name != "direct-dep" {
		t.Errorf("Expected component in path to be 'direct-dep', got %s", path[0].Name)
	}
}

func TestGetDependencyPathNotFound(t *testing.T) {
	// Test when no path exists to target
	directDep := cdx.Component{
		BOMRef:  "pkg:npm/direct-dep@1.0.0",
		Name:    "direct-dep",
		Version: "1.0.0",
		Scope:   cdx.ScopeRequired,
	}

	orphanComponent := cdx.Component{
		BOMRef:  "pkg:npm/orphan@1.0.0",
		Name:    "orphan",
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
		Components: &[]cdx.Component{directDep, orphanComponent},
		Dependencies: &[]cdx.Dependency{
			{
				Ref:          "pkg:npm/project@1.0.0",
				Dependencies: &[]string{"pkg:npm/direct-dep@1.0.0"},
			},
			{
				Ref:          "pkg:npm/direct-dep@1.0.0",
				Dependencies: &[]string{},
			},
			// orphan component has no dependencies and is not reachable
		},
	}

	path, err := GetDependencyPath(mockBOM, "pkg:npm/orphan@1.0.0")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if path != nil {
		t.Errorf("Expected no path to be found for orphan component, got path with length %d", len(path))
	}
}

func TestGetDependencyPathErrors(t *testing.T) {
	// Test with nil BOM
	_, err := GetDependencyPath(nil, "pkg:npm/target@1.0.0")
	if err == nil {
		t.Error("Expected error for nil BOM, got nil")
	}

	// Test with empty target reference
	validBOM := &cdx.BOM{
		Metadata: &cdx.Metadata{
			Component: &cdx.Component{
				BOMRef: "pkg:npm/project@1.0.0",
				Name:   "test-project",
			},
		},
		Components:   &[]cdx.Component{},
		Dependencies: &[]cdx.Dependency{},
	}

	_, err = GetDependencyPath(validBOM, "")
	if err == nil {
		t.Error("Expected error for empty target reference, got nil")
	}

	// Test with invalid BOM (missing metadata)
	invalidBOM := &cdx.BOM{}
	_, err = GetDependencyPath(invalidBOM, "pkg:npm/target@1.0.0")
	if err == nil {
		t.Error("Expected error for invalid BOM, got nil")
	}
}
