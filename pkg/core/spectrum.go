// Package core provides the intermediate representation (IR) models and validation logic
// for spectral library data used by DBKey.
package core

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Spectrum represents a single mass spectrum with all associated metadata.
type Spectrum struct {
	// Required fields
	Sequence          string  // Peptide sequence
	Charge            int     // Precursor charge state
	PrecursorMZ       float64 // Precursor m/z
	Peaks             []Peak  // Fragment peaks
	FragmentationMode string  // HCD, CID, etc.
	MassAnalyzer      string  // FT, IT, etc.

	// Optional metadata
	RetentionTime   *float64 // RT or iRT
	CollisionEnergy *float64 // Normalized collision energy
	Modifications   []Modification
	Instrument      string
	MassOffset      float64 // For massOffset CSV support
	CompoundClass   string  // For compound class CSV support

	// Internal tracking
	SourceFile   string
	SourceFormat string // msp, sptxt, blib
}

// Peak represents a single m/z, intensity pair with optional metadata.
type Peak struct {
	MZ         float64
	Intensity  float64
	Annotation string // Ion annotation (e.g., "y3", "b2^2")
	Charge     int    // Fragment charge (if available)
}

// Modification represents a peptide modification with position and mass shift.
type Modification struct {
	Mass     float64
	Position int    // 0-based position; -1 for N-term, len(seq) for C-term
	Name     string // Modification name (e.g., "Carbamidomethyl", "Oxidation")
}

// ValidationError represents an error found during spectrum validation.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error in %s: %s", e.Field, e.Message)
}

// Validate checks that a spectrum meets all requirements for processing.
func (s *Spectrum) Validate() error {
	var errs []string

	// Required fields
	if s.Sequence == "" {
		errs = append(errs, "sequence is required")
	}
	if s.Charge <= 0 {
		errs = append(errs, "charge must be positive")
	}
	if s.PrecursorMZ <= 0 {
		errs = append(errs, "precursor m/z must be positive")
	}
	if len(s.Peaks) == 0 {
		errs = append(errs, "at least one peak is required")
	}
	if s.FragmentationMode == "" {
		errs = append(errs, "fragmentation mode is required")
	}
	if s.MassAnalyzer == "" {
		errs = append(errs, "mass analyzer is required")
	}

	// Validate peaks
	for i, peak := range s.Peaks {
		if math.IsNaN(peak.MZ) || math.IsInf(peak.MZ, 0) {
			errs = append(errs, fmt.Sprintf("peak %d has invalid m/z", i))
		}
		if math.IsNaN(peak.Intensity) || math.IsInf(peak.Intensity, 0) {
			errs = append(errs, fmt.Sprintf("peak %d has invalid intensity", i))
		}
		if peak.MZ <= 0 {
			errs = append(errs, fmt.Sprintf("peak %d m/z must be positive", i))
		}
		if peak.Intensity < 0 {
			errs = append(errs, fmt.Sprintf("peak %d intensity must be non-negative", i))
		}
	}

	// Check if peaks are sorted
	if !s.ArePeaksSorted() {
		errs = append(errs, "peaks must be sorted by m/z")
	}

	if len(errs) > 0 {
		return &ValidationError{
			Field:   "Spectrum",
			Message: strings.Join(errs, "; "),
		}
	}

	return nil
}

// ArePeaksSorted checks if peaks are sorted by m/z in ascending order.
func (s *Spectrum) ArePeaksSorted() bool {
	for i := 1; i < len(s.Peaks); i++ {
		if s.Peaks[i].MZ < s.Peaks[i-1].MZ {
			return false
		}
	}
	return true
}

// SortPeaks sorts peaks by m/z in ascending order.
func (s *Spectrum) SortPeaks() {
	sort.Slice(s.Peaks, func(i, j int) bool {
		return s.Peaks[i].MZ < s.Peaks[j].MZ
	})
}

// TotalModMass returns the sum of all modification masses.
func (s *Spectrum) TotalModMass() float64 {
	total := 0.0
	for _, mod := range s.Modifications {
		total += mod.Mass
	}
	return total
}

// ModString returns a string representation of modifications in format "mass@pos;mass@pos;..."
func (s *Spectrum) ModString() string {
	if len(s.Modifications) == 0 {
		return ""
	}

	var parts []string
	for _, mod := range s.Modifications {
		parts = append(parts, fmt.Sprintf("%.6f@%d", mod.Mass, mod.Position))
	}
	return strings.Join(parts, ";")
}

// Name returns the spectrum name in format "Sequence/Charge"
func (s *Spectrum) Name() string {
	return fmt.Sprintf("%s/%d", s.Sequence, s.Charge)
}
