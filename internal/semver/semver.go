package semver

import (
	"fmt"
	"log/slog"
	"sbom-technical-lag/internal/deps"
	"slices"
	"sort"
	"time"

	"github.com/hashicorp/go-version"
)

type VersionDistance struct {
	MissedReleases int64
	MissedMajor    int64
	MissedMinor    int64
	MissedPatch    int64
}

func parseSemver(raw string) (*version.Version, error) {

	if raw == "" {
		return nil, fmt.Errorf("empty string")
	}

	return version.NewVersion(raw)
}

func GetVersionDistance(usedVersion string, versions []string) (*VersionDistance, error) {

	usedSemver, err := parseSemver(usedVersion)
	if err != nil {
		return nil, err
	}

	var semVers []*version.Version
	for _, v := range versions {
		semVer, err := parseSemver(v)
		if err != nil {
			fmt.Printf("can't parse %s to semver\n", v)
			continue
		}
		// Skip pre-release versions.
		if semVer.Prerelease() != "" {
			continue
		}
		semVers = append(semVers, semVer)
	}

	slices.SortFunc(semVers, func(a *version.Version, b *version.Version) int {
		if a == nil || b == nil {
			return 0
		}
		return a.Compare(b)
	})

	i := sort.Search(len(semVers),
		func(i int) bool { return semVers[i].GreaterThanOrEqual(usedSemver) })

	if i == len(semVers) {
		semVers = append(semVers, usedSemver)
	}

	if i < len(semVers) && !semVers[i].Equal(usedSemver) {
		// x is not present in the data,
		// but i is the index where it would be inserted.
		semVers = slices.Insert(semVers, i, usedSemver)
	}

	largestVersion := semVers[len(semVers)-1]

	// semVers[i] == usedSemver
	missedReleases := (len(semVers) - 1) - i

	return &VersionDistance{
		MissedReleases: int64(missedReleases),
		MissedMajor:    largestVersion.Segments64()[0] - usedSemver.Segments64()[0],
		MissedMinor:    largestVersion.Segments64()[1] - usedSemver.Segments64()[1],
		MissedPatch:    largestVersion.Segments64()[2] - usedSemver.Segments64()[2],
	}, nil
}

func GetLibyear(usedVersion string, versions []deps.Version) (*time.Duration, error) {

	usedSemver, err := parseSemver(usedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse usedVersion %s: %w", usedVersion, err)
	}

	validVersions := make([]deps.Version, 0, len(versions))
	for _, v := range versions {
		// Skip versions without a publication date.
		if v.PublishedAt == "" {
			slog.Default().Warn("Skipping version with no PublishedAt", "version", v.Version)
			continue
		}

		sv, err := parseSemver(v.Version)
		if err != nil {
			slog.Default().Warn("Skipping unparsable semver", "version", v.Version)
			continue
		}

		// Skip pre-release versions.
		if sv.Prerelease() != "" {
			continue
		}

		validVersions = append(validVersions, v)
	}

	if len(validVersions) == 0 {
		return nil, fmt.Errorf("no valid, non-prerelease versions found")
	}

	slices.SortFunc(validVersions, func(a, b deps.Version) int {
		semverA, _ := parseSemver(a.Version)
		semverB, _ := parseSemver(b.Version)
		return semverA.Compare(semverB)
	})

	idx := slices.IndexFunc(validVersions, func(v deps.Version) bool {
		sv, _ := parseSemver(v.Version)
		return sv.Equal(usedSemver)
	})

	if idx == -1 {
		return nil, fmt.Errorf("usedVersion %s not found among valid versions", usedVersion)
	}

	usedTime, err := validVersions[idx].Time()
	if err != nil {
		return nil, fmt.Errorf("could not parse time for used version %s: %w", validVersions[idx].Version, err)
	}

	// The newest version is the last element of the sorted slice.
	newestVersion := validVersions[len(validVersions)-1]
	newestTime, err := newestVersion.Time()
	if err != nil {
		return nil, fmt.Errorf("could not parse time for newest version %s: %w", newestVersion.Version, err)
	}

	duration := newestTime.Sub(usedTime)
	return &duration, nil
}
