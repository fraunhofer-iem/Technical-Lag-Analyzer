package semver

import (
	"sbom-technical-lag/internal/deps"
	"testing"
	"time"
)

func TestVersionDistanceValidUsedVersionNotContained(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []string{"1.0.1", "0.1.0", "0.2.0", "2.0.3", "1.2.0", "1.2.3"}

	d, err := GetVersionDistance(usedVersion, versions)
	if err != nil {
		t.Fatalf("no error expected")
	}

	if d.MissedReleases != 4 {
		t.Fatalf("unexpected number of missed releases. Expected 4, got %d", d.MissedReleases)
	}

	if d.MissedMajor != 1 {
		t.Fatalf("unexpected number of missed major releases. Expected 1, got %d", d.MissedMajor)
	}
	if d.MissedMinor != 1 {
		t.Fatalf("unexpected number of missed minor releases. Expected 1, got %d", d.MissedMinor)
	}

	if d.MissedPatch != 2 {
		t.Fatalf("unexpected number of missed patch releases. Expected 2, got %d", d.MissedPatch)
	}

	// Verify that the sum equals total missed releases
	totalIndividual := d.MissedMajor + d.MissedMinor + d.MissedPatch
	if totalIndividual != d.MissedReleases {
		t.Fatalf("sum of individual missed releases (%d) does not equal total missed releases (%d)", totalIndividual, d.MissedReleases)
	}
}

func TestVersionDistanceValidUsedVersion(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []string{"0.1.0", "1.0.0", "0.2.0", "1.0.1", "1.2.0", "1.2.3", "2.0.3"}

	d, err := GetVersionDistance(usedVersion, versions)
	if err != nil {
		t.Fatalf("no error expected")
	}

	if d.MissedReleases != 4 {
		t.Fatalf("unexpected number of missed releases. Expected 4, got %d", d.MissedReleases)
	}

	if d.MissedMajor != 1 {
		t.Fatalf("unexpected number of missed major releases. Expected 1, got %d", d.MissedMajor)
	}
	if d.MissedMinor != 1 {
		t.Fatalf("unexpected number of missed minor releases. Expected 1, got %d", d.MissedMinor)
	}

	if d.MissedPatch != 2 {
		t.Fatalf("unexpected number of missed patch releases. Expected 2, got %d", d.MissedPatch)
	}

	// Verify that the sum equals total missed releases
	totalIndividual := d.MissedMajor + d.MissedMinor + d.MissedPatch
	if totalIndividual != d.MissedReleases {
		t.Fatalf("sum of individual missed releases (%d) does not equal total missed releases (%d)", totalIndividual, d.MissedReleases)
	}
}

func TestGetLibyearSuccess(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []deps.Version{
		{Version: "0.9.0", PublishedAt: "2020-06-15T10:30:00Z"},
		{Version: "1.0.0", PublishedAt: "2021-01-20T14:45:30Z"},
		{Version: "1.1.0", PublishedAt: "2021-08-05T09:15:45Z"},
		{Version: "2.0.0", PublishedAt: "2022-03-18T16:20:10Z"},
	}

	libyear, err := GetLibyear(usedVersion, versions)
	if err != nil {
		t.Fatalf("no error expected, got: %v", err)
	}

	if libyear == nil {
		t.Fatalf("expected non-nil libyear")
	}

	// Calculate expected duration between used version (1.0.0) and the newest version (2.0.0)
	usedTime, _ := time.Parse(time.RFC3339, "2021-01-20T14:45:30Z")
	newestTime, _ := time.Parse(time.RFC3339, "2022-03-18T16:20:10Z")
	expectedDuration := newestTime.Sub(usedTime)

	if *libyear != expectedDuration {
		t.Fatalf("unexpected libyear duration. Expected %v, got %v", expectedDuration, *libyear)
	}
}

func TestGetLibyearVersionNotFound(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []deps.Version{
		{Version: "0.9.0", PublishedAt: "2020-05-12T08:45:20Z"},
		{Version: "1.1.0", PublishedAt: "2021-11-03T12:30:15Z"},
		{Version: "2.0.0", PublishedAt: "2022-07-22T17:10:05Z"},
	}

	_, err := GetLibyear(usedVersion, versions)
	if err == nil {
		t.Fatalf("expected error for version not found")
	}
}

func TestGetLibyearEmptyVersions(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []deps.Version{}

	_, err := GetLibyear(usedVersion, versions)
	if err == nil {
		t.Fatalf("expected error for empty versions array")
	}
}

func TestGetLibyearInvalidUsedVersion(t *testing.T) {
	usedVersion := "invalid.version"
	versions := []deps.Version{
		{Version: "0.9.0", PublishedAt: "2020-03-25T11:20:30Z"},
		{Version: "1.0.0", PublishedAt: "2021-09-14T15:40:55Z"},
	}

	_, err := GetLibyear(usedVersion, versions)
	if err == nil {
		t.Fatalf("expected error for invalid used version")
	}
}

func TestVersionDistanceEdgeCases(t *testing.T) {
	// Test case with only patch releases
	t.Run("OnlyPatchReleases", func(t *testing.T) {
		usedVersion := "1.0.0"
		versions := []string{"1.0.0", "1.0.1", "1.0.2", "1.0.3"}

		d, err := GetVersionDistance(usedVersion, versions)
		if err != nil {
			t.Fatalf("no error expected")
		}

		if d.MissedReleases != 3 {
			t.Errorf("Expected 3 missed releases, got %d", d.MissedReleases)
		}
		if d.MissedMajor != 0 {
			t.Errorf("Expected 0 missed major, got %d", d.MissedMajor)
		}
		if d.MissedMinor != 0 {
			t.Errorf("Expected 0 missed minor, got %d", d.MissedMinor)
		}
		if d.MissedPatch != 3 {
			t.Errorf("Expected 3 missed patch, got %d", d.MissedPatch)
		}

		// Verify sum equals total
		total := d.MissedMajor + d.MissedMinor + d.MissedPatch
		if total != d.MissedReleases {
			t.Errorf("Sum (%d) != total (%d)", total, d.MissedReleases)
		}
	})

	// Test case with only minor releases
	t.Run("OnlyMinorReleases", func(t *testing.T) {
		usedVersion := "1.0.0"
		versions := []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0"}

		d, err := GetVersionDistance(usedVersion, versions)
		if err != nil {
			t.Fatalf("no error expected")
		}

		if d.MissedReleases != 3 {
			t.Errorf("Expected 3 missed releases, got %d", d.MissedReleases)
		}
		if d.MissedMajor != 0 {
			t.Errorf("Expected 0 missed major, got %d", d.MissedMajor)
		}
		if d.MissedMinor != 3 {
			t.Errorf("Expected 3 missed minor, got %d", d.MissedMinor)
		}
		if d.MissedPatch != 0 {
			t.Errorf("Expected 0 missed patch, got %d", d.MissedPatch)
		}

		// Verify sum equals total
		total := d.MissedMajor + d.MissedMinor + d.MissedPatch
		if total != d.MissedReleases {
			t.Errorf("Sum (%d) != total (%d)", total, d.MissedReleases)
		}
	})

	// Test case with only major releases
	t.Run("OnlyMajorReleases", func(t *testing.T) {
		usedVersion := "1.0.0"
		versions := []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"}

		d, err := GetVersionDistance(usedVersion, versions)
		if err != nil {
			t.Fatalf("no error expected")
		}

		if d.MissedReleases != 3 {
			t.Errorf("Expected 3 missed releases, got %d", d.MissedReleases)
		}
		if d.MissedMajor != 3 {
			t.Errorf("Expected 3 missed major, got %d", d.MissedMajor)
		}
		if d.MissedMinor != 0 {
			t.Errorf("Expected 0 missed minor, got %d", d.MissedMinor)
		}
		if d.MissedPatch != 0 {
			t.Errorf("Expected 0 missed patch, got %d", d.MissedPatch)
		}

		// Verify sum equals total
		total := d.MissedMajor + d.MissedMinor + d.MissedPatch
		if total != d.MissedReleases {
			t.Errorf("Sum (%d) != total (%d)", total, d.MissedReleases)
		}
	})

	// Test case with mixed releases
	t.Run("MixedReleases", func(t *testing.T) {
		usedVersion := "1.0.0"
		versions := []string{"1.0.0", "1.0.1", "1.1.0", "2.0.0", "2.0.1", "2.1.0"}

		d, err := GetVersionDistance(usedVersion, versions)
		if err != nil {
			t.Fatalf("no error expected")
		}

		if d.MissedReleases != 5 {
			t.Errorf("Expected 5 missed releases, got %d", d.MissedReleases)
		}
		if d.MissedMajor != 1 {
			t.Errorf("Expected 1 missed major, got %d", d.MissedMajor)
		}
		if d.MissedMinor != 2 {
			t.Errorf("Expected 2 missed minor, got %d", d.MissedMinor)
		}
		if d.MissedPatch != 2 {
			t.Errorf("Expected 2 missed patch, got %d", d.MissedPatch)
		}

		// Verify sum equals total
		total := d.MissedMajor + d.MissedMinor + d.MissedPatch
		if total != d.MissedReleases {
			t.Errorf("Sum (%d) != total (%d)", total, d.MissedReleases)
		}
	})

	// Test case where used version is the latest
	t.Run("LatestVersion", func(t *testing.T) {
		usedVersion := "2.0.0"
		versions := []string{"1.0.0", "1.1.0", "2.0.0"}

		d, err := GetVersionDistance(usedVersion, versions)
		if err != nil {
			t.Fatalf("no error expected")
		}

		if d.MissedReleases != 0 {
			t.Errorf("Expected 0 missed releases, got %d", d.MissedReleases)
		}
		if d.MissedMajor != 0 {
			t.Errorf("Expected 0 missed major, got %d", d.MissedMajor)
		}
		if d.MissedMinor != 0 {
			t.Errorf("Expected 0 missed minor, got %d", d.MissedMinor)
		}
		if d.MissedPatch != 0 {
			t.Errorf("Expected 0 missed patch, got %d", d.MissedPatch)
		}

		// Verify sum equals total
		total := d.MissedMajor + d.MissedMinor + d.MissedPatch
		if total != d.MissedReleases {
			t.Errorf("Sum (%d) != total (%d)", total, d.MissedReleases)
		}
	})
}

func TestGetLibyearWithPrereleaseVersions(t *testing.T) {
	usedVersion := "1.0.0"
	versions := []deps.Version{
		{Version: "0.9.0", PublishedAt: "2020-10-05T13:25:40Z"},
		{Version: "1.0.0", PublishedAt: "2021-04-18T09:50:15Z"},
		{Version: "1.1.0-beta", PublishedAt: "2021-07-30T16:35:20Z"}, // This should be filtered out
		{Version: "2.0.0", PublishedAt: "2022-01-12T11:05:30Z"},
	}

	libyear, err := GetLibyear(usedVersion, versions)
	if err != nil {
		t.Fatalf("no error expected, got: %v", err)
	}

	// Calculate expected duration between used version (1.0.0) and the newest version (2.0.0)
	usedTime, _ := time.Parse(time.RFC3339, "2021-04-18T09:50:15Z")
	newestTime, _ := time.Parse(time.RFC3339, "2022-01-12T11:05:30Z")
	expectedDuration := newestTime.Sub(usedTime)

	if *libyear != expectedDuration {
		t.Fatalf("unexpected libyear duration. Expected %v, got %v", expectedDuration, *libyear)
	}
}

func TestVersionDistanceNoNegativeValues(t *testing.T) {
	// Test to verify that the fix prevents negative values and ensures sum consistency
	testCases := []struct {
		name        string
		usedVersion string
		versions    []string
		expectError bool
	}{
		{
			name:        "Complex version sequence",
			usedVersion: "2.1.5",
			versions:    []string{"1.0.0", "1.5.2", "2.0.0", "2.1.5", "2.2.0", "3.0.1", "3.1.0"},
			expectError: false,
		},
		{
			name:        "Used version in middle",
			usedVersion: "1.5.0",
			versions:    []string{"1.0.0", "1.2.3", "1.5.0", "1.5.1", "1.6.0", "2.0.0"},
			expectError: false,
		},
		{
			name:        "Version with high patch number",
			usedVersion: "1.0.100",
			versions:    []string{"1.0.100", "1.1.0", "2.0.0"},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := GetVersionDistance(tc.usedVersion, tc.versions)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify no negative values
			if d.MissedMajor < 0 {
				t.Errorf("MissedMajor is negative: %d", d.MissedMajor)
			}
			if d.MissedMinor < 0 {
				t.Errorf("MissedMinor is negative: %d", d.MissedMinor)
			}
			if d.MissedPatch < 0 {
				t.Errorf("MissedPatch is negative: %d", d.MissedPatch)
			}
			if d.MissedReleases < 0 {
				t.Errorf("MissedReleases is negative: %d", d.MissedReleases)
			}

			// Verify sum consistency
			sum := d.MissedMajor + d.MissedMinor + d.MissedPatch
			if sum != d.MissedReleases {
				t.Errorf("Sum of individual counters (%d) != MissedReleases (%d). Major: %d, Minor: %d, Patch: %d",
					sum, d.MissedReleases, d.MissedMajor, d.MissedMinor, d.MissedPatch)
			}

			t.Logf("Version %s: Major: %d, Minor: %d, Patch: %d, Total: %d",
				tc.usedVersion, d.MissedMajor, d.MissedMinor, d.MissedPatch, d.MissedReleases)
		})
	}
}
