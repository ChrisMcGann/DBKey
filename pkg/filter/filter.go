// Package filter provides peak filtering and transformation functions
package filter

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ChrisMcGann/DBKey/pkg/core"
)

// Config holds filtering configuration
type Config struct {
	TopN            int      // Keep only top N most intense peaks (0 = no limit)
	IntensityCutoff float64  // Keep only peaks above this % of base peak (0 = no cutoff)
	IonTypes        []string // Keep only specified ion types (nil = all)
	DeltaFragment   float64  // Mass adjustment for fragments
	OldModMass      float64  // Old modification mass to adjust
	NewModMass      float64  // New modification mass
}

// Apply applies all configured filters to a spectrum
func (c *Config) Apply(spec *core.Spectrum) error {
	// Filter by ion type first
	if len(c.IonTypes) > 0 {
		if err := c.filterByIonType(spec); err != nil {
			return err
		}
	}

	// Apply intensity filters
	if c.IntensityCutoff > 0 {
		c.filterByIntensity(spec)
	}

	// Apply top-N filter
	if c.TopN > 0 {
		c.filterTopN(spec)
	}

	// Apply fragment mass adjustment if configured
	if c.OldModMass != 0 && c.NewModMass != 0 {
		if err := c.adjustFragmentMasses(spec); err != nil {
			return err
		}
	}

	// Ensure peaks are sorted after all filtering
	spec.SortPeaks()

	return nil
}

// filterByIonType keeps only peaks matching specified ion types
func (c *Config) filterByIonType(spec *core.Spectrum) error {
	if len(c.IonTypes) == 0 {
		return nil
	}

	var filtered []core.Peak
	for _, peak := range spec.Peaks {
		if matchesIonType(peak.Annotation, c.IonTypes) {
			filtered = append(filtered, peak)
		}
	}

	spec.Peaks = filtered
	return nil
}

// matchesIonType checks if an annotation matches any of the allowed ion types
func matchesIonType(annotation string, ionTypes []string) bool {
	if annotation == "" {
		return false
	}

	for _, ionType := range ionTypes {
		// Match ion type at start of annotation (e.g., "y3", "b2^2")
		if strings.HasPrefix(annotation, ionType) {
			return true
		}
	}
	return false
}

// filterByIntensity removes peaks below the intensity cutoff percentage
func (c *Config) filterByIntensity(spec *core.Spectrum) {
	if len(spec.Peaks) == 0 {
		return
	}

	// Find maximum intensity
	maxIntensity := 0.0
	for _, peak := range spec.Peaks {
		if peak.Intensity > maxIntensity {
			maxIntensity = peak.Intensity
		}
	}

	// Calculate threshold
	threshold := (c.IntensityCutoff / 100.0) * maxIntensity

	// Filter peaks
	var filtered []core.Peak
	for _, peak := range spec.Peaks {
		if peak.Intensity >= threshold {
			filtered = append(filtered, peak)
		}
	}

	spec.Peaks = filtered
}

// filterTopN keeps only the N most intense peaks
func (c *Config) filterTopN(spec *core.Spectrum) {
	if len(spec.Peaks) <= c.TopN {
		return
	}

	// Create a copy and sort by intensity descending
	peaks := make([]core.Peak, len(spec.Peaks))
	copy(peaks, spec.Peaks)

	sort.Slice(peaks, func(i, j int) bool {
		return peaks[i].Intensity > peaks[j].Intensity
	})

	// Keep only top N
	spec.Peaks = peaks[:c.TopN]
}

// adjustFragmentMasses adjusts fragment m/z values based on modification position changes
func (c *Config) adjustFragmentMasses(spec *core.Spectrum) error {
	if len(spec.Modifications) == 0 {
		return nil
	}

	deltaMass := c.NewModMass - c.OldModMass

	for i := range spec.Peaks {
		peak := &spec.Peaks[i]

		if peak.Annotation == "" {
			continue
		}

		// Parse annotation to get ion type, position, and charge
		ionInfo, err := parseIonAnnotation(peak.Annotation)
		if err != nil {
			// Skip peaks with unparseable annotations
			continue
		}

		// Check each modification
		for _, mod := range spec.Modifications {
			// Skip if this modification doesn't match the old mass
			if fmt.Sprintf("%.6f", mod.Mass) != fmt.Sprintf("%.6f", c.OldModMass) {
				continue
			}

			// Adjust based on ion type and position
			// b ions: add mass if modification is at or before the fragment position
			// y ions: add mass if modification is after the fragment position (from C-term)
			shouldAdjust := false

			if ionInfo.ionType == "b" && ionInfo.position >= mod.Position {
				shouldAdjust = true
			} else if ionInfo.ionType == "y" {
				// Y ions count from C-terminus
				seqLen := len(spec.Sequence)
				modPosFromCTerm := seqLen - mod.Position - 1
				if ionInfo.position >= modPosFromCTerm {
					shouldAdjust = true
				}
			}

			if shouldAdjust {
				// Apply mass shift divided by fragment charge
				peak.MZ += deltaMass / float64(ionInfo.charge)
			}
		}
	}

	return nil
}

// ionAnnotationInfo stores parsed ion annotation
type ionAnnotationInfo struct {
	ionType  string
	position int
	charge   int
}

// parseIonAnnotation parses annotations like "y3", "b2^2", "y10^3"
func parseIonAnnotation(annotation string) (*ionAnnotationInfo, error) {
	// Pattern: (ion type)(number)[^(charge)]
	re := regexp.MustCompile(`^([a-z])(\d+)(?:\^(\d+))?`)
	matches := re.FindStringSubmatch(annotation)

	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid ion annotation format: %s", annotation)
	}

	info := &ionAnnotationInfo{
		ionType: matches[1],
		charge:  1, // default charge
	}

	// Parse position
	_, err := fmt.Sscanf(matches[2], "%d", &info.position)
	if err != nil {
		return nil, fmt.Errorf("invalid position in annotation %s: %w", annotation, err)
	}

	// Parse charge if present
	if len(matches) > 3 && matches[3] != "" {
		_, err := fmt.Sscanf(matches[3], "%d", &info.charge)
		if err != nil {
			return nil, fmt.Errorf("invalid charge in annotation %s: %w", annotation, err)
		}
	}

	return info, nil
}

// RemoveZeroIntensityPeaks removes peaks with zero or negative intensity
func RemoveZeroIntensityPeaks(spec *core.Spectrum) {
	var filtered []core.Peak
	for _, peak := range spec.Peaks {
		if peak.Intensity > 0 {
			filtered = append(filtered, peak)
		}
	}
	spec.Peaks = filtered
}
