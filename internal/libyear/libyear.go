package libyear

import (
	cdx "github.com/CycloneDX/cyclonedx-go"
	"log/slog"
	"sbom-technical-lag/internal/deps"
	"sbom-technical-lag/internal/semver"
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
