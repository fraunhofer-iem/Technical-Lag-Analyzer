package sbom

import (
	"fmt"
	"log/slog"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

func GetDirectDeps(bom *cdx.BOM) ([]cdx.Component, error) {
	projectRef := bom.Metadata.Component.BOMRef

	if projectRef == "" || bom.Dependencies == nil || bom.Components == nil {
		slog.Default().Warn("Missing data in bom")
		return []cdx.Component{}, fmt.Errorf("missing data in bom, %s, %+v, %+v", projectRef, bom.Dependencies, bom.Components)
	}

	deps := *bom.Dependencies

	var i int
	for i = 0; i < len(deps); i++ {
		if deps[i].Ref == projectRef {
			break
		}
	}

	// Check if the index is valid and the element at that index matches the condition
	if i >= len(deps) {
		slog.Default().Warn("Project reference not found in dependencies", "ref", projectRef)
		return []cdx.Component{}, fmt.Errorf("project reference not found in dependencies")
	}

	if deps[i].Dependencies == nil {
		slog.Default().Warn("No direct dependencies found in bom")
		return []cdx.Component{}, fmt.Errorf("no direct dependencies found in bom")
	}

	directDepsRef := make(map[string]struct{})
	for _, dep := range *deps[i].Dependencies {
		directDepsRef[dep] = struct{}{}
	}

	components := *bom.Components

	directDeps := make([]cdx.Component, 0, len(directDepsRef))
	for _, c := range components {
		if _, found := directDepsRef[c.BOMRef]; found {
			directDeps = append(directDeps, c)
		}
	}

	return directDeps, nil
}
