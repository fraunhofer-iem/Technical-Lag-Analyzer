package semver

import (
	"errors"
	"fmt"
	"log/slog"
	"sbom-technical-lag/internal/deps"
	"slices"
	"sort"
	"time"

	"github.com/hashicorp/go-version"
)

var (
	// ErrEmptyVersion is returned when an empty version string is provided
	ErrEmptyVersion = errors.New("version string cannot be empty")
	// ErrNoValidVersions is returned when no valid versions are found
	ErrNoValidVersions = errors.New("no valid, non-prerelease versions found")
	// ErrVersionNotFound is returned when the used version is not found among valid versions
	ErrVersionNotFound = errors.New("used version not found among valid versions")
	// ErrNoVersionsProvided is returned when an empty versions slice is provided
	ErrNoVersionsProvided = errors.New("no versions provided")
)

// VersionDistance represents the distance metrics between versions
type VersionDistance struct {
	MissedReleases int64 `json:"missedReleases"`
	MissedMajor    int64 `json:"missedMajor"`
	MissedMinor    int64 `json:"missedMinor"`
	MissedPatch    int64 `json:"missedPatch"`
}

// parseSemver parses a version string into a semantic version with better error handling
func parseSemver(rawVersion string) (*version.Version, error) {
	if rawVersion == "" {
		return nil, ErrEmptyVersion
	}

	v, err := version.NewVersion(rawVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %q: %w", rawVersion, err)
	}

	return v, nil
}

// parseAndFilterVersions converts string versions to semver and filters out invalid/prerelease versions
func parseAndFilterVersions(versions []string) ([]*version.Version, error) {
	if len(versions) == 0 {
		return nil, ErrNoVersionsProvided
	}

	semVers := make([]*version.Version, 0, len(versions))
	var parseErrors int

	for _, v := range versions {
		semVer, err := parseSemver(v)
		if err != nil {
			slog.Default().Debug("Skipping unparsable version", "version", v, "error", err)
			parseErrors++
			continue
		}

		// Skip pre-release versions as they're not stable releases
		if semVer.Prerelease() != "" {
			slog.Default().Debug("Skipping pre-release version", "version", v, "prerelease", semVer.Prerelease())
			continue
		}

		semVers = append(semVers, semVer)
	}

	if len(semVers) == 0 {
		if parseErrors > 0 {
			return nil, fmt.Errorf("%w (failed to parse %d versions)", ErrNoValidVersions, parseErrors)
		}
		return nil, ErrNoValidVersions
	}

	// Sort versions in ascending order for consistent processing
	slices.SortFunc(semVers, func(a, b *version.Version) int {
		return a.Compare(b)
	})

	slog.Default().Debug("Parsed and filtered versions",
		"total", len(versions),
		"valid", len(semVers),
		"parse_errors", parseErrors)

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

	if missedReleases <= 0 {
		return &VersionDistance{
			MissedReleases: 0,
			MissedMajor:    0,
			MissedMinor:    0,
			MissedPatch:    0,
		}
	}

	// Count missed releases by type
	var missedMajor, missedMinor, missedPatch int64

	usedSegments := normalizeSegments(usedVersion.Segments64())

	// Count missed releases by examining each version after the used version
	for i := usedIndex + 1; i < len(sortedVersions); i++ {
		currentSegments := normalizeSegments(sortedVersions[i].Segments64())

		var prevSegments []int64
		if i == usedIndex+1 {
			prevSegments = usedSegments
		} else {
			prevSegments = normalizeSegments(sortedVersions[i-1].Segments64())
		}

		// Determine release type based on which segment changed
		switch {
		case currentSegments[0] > prevSegments[0]:
			missedMajor++
		case currentSegments[1] > prevSegments[1]:
			missedMinor++
		case currentSegments[2] > prevSegments[2]:
			missedPatch++
		default:
			// This shouldn't happen with properly sorted versions, but handle gracefully
			slog.Default().Debug("Unexpected version ordering",
				"current", sortedVersions[i].String(),
				"previous", sortedVersions[i-1].String())
			missedPatch++ // Default to patch release
		}
	}

	result := &VersionDistance{
		MissedReleases: int64(missedReleases),
		MissedMajor:    missedMajor,
		MissedMinor:    missedMinor,
		MissedPatch:    missedPatch,
	}

	// Verify consistency - the sum should equal total missed releases
	sum := missedMajor + missedMinor + missedPatch
	if sum != int64(missedReleases) {
		slog.Default().Warn("Version distance calculation inconsistency",
			"expected_total", missedReleases,
			"calculated_sum", sum,
			"major", missedMajor,
			"minor", missedMinor,
			"patch", missedPatch,
			"used_version", usedVersion.String())

		// Adjust patch count to maintain consistency
		result.MissedPatch = max(int64(missedReleases)-missedMajor-missedMinor, 0)
	}

	return result
}

// normalizeSegments ensures version segments have at least major.minor.patch
func normalizeSegments(segments []int64) []int64 {
	if len(segments) >= 3 {
		return segments[:3] // Take only first 3 segments
	}

	normalized := make([]int64, 3)
	copy(normalized, segments)
	// Remaining elements are already 0 due to zero value

	return normalized
}

// GetVersionDistance calculates how far behind a used version is compared to available versions
func GetVersionDistance(usedVersion string, versions []string) (*VersionDistance, error) {
	if len(versions) == 0 {
		return nil, ErrNoVersionsProvided
	}

	usedSemver, err := parseSemver(usedVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid used version %q: %w", usedVersion, err)
	}

	sortedVersions, err := parseAndFilterVersions(versions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse available versions: %w", err)
	}

	usedIndex := findVersionIndex(sortedVersions, usedSemver)
	sortedVersions, usedIndex = insertVersionIfMissing(sortedVersions, usedSemver, usedIndex)

	distance := calculateVersionDistance(sortedVersions, usedIndex, usedSemver)

	slog.Default().Debug("Calculated version distance",
		"used_version", usedVersion,
		"total_versions", len(versions),
		"valid_versions", len(sortedVersions),
		"used_index", usedIndex,
		"missed_releases", distance.MissedReleases,
		"missed_major", distance.MissedMajor,
		"missed_minor", distance.MissedMinor,
		"missed_patch", distance.MissedPatch)

	return distance, nil
}

// filterValidVersions filters out versions without publication dates or invalid semver
func filterValidVersions(versions []deps.Version) ([]deps.Version, error) {
	if len(versions) == 0 {
		return nil, ErrNoVersionsProvided
	}

	validVersions := make([]deps.Version, 0, len(versions))
	var invalidCount int

	for _, v := range versions {
		// Skip versions without a publication date
		if v.PublishedAt == "" {
			slog.Default().Debug("Skipping version with no publication date", "version", v.Version)
			invalidCount++
			continue
		}

		// Validate that we can parse the publication date
		if _, err := v.Time(); err != nil {
			slog.Default().Debug("Skipping version with invalid publication date",
				"version", v.Version,
				"published_at", v.PublishedAt,
				"error", err)
			invalidCount++
			continue
		}

		sv, err := parseSemver(v.Version)
		if err != nil {
			slog.Default().Debug("Skipping version with invalid semver", "version", v.Version, "error", err)
			invalidCount++
			continue
		}

		// Skip pre-release versions
		if sv.Prerelease() != "" {
			slog.Default().Debug("Skipping pre-release version", "version", v.Version, "prerelease", sv.Prerelease())
			continue
		}

		validVersions = append(validVersions, v)
	}

	if len(validVersions) == 0 {
		if invalidCount > 0 {
			return nil, fmt.Errorf("%w (%d versions had issues)", ErrNoValidVersions, invalidCount)
		}
		return nil, ErrNoValidVersions
	}

	slog.Default().Debug("Filtered versions for libyear calculation",
		"total", len(versions),
		"valid", len(validVersions),
		"invalid", invalidCount)

	return validVersions, nil
}

// sortVersionsBySemanticVersion sorts versions by their semantic version
func sortVersionsBySemanticVersion(versions []deps.Version) error {
	slices.SortFunc(versions, func(a, b deps.Version) int {
		semverA, errA := parseSemver(a.Version)
		semverB, errB := parseSemver(b.Version)

		// This shouldn't happen since we already filtered, but be defensive
		if errA != nil && errB != nil {
			return 0 // Consider them equal if both are invalid
		}
		if errA != nil {
			return 1 // Put invalid versions at the end
		}
		if errB != nil {
			return -1 // Put invalid versions at the end
		}

		return semverA.Compare(semverB)
	})

	return nil
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
		return -1, ErrVersionNotFound
	}

	return idx, nil
}

// GetLibyear calculates the "libyear" metric - time difference between used version and newest version
func GetLibyear(usedVersion string, versions []deps.Version) (*time.Duration, error) {
	if len(versions) == 0 {
		return nil, ErrNoVersionsProvided
	}

	usedSemver, err := parseSemver(usedVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid used version %q: %w", usedVersion, err)
	}

	validVersions, err := filterValidVersions(versions)
	if err != nil {
		return nil, fmt.Errorf("failed to filter versions: %w", err)
	}

	if err := sortVersionsBySemanticVersion(validVersions); err != nil {
		return nil, fmt.Errorf("failed to sort versions: %w", err)
	}

	usedIdx, err := findUsedVersionIndex(validVersions, usedSemver)
	if err != nil {
		return nil, fmt.Errorf("used version %q not found: %w", usedVersion, err)
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

	// Ensure duration is not negative (shouldn't happen with proper sorting)
	if duration < 0 {
		slog.Default().Warn("Negative libyear duration detected",
			"used_version", usedVersion,
			"used_time", usedTime,
			"newest_version", newestVersion.Version,
			"newest_time", newestTime,
			"duration", duration)
		duration = 0
	}

	slog.Default().Debug("Calculated libyear",
		"used_version", usedVersion,
		"newest_version", newestVersion.Version,
		"duration", duration,
		"days", duration.Hours()/24)

	return &duration, nil
}
