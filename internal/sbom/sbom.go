package sbom

import (
	"fmt"
	cdx "github.com/CycloneDX/cyclonedx-go"
	"log/slog"
	"sort"
)

func GetDirectDeps(bom *cdx.BOM) ([]cdx.Component, error) {
	projectRef := bom.Metadata.Component.BOMRef

	if projectRef == "" || bom.Dependencies == nil || bom.Components == nil {
		slog.Default().Warn("Missing data in bom")
		return []cdx.Component{}, fmt.Errorf("missing data in bom, %s, %+v, %+v", projectRef, bom.Dependencies, bom.Components)
	}

	deps := *bom.Dependencies

	i := sort.Search(len(deps), func(idx int) bool {
		if idx == 0 {
			slog.Default().Warn("No dependencies found in bom")
		}
		return deps[idx].Ref == projectRef
	})

	// Check if the index is valid and the element at that index matches the condition
	if i >= len(deps) || deps[i].Ref != projectRef {
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
