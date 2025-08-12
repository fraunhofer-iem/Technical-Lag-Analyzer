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
	for i := 0; i < calc.maxWorkers; i++ {
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
	Libdays                        float64        `json:"libdays"`
	MissedReleases                 int64          `json:"missedReleases"`
	MissedMajor                    int64          `json:"missedMajor"`
	MissedMinor                    int64          `json:"missedMinor"`
	MissedPatch                    int64          `json:"missedPatch"`
	NumComponents                  int            `json:"numComponents"`
	HighestLibdays                 float64        `json:"highestLibdays"`
	HighestMissedReleases          int64          `json:"highestMissedReleases"`
	ComponentHighestMissedReleases cdx.Component  `json:"componentHighestMissedReleases"`
	ComponentHighestLibdays        cdx.Component  `json:"componentHighestLibdays"`
	Components                     []ComponentLag `json:"components"`
}

// ComponentLag represents technical lag for a single component
type ComponentLag struct {
	Component      cdx.Component `json:"component"`
	Libdays        float64       `json:"libdays"`
	MissedReleases int64         `json:"missedReleases"`
	MissedMajor    int64         `json:"missedMajor"`
	MissedMinor    int64         `json:"missedMinor"`
	MissedPatch    int64         `json:"missedPatch"`
}

// Result contains comprehensive technical lag analysis results
type Result struct {
	Production       TechLagStats `json:"production"`
	Optional         TechLagStats `json:"optional"`
	DirectProduction TechLagStats `json:"directProduction"`
	DirectOptional   TechLagStats `json:"directOptional"`
	Timestamp        int64        `json:"timestamp"`
	Summary          Summary      `json:"summary"`
}

// Summary provides high-level metrics across all categories
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

	// Process all components
	for component, lag := range componentMetrics {
		componentLag := ComponentLag{
			Component:      component,
			Libdays:        lag.Libdays,
			MissedReleases: lag.VersionDistance.MissedReleases,
			MissedMajor:    lag.VersionDistance.MissedMajor,
			MissedMinor:    lag.VersionDistance.MissedMinor,
			MissedPatch:    lag.VersionDistance.MissedPatch,
		}

		if isProductionScope(component.Scope) {
			updateTechLagStats(&result.Production, lag, component, componentLag)
		} else {
			updateTechLagStats(&result.Optional, lag, component, componentLag)
		}
	}

	// Process direct dependencies
	directDeps, err := sbom.GetDirectDeps(bom)
	if err != nil {
		slog.Default().Warn("Failed to get direct dependencies", "error", err)
	} else {
		for _, dep := range directDeps {
			if lag, exists := componentMetrics[dep]; exists {
				componentLag := ComponentLag{
					Component:      dep,
					Libdays:        lag.Libdays,
					MissedReleases: lag.VersionDistance.MissedReleases,
					MissedMajor:    lag.VersionDistance.MissedMajor,
					MissedMinor:    lag.VersionDistance.MissedMinor,
					MissedPatch:    lag.VersionDistance.MissedPatch,
				}

				if isProductionScope(dep.Scope) {
					updateTechLagStats(&result.DirectProduction, lag, dep, componentLag)
				} else {
					updateTechLagStats(&result.DirectOptional, lag, dep, componentLag)
				}
			}
		}
	}

	// Calculate summary
	result.Summary = calculateSummary(result)

	return result, nil
}

// isProductionScope determines if a component scope is production-related
func isProductionScope(scope cdx.Scope) bool {
	return scope == "" || scope == "required" || scope == "runtime"
}

// updateTechLagStats updates aggregate statistics with component data
func updateTechLagStats(stats *TechLagStats, lag TechnicalLag, component cdx.Component, componentLag ComponentLag) {
	stats.Libdays += lag.Libdays
	stats.MissedReleases += lag.VersionDistance.MissedReleases
	stats.MissedMajor += lag.VersionDistance.MissedMajor
	stats.MissedMinor += lag.VersionDistance.MissedMinor
	stats.MissedPatch += lag.VersionDistance.MissedPatch
	stats.NumComponents++
	stats.Components = append(stats.Components, componentLag)

	if lag.VersionDistance.MissedReleases > stats.HighestMissedReleases {
		stats.HighestMissedReleases = lag.VersionDistance.MissedReleases
		stats.ComponentHighestMissedReleases = component
	}
	if lag.Libdays > stats.HighestLibdays {
		stats.HighestLibdays = lag.Libdays
		stats.ComponentHighestLibdays = component
	}
}

// calculateSummary computes summary statistics across all categories
func calculateSummary(result Result) Summary {
	totalComponents := result.Production.NumComponents + result.Optional.NumComponents
	totalLibdays := result.Production.Libdays + result.Optional.Libdays
	totalMissedReleases := result.Production.MissedReleases + result.Optional.MissedReleases

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

// String returns a formatted string representation of the results
func (r *Result) String() string {
	const (
		intFormat   = "%-25s prod: %-10d opt: %-10d direct prod: %-10d direct opt: %d\n"
		floatFormat = "%-25s prod: %-10.2f opt: %-10.2f direct prod: %-10.2f direct opt: %.2f\n"
	)

	return fmt.Sprintf(
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
		"Components", r.Production.NumComponents, r.Optional.NumComponents, r.DirectProduction.NumComponents, r.DirectOptional.NumComponents,
		"Libdays", r.Production.Libdays, r.Optional.Libdays, r.DirectProduction.Libdays, r.DirectOptional.Libdays,
		"Missed releases", r.Production.MissedReleases, r.Optional.MissedReleases, r.DirectProduction.MissedReleases, r.DirectOptional.MissedReleases,
		"Missed major", r.Production.MissedMajor, r.Optional.MissedMajor, r.DirectProduction.MissedMajor, r.DirectOptional.MissedMajor,
		"Missed minor", r.Production.MissedMinor, r.Optional.MissedMinor, r.DirectProduction.MissedMinor, r.DirectOptional.MissedMinor,
		"Missed patch", r.Production.MissedPatch, r.Optional.MissedPatch, r.DirectProduction.MissedPatch, r.DirectOptional.MissedPatch,

		// Summary
		r.Summary.TotalComponents,
		r.Summary.TotalLibdays,
		r.Summary.TotalMissedRelease,
		r.Summary.AvgLibdays,
		r.Summary.AvgMissedReleases,
	)
}
