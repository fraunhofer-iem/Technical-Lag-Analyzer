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

// parseSemver parses a version string into a semantic version
func parseSemver(raw string) (*version.Version, error) {
	if raw == "" {
		return nil, fmt.Errorf("version string cannot be empty")
	}
	return version.NewVersion(raw)
}

// parseAndFilterVersions converts string versions to semver and filters out invalid/prerelease versions
func parseAndFilterVersions(versions []string) ([]*version.Version, error) {
	semVers := make([]*version.Version, 0, len(versions))

	for _, v := range versions {
		semVer, err := parseSemver(v)
		if err != nil {
			slog.Default().Warn("Skipping unparsable version", "version", v, "error", err)
			continue
		}

		// Skip pre-release versions
		if semVer.Prerelease() != "" {
			continue
		}

		semVers = append(semVers, semVer)
	}

	if len(semVers) == 0 {
		return nil, fmt.Errorf("no valid, non-prerelease versions found")
	}

	// Sort versions in ascending order
	slices.SortFunc(semVers, func(a, b *version.Version) int {
		return a.Compare(b)
	})

	return semVers, nil
}

// findVersionIndex finds the index where usedVersion should be inserted in the sorted slice
func findVersionIndex(sortedVersions []*version.Version, usedVersion *version.Version) int {
	return sort.Search(len(sortedVersions), func(i int) bool {
		return sortedVersions[i].GreaterThanOrEqual(usedVersion)
	})
}

// insertVersionIfMissing inserts the used version into the sorted slice if it's not already present
func insertVersionIfMissing(sortedVersions []*version.Version, usedVersion *version.Version, index int) ([]*version.Version, int) {
	// If index is at the end, the used version is newer than all existing versions
	if index == len(sortedVersions) {
		return append(sortedVersions, usedVersion), index
	}

	// If the version at index is not equal to usedVersion, insert it
	if !sortedVersions[index].Equal(usedVersion) {
		return slices.Insert(sortedVersions, index, usedVersion), index
	}

	// Version already exists at the correct position
	return sortedVersions, index
}

// calculateVersionDistance calculates the distance metrics between versions
func calculateVersionDistance(sortedVersions []*version.Version, usedIndex int, usedVersion *version.Version) *VersionDistance {
	missedReleases := len(sortedVersions) - 1 - usedIndex

	// Count actual missed releases by type
	var missedMajor, missedMinor, missedPatch int64

	usedSegments := usedVersion.Segments64()
	if len(usedSegments) < 3 {
		usedSegments = append(usedSegments, make([]int64, 3-len(usedSegments))...)
	}

	// Count missed releases by examining each version after the used version
	for i := usedIndex + 1; i < len(sortedVersions); i++ {
		currentSegments := sortedVersions[i].Segments64()
		if len(currentSegments) < 3 {
			currentSegments = append(currentSegments, make([]int64, 3-len(currentSegments))...)
		}

		prevSegments := usedSegments
		if i > usedIndex+1 {
			prevSegments = sortedVersions[i-1].Segments64()
			if len(prevSegments) < 3 {
				prevSegments = append(prevSegments, make([]int64, 3-len(prevSegments))...)
			}
		}

		// Determine release type based on which segment changed
		if currentSegments[0] > prevSegments[0] {
			missedMajor++
		} else if currentSegments[1] > prevSegments[1] {
			missedMinor++
		} else if currentSegments[2] > prevSegments[2] {
			missedPatch++
		}
	}

	return &VersionDistance{
		MissedReleases: int64(missedReleases),
		MissedMajor:    missedMajor,
		MissedMinor:    missedMinor,
		MissedPatch:    missedPatch,
	}
}

// GetVersionDistance calculates how far behind a used version is compared to available versions
func GetVersionDistance(usedVersion string, versions []string) (*VersionDistance, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions provided")
	}

	usedSemver, err := parseSemver(usedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse used version %q: %w", usedVersion, err)
	}

	sortedVersions, err := parseAndFilterVersions(versions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse versions: %w", err)
	}

	usedIndex := findVersionIndex(sortedVersions, usedSemver)
	sortedVersions, usedIndex = insertVersionIfMissing(sortedVersions, usedSemver, usedIndex)

	return calculateVersionDistance(sortedVersions, usedIndex, usedSemver), nil
}

// filterValidVersions filters out versions without publication dates or invalid semver
func filterValidVersions(versions []deps.Version) ([]deps.Version, error) {
	validVersions := make([]deps.Version, 0, len(versions))

	for _, v := range versions {
		// Skip versions without a publication date
		if v.PublishedAt == "" {
			slog.Default().Warn("Skipping version with no PublishedAt", "version", v.Version)
			continue
		}

		sv, err := parseSemver(v.Version)
		if err != nil {
			slog.Default().Warn("Skipping unparsable semver", "version", v.Version, "error", err)
			continue
		}

		// Skip pre-release versions
		if sv.Prerelease() != "" {
			continue
		}

		validVersions = append(validVersions, v)
	}

	if len(validVersions) == 0 {
		return nil, fmt.Errorf("no valid, non-prerelease versions found")
	}

	return validVersions, nil
}

// sortVersionsBySemanticVersion sorts versions by their semantic version
func sortVersionsBySemanticVersion(versions []deps.Version) {
	slices.SortFunc(versions, func(a, b deps.Version) int {
		semverA, errA := parseSemver(a.Version)
		semverB, errB := parseSemver(b.Version)

		// This shouldn't happen since we already filtered, but be safe
		if errA != nil || errB != nil {
			return 0
		}

		return semverA.Compare(semverB)
	})
}

// findUsedVersionIndex finds the index of the used version in the sorted slice
func findUsedVersionIndex(sortedVersions []deps.Version, usedSemver *version.Version) (int, error) {
	idx := slices.IndexFunc(sortedVersions, func(v deps.Version) bool {
		sv, err := parseSemver(v.Version)
		if err != nil {
			return false
		}
		return sv.Equal(usedSemver)
	})

	if idx == -1 {
		return -1, fmt.Errorf("used version not found among valid versions")
	}

	return idx, nil
}

// GetLibyear calculates the "libyear" metric - how old the used version is compared to the newest
func GetLibyear(usedVersion string, versions []deps.Version) (*time.Duration, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions provided")
	}

	usedSemver, err := parseSemver(usedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse used version %q: %w", usedVersion, err)
	}

	validVersions, err := filterValidVersions(versions)
	if err != nil {
		return nil, err
	}

	sortVersionsBySemanticVersion(validVersions)

	usedIdx, err := findUsedVersionIndex(validVersions, usedSemver)
	if err != nil {
		return nil, fmt.Errorf("used version %q: %w", usedVersion, err)
	}

	usedTime, err := validVersions[usedIdx].Time()
	if err != nil {
		return nil, fmt.Errorf("failed to parse time for used version %q: %w", validVersions[usedIdx].Version, err)
	}

	// The newest version is the last element of the sorted slice
	newestVersion := validVersions[len(validVersions)-1]
	newestTime, err := newestVersion.Time()
	if err != nil {
		return nil, fmt.Errorf("failed to parse time for newest version %q: %w", newestVersion.Version, err)
	}

	duration := newestTime.Sub(usedTime)
	return &duration, nil
}
