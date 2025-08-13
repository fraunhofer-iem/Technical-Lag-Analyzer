package sbom

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

var (
	// ErrNoMetadata is returned when the SBOM has no metadata
	ErrNoMetadata = errors.New("SBOM metadata is missing")
	// ErrNoProjectComponent is returned when the SBOM has no project component in metadata
	ErrNoProjectComponent = errors.New("SBOM metadata has no project component")
	// ErrEmptyProjectRef is returned when the project component has no BOM reference
	ErrEmptyProjectRef = errors.New("project component has no BOM reference")
	// ErrNoDependencies is returned when the SBOM has no dependencies section
	ErrNoDependencies = errors.New("SBOM has no dependencies section")
	// ErrNoComponents is returned when the SBOM has no components section
	ErrNoComponents = errors.New("SBOM has no components section")
	// ErrProjectNotFound is returned when the project reference is not found in dependencies
	ErrProjectNotFound = errors.New("project reference not found in dependencies")
	// ErrNoDirectDependencies is returned when no direct dependencies are found for the project
	ErrNoDirectDependencies = errors.New("no direct dependencies found for project")
)

// ValidateBOM performs basic validation on a CycloneDX BOM structure
func ValidateBOM(bom *cdx.BOM) error {
	if bom == nil {
		return fmt.Errorf("BOM is nil")
	}

	if bom.Metadata == nil {
		return ErrNoMetadata
	}

	if bom.Metadata.Component == nil {
		return ErrNoProjectComponent
	}

	if bom.Metadata.Component.BOMRef == "" {
		return ErrEmptyProjectRef
	}

	if bom.Dependencies == nil {
		return ErrNoDependencies
	}

	if bom.Components == nil {
		return ErrNoComponents
	}

	return nil
}

// GetDirectDeps extracts direct dependencies of the main project from a CycloneDX BOM
func GetDirectDeps(bom *cdx.BOM) ([]cdx.Component, error) {
	if err := ValidateBOM(bom); err != nil {
		return nil, fmt.Errorf("invalid BOM: %w", err)
	}

	projectRef := bom.Metadata.Component.BOMRef
	logger := slog.Default()

	logger.Debug("Looking for direct dependencies", "project_ref", projectRef)

	dependencies := *bom.Dependencies
	components := *bom.Components

	// Find the project's dependency entry
	projectDep, err := findProjectDependency(dependencies, projectRef)
	if err != nil {
		return nil, err
	}

	if projectDep.Dependencies == nil {
		logger.Debug("Project has no direct dependencies", "project_ref", projectRef)
		return []cdx.Component{}, ErrNoDirectDependencies
	}

	// Create a set of direct dependency references for efficient lookup
	directDepRefs := make(map[string]struct{}, len(*projectDep.Dependencies))
	for _, depRef := range *projectDep.Dependencies {
		directDepRefs[depRef] = struct{}{}
	}

	logger.Debug("Found direct dependency references", "count", len(directDepRefs))

	// Find components that match the direct dependency references
	directDeps := make([]cdx.Component, 0, len(directDepRefs))
	foundRefs := make(map[string]bool, len(directDepRefs))

	for _, component := range components {
		if _, isDirect := directDepRefs[component.BOMRef]; isDirect {
			directDeps = append(directDeps, component)
			foundRefs[component.BOMRef] = true
			logger.Debug("Found direct dependency component",
				"name", component.Name,
				"version", component.Version,
				"ref", component.BOMRef)
		}
	}

	// Log any missing components (references in dependencies but not in components)
	for ref := range directDepRefs {
		if !foundRefs[ref] {
			logger.Warn("Direct dependency reference not found in components", "ref", ref)
		}
	}

	logger.Info("Retrieved direct dependencies",
		"project", bom.Metadata.Component.Name,
		"version", bom.Metadata.Component.Version,
		"direct_deps", len(directDeps),
		"missing_refs", len(directDepRefs)-len(foundRefs))

	return directDeps, nil
}

// findProjectDependency locates the dependency entry for the main project
func findProjectDependency(dependencies []cdx.Dependency, projectRef string) (*cdx.Dependency, error) {
	for i, dep := range dependencies {
		if dep.Ref == projectRef {
			slog.Default().Debug("Found project dependency entry", "ref", projectRef, "index", i)
			return &dependencies[i], nil
		}
	}

	slog.Default().Debug("Project dependency entry not found", "ref", projectRef, "total_deps", len(dependencies))
	return nil, ErrProjectNotFound
}

// GetAllComponents returns all components from the SBOM with basic validation
func GetAllComponents(bom *cdx.BOM) ([]cdx.Component, error) {
	if bom == nil {
		return nil, fmt.Errorf("BOM is nil")
	}

	if bom.Components == nil {
		return nil, ErrNoComponents
	}

	components := *bom.Components
	slog.Default().Debug("Retrieved all components", "count", len(components))

	return components, nil
}

// ComponentStats provides statistics about components in the SBOM
type ComponentStats struct {
	Total      int            `json:"total"`
	ByType     map[string]int `json:"byType"`
	ByScope    map[string]int `json:"byScope"`
	WithPURL   int            `json:"withPURL"`
	WithoutURL int            `json:"withoutPURL"`
}

// GetComponentStats analyzes components and returns statistics
func GetComponentStats(bom *cdx.BOM) (*ComponentStats, error) {
	components, err := GetAllComponents(bom)
	if err != nil {
		return nil, err
	}

	stats := &ComponentStats{
		Total:   len(components),
		ByType:  make(map[string]int),
		ByScope: make(map[string]int),
	}

	for _, comp := range components {
		// Count by type
		compType := string(comp.Type)
		if compType == "" {
			compType = "unknown"
		}
		stats.ByType[compType]++

		// Count by scope
		scope := string(comp.Scope)
		if scope == "" {
			scope = "unspecified"
		}
		stats.ByScope[scope]++

		// Count PURL presence
		if comp.PackageURL != "" {
			stats.WithPURL++
		} else {
			stats.WithoutURL++
		}
	}

	slog.Default().Debug("Generated component statistics",
		"total", stats.Total,
		"types", len(stats.ByType),
		"scopes", len(stats.ByScope),
		"with_purl", stats.WithPURL,
		"without_purl", stats.WithoutURL)

	return stats, nil
}

// FilterComponentsByScope returns components filtered by their scope
func FilterComponentsByScope(bom *cdx.BOM, scopes ...string) ([]cdx.Component, error) {
	components, err := GetAllComponents(bom)
	if err != nil {
		return nil, err
	}

	if len(scopes) == 0 {
		return components, nil
	}

	// Create scope lookup map
	scopeSet := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scopeSet[scope] = struct{}{}
	}

	var filtered []cdx.Component
	for _, comp := range components {
		scope := string(comp.Scope)
		if scope == "" {
			scope = "unspecified"
		}

		if _, matches := scopeSet[scope]; matches {
			filtered = append(filtered, comp)
		}
	}

	slog.Default().Debug("Filtered components by scope",
		"requested_scopes", scopes,
		"total_components", len(components),
		"filtered_count", len(filtered))

	return filtered, nil
}

// FindComponentByRef finds a component by its BOM reference
func FindComponentByRef(bom *cdx.BOM, ref string) (*cdx.Component, error) {
	if ref == "" {
		return nil, fmt.Errorf("empty reference provided")
	}

	components, err := GetAllComponents(bom)
	if err != nil {
		return nil, err
	}

	for i, comp := range components {
		if comp.BOMRef == ref {
			slog.Default().Debug("Found component by reference", "ref", ref, "name", comp.Name)
			return &components[i], nil
		}
	}

	slog.Default().Debug("Component not found by reference", "ref", ref)
	return nil, fmt.Errorf("component with reference %q not found", ref)
}

// FindComponentsByName finds all components with the given name
func FindComponentsByName(bom *cdx.BOM, name string) ([]cdx.Component, error) {
	if name == "" {
		return nil, fmt.Errorf("empty name provided")
	}

	components, err := GetAllComponents(bom)
	if err != nil {
		return nil, err
	}

	var matches []cdx.Component
	for _, comp := range components {
		if comp.Name == name {
			matches = append(matches, comp)
		}
	}

	slog.Default().Debug("Found components by name",
		"name", name,
		"matches", len(matches))

	return matches, nil
}

// GetComponentDependencies returns all components that depend on the given component reference
func GetComponentDependencies(bom *cdx.BOM, componentRef string) ([]cdx.Component, error) {
	if err := ValidateBOM(bom); err != nil {
		return nil, fmt.Errorf("invalid BOM: %w", err)
	}

	if componentRef == "" {
		return nil, fmt.Errorf("empty component reference provided")
	}

	dependencies := *bom.Dependencies
	components := *bom.Components

	// Find all dependencies that depend on the given component
	var dependentRefs []string
	for _, dep := range dependencies {
		if dep.Dependencies != nil {
			if slices.Contains(*dep.Dependencies, componentRef) {
				dependentRefs = append(dependentRefs, dep.Ref)
			}
		}
	}

	// Convert references to actual components
	var dependentComponents []cdx.Component
	for _, ref := range dependentRefs {
		for _, comp := range components {
			if comp.BOMRef == ref {
				dependentComponents = append(dependentComponents, comp)
				break
			}
		}
	}

	slog.Default().Debug("Found component dependencies",
		"component_ref", componentRef,
		"dependent_count", len(dependentComponents))

	return dependentComponents, nil
}

// GetDirectDependenciesOf returns the direct dependencies of a specific component
func GetDirectDependenciesOf(bom *cdx.BOM, componentRef string) ([]cdx.Component, error) {
	if err := ValidateBOM(bom); err != nil {
		return nil, fmt.Errorf("invalid BOM: %w", err)
	}

	if componentRef == "" {
		return nil, fmt.Errorf("empty component reference provided")
	}

	dependencies := *bom.Dependencies
	components := *bom.Components

	// Find the dependency entry for this component
	var componentDep *cdx.Dependency
	for i, dep := range dependencies {
		if dep.Ref == componentRef {
			componentDep = &dependencies[i]
			break
		}
	}

	if componentDep == nil {
		slog.Default().Debug("No dependency entry found for component", "ref", componentRef)
		return []cdx.Component{}, nil
	}

	if componentDep.Dependencies == nil {
		slog.Default().Debug("Component has no direct dependencies", "ref", componentRef)
		return []cdx.Component{}, nil
	}

	// Convert dependency references to actual components
	var directDeps []cdx.Component
	for _, depRef := range *componentDep.Dependencies {
		for _, comp := range components {
			if comp.BOMRef == depRef {
				directDeps = append(directDeps, comp)
				break
			}
		}
	}

	slog.Default().Debug("Found direct dependencies of component",
		"component_ref", componentRef,
		"direct_deps_count", len(directDeps))

	return directDeps, nil
}
