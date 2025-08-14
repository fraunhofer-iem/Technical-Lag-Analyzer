package technicalLag

import (
	"context"
	"fmt"
	"log/slog"
	"sbom-technical-lag/internal/deps"
	"sbom-technical-lag/internal/sbom"
	"sbom-technical-lag/internal/semver"
	"sync"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

// TechnicalLag represents the technical lag metrics for a component
type TechnicalLag struct {
	Libdays         float64                `json:"libdays"`
	VersionDistance semver.VersionDistance `json:"versionDistance"`
}

// Calculator handles technical lag calculations
type Calculator struct {
	depsClient *deps.Client
	logger     *slog.Logger
	maxWorkers int
}

// NewCalculator creates a new technical lag calculator
func NewCalculator(logger *slog.Logger, maxWorkers int) *Calculator {
	if logger == nil {
		logger = slog.Default()
	}
	if maxWorkers <= 0 {
		maxWorkers = 10 // Default number of concurrent workers
	}

	return &Calculator{
		depsClient: deps.NewClient(logger),
		logger:     logger,
		maxWorkers: maxWorkers,
	}
}

// componentJob represents a job for processing a component
type componentJob struct {
	component cdx.Component
	index     int
}

// componentResult represents the result of processing a component
type componentResult struct {
	component cdx.Component
	lag       TechnicalLag
	index     int
	err       error
}

// Calculate computes technical lag metrics for all components in the SBOM
func (calc *Calculator) Calculate(ctx context.Context, bom *cdx.BOM) (map[cdx.Component]TechnicalLag, error) {
	if bom.Components == nil {
		return nil, fmt.Errorf("no components found in SBOM")
	}

	components := *bom.Components
	if len(components) == 0 {
		return make(map[cdx.Component]TechnicalLag), nil
	}

	calc.logger.Info("Starting technical lag calculation", "components", len(components), "workers", calc.maxWorkers)

	// Create channels for job distribution and result collection
	jobs := make(chan componentJob, len(components))
	results := make(chan componentResult, len(components))

	// Start worker goroutines
	var wg sync.WaitGroup
	for range calc.maxWorkers {
		wg.Add(1)
		go calc.worker(ctx, &wg, jobs, results)
	}

	// Send jobs to workers
	go func() {
		defer close(jobs)
		for i, component := range components {
			select {
			case jobs <- componentJob{component: component, index: i}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	componentToLag := make(map[cdx.Component]TechnicalLag, len(components))
	var errorCount int

	for result := range results {
		if result.err != nil {
			calc.logger.Warn("Failed to calculate lag for component",
				"component", result.component.Name,
				"purl", result.component.PackageURL,
				"error", result.err)
			errorCount++
			continue
		}
		componentToLag[result.component] = result.lag
	}

	if errorCount > 0 {
		calc.logger.Warn("Some components failed processing", "failed", errorCount, "total", len(components))
	}

	calc.logger.Info("Technical lag calculation completed",
		"processed", len(componentToLag),
		"failed", errorCount,
		"total", len(components))

	return componentToLag, nil
}

// worker processes component jobs concurrently
func (calc *Calculator) worker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan componentJob, results chan<- componentResult) {
	defer wg.Done()

	for {
		select {
		case job, ok := <-jobs:
			if !ok {
				return // Channel closed
			}
			lag, err := calc.calculateComponentLag(ctx, job.component)
			results <- componentResult{
				component: job.component,
				lag:       lag,
				index:     job.index,
				err:       err,
			}
		case <-ctx.Done():
			return // Context cancelled
		}
	}
}

// calculateComponentLag calculates technical lag for a single component
func (calc *Calculator) calculateComponentLag(ctx context.Context, component cdx.Component) (TechnicalLag, error) {
	if component.PackageURL == "" {
		return TechnicalLag{}, fmt.Errorf("component %s has no package URL", component.Name)
	}

	if component.Version == "" {
		return TechnicalLag{}, fmt.Errorf("component %s has no version", component.Name)
	}

	// Get versions from deps.dev API
	depsResp, err := calc.depsClient.GetVersions(ctx, component.PackageURL)
	if err != nil {
		return TechnicalLag{}, fmt.Errorf("failed to get versions for %s: %w", component.PackageURL, err)
	}

	if len(depsResp.Versions) == 0 {
		return TechnicalLag{}, fmt.Errorf("no versions found for component %s", component.Name)
	}

	// Convert API response to internal format
	versions := make([]deps.Version, 0, len(depsResp.Versions))
	rawVersions := make([]string, 0, len(depsResp.Versions))

	for _, v := range depsResp.Versions {
		version := v.Version
		// Handle cases where publication date is in the outer structure
		if version.PublishedAt == "" && v.PublishedAt != "" {
			version.PublishedAt = v.PublishedAt
		}
		versions = append(versions, version)
		rawVersions = append(rawVersions, version.Version)
	}

	// Calculate libyear (time-based lag)
	libduration, err := semver.GetLibyear(component.Version, versions)
	if err != nil {
		return TechnicalLag{}, fmt.Errorf("failed to calculate libyear for %s: %w", component.Name, err)
	}

	libdays := libduration.Hours() / 24

	// Calculate version distance (release-based lag)
	versionDistance, err := semver.GetVersionDistance(component.Version, rawVersions)
	if err != nil {
		return TechnicalLag{}, fmt.Errorf("failed to calculate version distance for %s: %w", component.Name, err)
	}

	return TechnicalLag{
		Libdays:         libdays,
		VersionDistance: *versionDistance,
	}, nil
}

// Calculate provides a convenient function using the default calculator
func Calculate(ctx context.Context, bom *cdx.BOM) (map[cdx.Component]TechnicalLag, error) {
	calc := NewCalculator(slog.Default(), 10)
	return calc.Calculate(ctx, bom)
}

// TechLagStats aggregates technical lag statistics
type TechLagStats struct {
	HighestLibdays                       float64        `json:"highestLibdays"`
	HighestMissedReleases                int64          `json:"highestMissedReleases"`
	HighestCriticalityScore              float64        `json:"highestCriticalityScore"`
	ComponentHighestMissedReleases       cdx.Component  `json:"componentHighestMissedReleases"`
	ComponentHighestLibdays              cdx.Component  `json:"componentHighestLibdays"`
	ComponentHighestCriticalityScore     cdx.Component  `json:"componentHighestCriticalityScore"`
	ComponentHighestCriticalityScorePath []ComponentLag `json:"componentHighestCriticalityScorePath"`
	Components                           []ComponentLag `json:"components"`
	// Computed fields for serialization
	TotalLibdays        float64 `json:"totalLibdays"`
	TotalMissedReleases int64   `json:"totalMissedReleases"`
	TotalMissedMajor    int64   `json:"totalMissedMajor"`
	TotalMissedMinor    int64   `json:"totalMissedMinor"`
	TotalMissedPatch    int64   `json:"totalMissedPatch"`
	TotalNumComponents  int     `json:"totalNumComponents"`
}

// ComponentLag represents technical lag for a single component
type ComponentLag struct {
	Component        cdx.Component `json:"component"`
	TechnicalLag     TechnicalLag  `json:"technicalLag"`
	CriticalityScore float64       `json:"criticalityScore"`
}

// Convenience getters for ComponentLag to maintain compatibility
func (cl ComponentLag) Libdays() float64 {
	return cl.TechnicalLag.Libdays
}

func (cl ComponentLag) MissedReleases() int64 {
	return cl.TechnicalLag.VersionDistance.MissedReleases
}

func (cl ComponentLag) MissedMajor() int64 {
	return cl.TechnicalLag.VersionDistance.MissedMajor
}

func (cl ComponentLag) MissedMinor() int64 {
	return cl.TechnicalLag.VersionDistance.MissedMinor
}

func (cl ComponentLag) MissedPatch() int64 {
	return cl.TechnicalLag.VersionDistance.MissedPatch
}

// computeTotals calculates totals from Components slice
func (stats *TechLagStats) computeTotals() {
	stats.TotalLibdays = 0
	stats.TotalMissedReleases = 0
	stats.TotalMissedMajor = 0
	stats.TotalMissedMinor = 0
	stats.TotalMissedPatch = 0
	stats.TotalNumComponents = len(stats.Components)

	for _, comp := range stats.Components {
		stats.TotalLibdays += comp.Libdays()
		stats.TotalMissedReleases += comp.MissedReleases()
		stats.TotalMissedMajor += comp.MissedMajor()
		stats.TotalMissedMinor += comp.MissedMinor()
		stats.TotalMissedPatch += comp.MissedPatch()
	}
}

// Result contains comprehensive technical lag analysis results
type Result struct {
	Production       TechLagStats `json:"production"`
	Optional         TechLagStats `json:"optional"`
	DirectProduction TechLagStats `json:"directProduction"`
	DirectOptional   TechLagStats `json:"directOptional"`
	Timestamp        int64        `json:"timestamp"`
}

// Summary provides high-level metrics across all categories - computed from Result
func (r Result) Summary() Summary {
	totalComponents := r.Production.TotalNumComponents + r.Optional.TotalNumComponents
	totalLibdays := r.Production.TotalLibdays + r.Optional.TotalLibdays
	totalMissedReleases := r.Production.TotalMissedReleases + r.Optional.TotalMissedReleases

	var avgLibdays, avgMissedReleases float64
	if totalComponents > 0 {
		avgLibdays = totalLibdays / float64(totalComponents)
		avgMissedReleases = float64(totalMissedReleases) / float64(totalComponents)
	}

	return Summary{
		TotalComponents:    totalComponents,
		TotalLibdays:       totalLibdays,
		TotalMissedRelease: totalMissedReleases,
		AvgLibdays:         avgLibdays,
		AvgMissedReleases:  avgMissedReleases,
	}
}

type Summary struct {
	TotalComponents    int     `json:"totalComponents"`
	TotalLibdays       float64 `json:"totalLibdays"`
	TotalMissedRelease int64   `json:"totalMissedReleases"`
	AvgLibdays         float64 `json:"avgLibdays"`
	AvgMissedReleases  float64 `json:"avgMissedReleases"`
}

// CreateResult generates a comprehensive result from component metrics
func CreateResult(bom *cdx.BOM, componentMetrics map[cdx.Component]TechnicalLag) (Result, error) {
	result := Result{
		Production:       TechLagStats{Components: make([]ComponentLag, 0)},
		Optional:         TechLagStats{Components: make([]ComponentLag, 0)},
		DirectProduction: TechLagStats{Components: make([]ComponentLag, 0)},
		DirectOptional:   TechLagStats{Components: make([]ComponentLag, 0)},
		Timestamp:        time.Now().Unix(),
	}

	// Calculate total scope libyears for criticality scores
	var totalProductionLibyears, totalOptionalLibyears float64
	for component, lag := range componentMetrics {
		if isProductionScope(component.Scope) {
			totalProductionLibyears += lag.Libdays
		} else {
			totalOptionalLibyears += lag.Libdays
		}
	}

	// Process all components
	for component, lag := range componentMetrics {
		var totalScopeLibyears float64
		if isProductionScope(component.Scope) {
			totalScopeLibyears = totalProductionLibyears
		} else {
			totalScopeLibyears = totalOptionalLibyears
		}

		criticalityScore := CalculateCriticalityScore(component, bom, componentMetrics, totalScopeLibyears)

		componentLag := ComponentLag{
			Component:        component,
			TechnicalLag:     lag,
			CriticalityScore: criticalityScore,
		}

		if isProductionScope(component.Scope) {
			updateTechLagStats(&result.Production, lag, component, componentLag, bom, componentMetrics)
		} else {
			updateTechLagStats(&result.Optional, lag, component, componentLag, bom, componentMetrics)
		}
	}

	// Calculate total direct scope libyears for criticality scores
	var totalDirectProductionLibyears, totalDirectOptionalLibyears float64
	directDeps, err := sbom.GetDirectDeps(bom)
	if err != nil {
		slog.Default().Warn("Failed to get direct dependencies", "error", err)
	} else {
		for _, dep := range directDeps {
			if lag, exists := componentMetrics[dep]; exists {
				if isProductionScope(dep.Scope) {
					totalDirectProductionLibyears += lag.Libdays
				} else {
					totalDirectOptionalLibyears += lag.Libdays
				}
			}
		}

		// Process direct dependencies
		for _, dep := range directDeps {
			if lag, exists := componentMetrics[dep]; exists {
				var totalScopeLibyears float64
				if isProductionScope(dep.Scope) {
					totalScopeLibyears = totalDirectProductionLibyears
				} else {
					totalScopeLibyears = totalDirectOptionalLibyears
				}

				criticalityScore := CalculateCriticalityScore(dep, bom, componentMetrics, totalScopeLibyears)

				componentLag := ComponentLag{
					Component:        dep,
					TechnicalLag:     lag,
					CriticalityScore: criticalityScore,
				}

				if isProductionScope(dep.Scope) {
					updateTechLagStats(&result.DirectProduction, lag, dep, componentLag, bom, componentMetrics)
				} else {
					updateTechLagStats(&result.DirectOptional, lag, dep, componentLag, bom, componentMetrics)
				}
			}
		}
	}

	// Finalize computed totals for serialization
	result.Production.computeTotals()
	result.Optional.computeTotals()
	result.DirectProduction.computeTotals()
	result.DirectOptional.computeTotals()

	return result, nil
}

// isProductionScope determines if a component scope is production-related
func isProductionScope(scope cdx.Scope) bool {
	return scope == "" || scope == "required" || scope == "runtime"
}

// updateTechLagStats updates aggregate statistics with component data
func updateTechLagStats(stats *TechLagStats, lag TechnicalLag, component cdx.Component, componentLag ComponentLag, bom *cdx.BOM, componentMetrics map[cdx.Component]TechnicalLag) {
	stats.Components = append(stats.Components, componentLag)

	// Update computed totals
	stats.TotalLibdays += componentLag.Libdays()
	stats.TotalMissedReleases += componentLag.MissedReleases()
	stats.TotalMissedMajor += componentLag.MissedMajor()
	stats.TotalMissedMinor += componentLag.MissedMinor()
	stats.TotalMissedPatch += componentLag.MissedPatch()
	stats.TotalNumComponents++

	if lag.VersionDistance.MissedReleases > stats.HighestMissedReleases {
		stats.HighestMissedReleases = lag.VersionDistance.MissedReleases
		stats.ComponentHighestMissedReleases = component
	}
	if lag.Libdays > stats.HighestLibdays {
		stats.HighestLibdays = lag.Libdays
		stats.ComponentHighestLibdays = component
	}
	if componentLag.CriticalityScore > stats.HighestCriticalityScore {
		stats.HighestCriticalityScore = componentLag.CriticalityScore
		stats.ComponentHighestCriticalityScore = component

		// Find and store the dependency path to this component
		componentPath, err := sbom.GetDependencyPath(bom, component.BOMRef)
		if err != nil {
			slog.Default().Warn("Failed to get dependency path for highest criticality component",
				"component", component.Name, "ref", component.BOMRef, "error", err)
			stats.ComponentHighestCriticalityScorePath = nil
		} else if componentPath != nil {
			// Convert component path to ComponentLag path
			var componentLagPath []ComponentLag
			for _, pathComponent := range componentPath {
				if pathLag, exists := componentMetrics[pathComponent]; exists {
					var criticalityScore float64
					// If this is the target component (the one with highest criticality), use the original score
					if pathComponent.BOMRef == component.BOMRef {
						criticalityScore = componentLag.CriticalityScore
					} else {
						// Calculate criticality score for path component
						var totalScopeLibyears float64
						for comp, compLag := range componentMetrics {
							if isProductionScope(comp.Scope) == isProductionScope(pathComponent.Scope) {
								totalScopeLibyears += compLag.Libdays
							}
						}
						criticalityScore = CalculateCriticalityScore(pathComponent, bom, componentMetrics, totalScopeLibyears)
					}

					pathComponentLag := ComponentLag{
						Component:        pathComponent,
						TechnicalLag:     pathLag,
						CriticalityScore: criticalityScore,
					}
					componentLagPath = append(componentLagPath, pathComponentLag)
				}
			}
			stats.ComponentHighestCriticalityScorePath = componentLagPath
			slog.Default().Debug("Found dependency path for highest criticality component",
				"component", component.Name, "path_length", len(componentLagPath))
		} else {
			stats.ComponentHighestCriticalityScorePath = nil
		}
	}
}

// String returns a formatted string representation of the results
func (r *Result) String() string {
	const (
		intFormat   = "%-25s prod: %-10d opt: %-10d direct prod: %-10d direct opt: %d\n"
		floatFormat = "%-25s prod: %-10.2f opt: %-10.2f direct prod: %-10.2f direct opt: %.2f\n"
	)

	output := fmt.Sprintf(
		"=== Technical Lag Analysis ===\n"+
			intFormat+ // NumComponents
			floatFormat+ // Libdays
			intFormat+ // MissedReleases
			intFormat+ // MissedMajor
			intFormat+ // MissedMinor
			intFormat+ // MissedPatch
			"\n=== Summary ===\n"+
			"Total components: %d\n"+
			"Total libdays: %.2f\n"+
			"Total missed releases: %d\n"+
			"Average libdays per component: %.2f\n"+
			"Average missed releases per component: %.2f\n",

		// Main metrics
		"Components", r.Production.TotalNumComponents, r.Optional.TotalNumComponents, r.DirectProduction.TotalNumComponents, r.DirectOptional.TotalNumComponents,
		"Libdays", r.Production.TotalLibdays, r.Optional.TotalLibdays, r.DirectProduction.TotalLibdays, r.DirectOptional.TotalLibdays,
		"Missed releases", r.Production.TotalMissedReleases, r.Optional.TotalMissedReleases, r.DirectProduction.TotalMissedReleases, r.DirectOptional.TotalMissedReleases,
		"Missed major", r.Production.TotalMissedMajor, r.Optional.TotalMissedMajor, r.DirectProduction.TotalMissedMajor, r.DirectOptional.TotalMissedMajor,
		"Missed minor", r.Production.TotalMissedMinor, r.Optional.TotalMissedMinor, r.DirectProduction.TotalMissedMinor, r.DirectOptional.TotalMissedMinor,
		"Missed patch", r.Production.TotalMissedPatch, r.Optional.TotalMissedPatch, r.DirectProduction.TotalMissedPatch, r.DirectOptional.TotalMissedPatch,

		// Summary
		r.Summary().TotalComponents,
		r.Summary().TotalLibdays,
		r.Summary().TotalMissedRelease,
		r.Summary().AvgLibdays,
		r.Summary().AvgMissedReleases,
	)

	return output
}

// CalculateCriticalityScore calculates the criticality score for a component
// criticality_score = Sum(libyears of all direct dependencies) / libyears of whole scope
func CalculateCriticalityScore(component cdx.Component, bom *cdx.BOM, componentMetrics map[cdx.Component]TechnicalLag, totalScopeLibyears float64) float64 {
	if totalScopeLibyears == 0 {
		return 0.0
	}

	// Get direct dependencies of this component
	directDeps, err := sbom.GetDirectDependenciesOf(bom, component.BOMRef)
	if err != nil {
		slog.Default().Debug("Failed to get direct dependencies for criticality score",
			"component", component.Name,
			"error", err)
		return 0.0
	}

	// Sum up libyears of all direct dependencies
	var sumDirectDepsLibyears float64
	for _, dep := range directDeps {
		if metrics, exists := componentMetrics[dep]; exists {
			sumDirectDepsLibyears += metrics.Libdays
		}
	}

	criticalityScore := sumDirectDepsLibyears / totalScopeLibyears

	slog.Default().Debug("Calculated criticality score",
		"component", component.Name,
		"direct_deps_count", len(directDeps),
		"sum_direct_deps_libyears", sumDirectDepsLibyears,
		"total_scope_libyears", totalScopeLibyears,
		"criticality_score", criticalityScore)

	return criticalityScore
}
