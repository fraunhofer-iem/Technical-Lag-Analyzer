package technicalLag

import (
	"fmt"
	"log/slog"
	"sbom-technical-lag/internal/deps"
	"sbom-technical-lag/internal/sbom"
	"sbom-technical-lag/internal/semver"
	"time"

	cdx "github.com/CycloneDX/cyclonedx-go"
)

type TechnicalLag struct {
	Libdays         float64
	VersionDistance semver.VersionDistance
}

func Calculate(bom *cdx.BOM) (map[cdx.Component]TechnicalLag, error) {

	componentToVersions := make(map[cdx.Component]TechnicalLag)
	errC := 0

	for _, c := range *bom.Components {

		depsRes, err := deps.GetVersions(c.PackageURL)
		if err != nil {
			slog.Default().Warn("Deps.dev api query failed", "purl", c.PackageURL, "err", err)
			errC++
			continue
		}

		versions := make([]deps.Version, 0, len(depsRes.Versions))
		rawVersions := make([]string, 0, len(depsRes.Versions))
		for _, v := range depsRes.Versions {
			if v.Version.PublishedAt == "" && v.PublishedAt != "" {
				v.Version.PublishedAt = v.PublishedAt
			}
			versions = append(versions, v.Version)
			rawVersions = append(rawVersions, v.Version.Version)
		}

		t, err := semver.GetLibyear(c.Version, versions)
		if err != nil {
			slog.Default().Warn("Failed to calculate libyear", "err", err)
			errC++
			continue
		}

		days := t.Hours() / 24

		versionDistance, err := semver.GetVersionDistance(c.Version, rawVersions)
		if err != nil {
			slog.Default().Warn("Failed to calculate version distance", "err", err)
			errC++
			continue
		}

		componentToVersions[c] = TechnicalLag{Libdays: days, VersionDistance: *versionDistance}
	}

	if errC > 0 {
		slog.Default().Warn("Requests failed", "counter", errC)
	}

	return componentToVersions, nil
}

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

type ComponentLag struct {
	Component      cdx.Component `json:"component"`
	Libdays        float64       `json:"libdays"`
	MissedReleases int64         `json:"missedReleases"`
	MissedMajor    int64         `json:"missedMajor"`
	MissedMinor    int64         `json:"missedMinor"`
	MissedPatch    int64         `json:"missedPatch"`
}

type Result struct {
	Opt        TechLagStats `json:"optional"`
	Prod       TechLagStats `json:"production"`
	DirectOpt  TechLagStats `json:"directOptional"`
	DirectProd TechLagStats `json:"directProduction"`
	Timestamp  int64        `json:"timestamp"`
}

func CreateResult(bom *cdx.BOM, cm map[cdx.Component]TechnicalLag) (Result, error) {
	result := Result{
		Opt:        TechLagStats{Components: make([]ComponentLag, 0)},
		Prod:       TechLagStats{Components: make([]ComponentLag, 0)},
		DirectOpt:  TechLagStats{Components: make([]ComponentLag, 0)},
		DirectProd: TechLagStats{Components: make([]ComponentLag, 0)},
	}

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
			updateTechLagStats(&result.Prod, v, k, componentLag)
		} else {
			updateTechLagStats(&result.Opt, v, k, componentLag)
		}
	}

	directDeps, err := sbom.GetDirectDeps(bom)
	if err != nil {
		slog.Default().Warn("Failed to get direct dependencies", "err", err)
	}

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
			updateTechLagStats(&result.DirectProd, tl, dep, componentLag)
		} else {
			updateTechLagStats(&result.DirectOpt, tl, dep, componentLag)
		}
	}

	result.Timestamp = time.Now().Unix()

	return result, nil
}

func (r *Result) String() string {
	// Format specifier for integer types (int, int64)
	intFormat := "%-25s prod: %-10d opt: %d\n"
	// Format specifier for float types, rounding to 2 decimal places
	floatFormat := "%-25s prod: %-10.2f opt: %.2f\n"

	return fmt.Sprintf(
		"--- Overall ---\n"+
			intFormat+ // For NumComponents
			floatFormat+ // For Libdays
			intFormat+ // For MissedReleases
			intFormat+ // For MissedMajor
			intFormat+ // For MissedMinor
			intFormat+ // For MissedPatch
			"\n--- Direct ---\n"+
			intFormat+ // For NumComponents
			floatFormat+ // For Libdays
			intFormat+ // For MissedReleases
			intFormat+ // For MissedMajor
			intFormat+ // For MissedMinor
			intFormat, // For MissedPatch

		// Arguments for the "Overall" section
		"Number components", r.Prod.NumComponents, r.Opt.NumComponents,
		"Libdays", r.Prod.Libdays, r.Opt.Libdays,
		"Missed releases", r.Prod.MissedReleases, r.Opt.MissedReleases,
		"Missed major", r.Prod.MissedMajor, r.Opt.MissedMajor,
		"Missed minor", r.Prod.MissedMinor, r.Opt.MissedMinor,
		"Missed patch", r.Prod.MissedPatch, r.Opt.MissedPatch,

		// Arguments for the "Direct" section
		"Number components", r.DirectProd.NumComponents, r.DirectOpt.NumComponents,
		"Libdays direct", r.DirectProd.Libdays, r.DirectOpt.Libdays,
		"Missed releases direct", r.DirectProd.MissedReleases, r.DirectOpt.MissedReleases,
		"Missed major direct", r.DirectProd.MissedMajor, r.DirectOpt.MissedMajor,
		"Missed minor direct", r.DirectProd.MissedMinor, r.DirectOpt.MissedMinor,
		"Missed patch direct", r.DirectProd.MissedPatch, r.DirectOpt.MissedPatch,
	)
}

// updateTechLagStats updates the TechLagStats fields with the given technical lag information
func updateTechLagStats(stats *TechLagStats, tl TechnicalLag, c cdx.Component, componentLag ComponentLag) {
	stats.Libdays += tl.Libdays
	stats.MissedReleases += tl.VersionDistance.MissedReleases
	stats.MissedMajor += tl.VersionDistance.MissedMajor
	stats.MissedMinor += tl.VersionDistance.MissedMinor
	stats.MissedPatch += tl.VersionDistance.MissedPatch
	stats.NumComponents++
	stats.Components = append(stats.Components, componentLag)

	if tl.VersionDistance.MissedReleases > stats.HighestMissedReleases {
		stats.HighestMissedReleases = tl.VersionDistance.MissedReleases
		stats.ComponentHighestMissedReleases = c
	}
	if tl.Libdays > stats.HighestLibdays {
		stats.HighestLibdays = tl.Libdays
		stats.ComponentHighestLibdays = c
	}
}
