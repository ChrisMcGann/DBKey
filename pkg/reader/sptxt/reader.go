// Package sptxt provides streaming readers for SPTXT (SpectraST) format spectral libraries
package sptxt

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChrisMcGann/DBKey/pkg/core"
)

// Reader provides streaming access to SPTXT format files
type Reader struct {
	scanner     *bufio.Scanner
	modDB       *core.ModDatabase
	lineNum     int
	currentSpec *core.Spectrum
	err         error
}

// NewReader creates a new SPTXT reader
func NewReader(r io.Reader, modDB *core.ModDatabase) *Reader {
	if modDB == nil {
		modDB = core.DefaultModDatabase()
	}

	return &Reader{
		scanner: bufio.NewScanner(r),
		modDB:   modDB,
	}
}

// Next advances to the next spectrum. Returns false when no more spectra or error.
func (r *Reader) Next() bool {
	r.currentSpec = nil

	spec, err := r.readSpectrum()
	if err != nil {
		if err != io.EOF {
			r.err = err
		}
		return false
	}

	r.currentSpec = spec
	return true
}

// Spectrum returns the current spectrum
func (r *Reader) Spectrum() *core.Spectrum {
	return r.currentSpec
}

// Err returns any error encountered during reading
func (r *Reader) Err() error {
	return r.err
}

// readSpectrum reads a single spectrum entry from the SPTXT file
func (r *Reader) readSpectrum() (*core.Spectrum, error) {
	spec := &core.Spectrum{
		SourceFormat: "sptxt",
		Peaks:        []core.Peak{},
	}

	var numPeaks int
	inPeaks := false
	peaksRead := 0

	for r.scanner.Scan() {
		r.lineNum++
		line := strings.TrimSpace(r.scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "###") {
			continue
		}

		// If we've read all peaks, we're done with this entry
		if inPeaks && peaksRead >= numPeaks {
			return spec, nil
		}

		if !inPeaks {
			// Parse header fields
			if strings.HasPrefix(line, "Name: ") {
				name := strings.TrimPrefix(line, "Name: ")
				if err := r.parseName(spec, name); err != nil {
					return nil, fmt.Errorf("line %d: %w", r.lineNum, err)
				}
			} else if strings.HasPrefix(line, "MW: ") {
				// Skip MW, we'll recalculate
			} else if strings.HasPrefix(line, "PrecursorMZ: ") {
				mzStr := strings.TrimPrefix(line, "PrecursorMZ: ")
				mz, err := strconv.ParseFloat(mzStr, 64)
				if err == nil {
					spec.PrecursorMZ = mz
				}
			} else if strings.HasPrefix(line, "Comment: ") {
				comment := strings.TrimPrefix(line, "Comment: ")
				if err := r.parseComment(spec, comment); err != nil {
					return nil, fmt.Errorf("line %d: %w", r.lineNum, err)
				}
			} else if strings.HasPrefix(line, "NumPeaks: ") {
				numPeaksStr := strings.TrimPrefix(line, "NumPeaks: ")
				n, err := strconv.Atoi(numPeaksStr)
				if err != nil {
					return nil, fmt.Errorf("line %d: invalid num peaks: %w", r.lineNum, err)
				}
				numPeaks = n
				inPeaks = true
			}
		} else {
			// Parse peak line
			peak, err := r.parsePeak(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", r.lineNum, err)
			}
			spec.Peaks = append(spec.Peaks, peak)
			peaksRead++

			// Check if we've read all peaks
			if peaksRead >= numPeaks {
				return spec, nil
			}
		}
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	// If we have a partially read spectrum, return it
	if spec.Sequence != "" {
		return spec, nil
	}

	return nil, io.EOF
}

// parseName extracts sequence, charge, and modifications from Name field
// Format: "n[305]AAAAQDEITGDGTTTVVC[160]LVGELLR/3"
func (r *Reader) parseName(spec *core.Spectrum, name string) error {
	// Split by /
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid name format '%s', expected 'SEQUENCE/CHARGE'", name)
	}

	// Parse charge
	charge, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid charge in name '%s': %w", name, err)
	}
	spec.Charge = charge

	// Parse sequence with inline modifications
	// Format: n[305]AAAAQDEITGDGTTTVVC[160]LVGELLR
	rawSeq := parts[0]
	sequence, mods, err := r.parseInlineModifications(rawSeq)
	if err != nil {
		return fmt.Errorf("failed to parse modifications from sequence: %w", err)
	}

	spec.Sequence = sequence
	spec.Modifications = mods

	return nil
}

// parseInlineModifications parses sequence with inline modifications like n[305]SEQUENCE[160]
func (r *Reader) parseInlineModifications(rawSeq string) (string, []core.Modification, error) {
	var sequence strings.Builder
	var mods []core.Modification
	position := 0

	// Pattern to match modifications: letter followed by [mass]
	re := regexp.MustCompile(`([a-zA-Z]?)(\[\d+(?:\.\d+)?\])`)

	lastIdx := 0
	for _, match := range re.FindAllStringSubmatchIndex(rawSeq, -1) {
		// Add unmodified sequence before this match
		sequence.WriteString(rawSeq[lastIdx:match[0]])

		// Get the amino acid and modification
		aa := rawSeq[match[2]:match[3]]
		modStr := rawSeq[match[4]:match[5]]

		// Parse mass from [mass]
		modStr = strings.Trim(modStr, "[]")
		mass, err := strconv.ParseFloat(modStr, 64)
		if err != nil {
			return "", nil, fmt.Errorf("invalid modification mass '%s': %w", modStr, err)
		}

		// Handle N-terminal modification (n[mass])
		if aa == "n" || aa == "" {
			mods = append(mods, core.Modification{
				Mass:     mass,
				Position: -1, // N-terminal
				Name:     fmt.Sprintf("%.0f", mass),
			})
		} else {
			// Regular amino acid modification
			sequence.WriteString(aa)
			mods = append(mods, core.Modification{
				Mass:     mass,
				Position: position,
				Name:     fmt.Sprintf("%.0f", mass),
			})
			position++
		}

		lastIdx = match[1]
	}

	// Add remaining sequence
	sequence.WriteString(rawSeq[lastIdx:])

	return sequence.String(), mods, nil
}

// parseComment extracts metadata from Comment field
func (r *Reader) parseComment(spec *core.Spectrum, comment string) error {
	// Split by spaces, but be careful with quoted values
	fields := strings.Fields(comment)

	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "Parent":
			mz, err := strconv.ParseFloat(value, 64)
			if err == nil {
				spec.PrecursorMZ = mz
			}

		case "CollisionEnergy":
			ce, err := strconv.ParseFloat(value, 64)
			if err == nil {
				spec.CollisionEnergy = &ce
			}

		case "RetentionTime":
			// May be comma-separated list, take first value
			rtParts := strings.Split(value, ",")
			if len(rtParts) > 0 {
				rt, err := strconv.ParseFloat(rtParts[0], 64)
				if err == nil {
					spec.RetentionTime = &rt
				}
			}

		case "Mods":
			// Format: count/position,AA,ModName/...
			if err := r.parseMods(spec, value); err != nil {
				// Non-fatal, continue
			}
		}
	}

	return nil
}

// parseMods parses modification information from Mods field
func (r *Reader) parseMods(spec *core.Spectrum, modsStr string) error {
	// Format: "2/-1,A,iTRAQ8plex/17,C,Carbamidomethyl"
	parts := strings.Split(modsStr, "/")

	for i := 0; i < len(parts); i += 3 {
		if i+2 >= len(parts) {
			break
		}

		// Parse position from "count/position"
		posParts := strings.Split(parts[i], ",")
		if len(posParts) == 0 {
			continue
		}
		posSubParts := strings.Split(posParts[0], "/")
		if len(posSubParts) != 2 {
			continue
		}

		pos, err := strconv.Atoi(posSubParts[1])
		if err != nil {
			continue
		}

		// Get amino acid and mod name
		// aa := parts[i+1]
		modName := parts[i+2]

		mass, ok := r.modDB.GetMass(modName)
		if ok {
			// Check if this modification already exists (from inline parsing)
			exists := false
			for j := range spec.Modifications {
				if spec.Modifications[j].Position == pos {
					// Update with proper name
					spec.Modifications[j].Name = modName
					exists = true
					break
				}
			}

			if !exists {
				spec.Modifications = append(spec.Modifications, core.Modification{
					Mass:     mass,
					Position: pos,
					Name:     modName,
				})
			}
		}
	}

	return nil
}

// parsePeak parses a single peak line
// Format: "mz\tintensity\tannotation\t..." or similar
func (r *Reader) parsePeak(line string) (core.Peak, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return core.Peak{}, fmt.Errorf("invalid peak format, expected at least 2 fields")
	}

	mz, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return core.Peak{}, fmt.Errorf("invalid m/z value: %w", err)
	}

	intensity, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return core.Peak{}, fmt.Errorf("invalid intensity value: %w", err)
	}

	peak := core.Peak{
		MZ:        mz,
		Intensity: intensity,
	}

	// Parse annotation if present (third field)
	if len(fields) >= 3 {
		annotation := fields[2]
		// Remove ppm info if present (format: "y3/0.5ppm")
		if idx := strings.Index(annotation, "/"); idx > 0 {
			annotation = annotation[:idx]
		}
		// Remove charge info if in format like "y3-17^2"
		// For now, keep the full annotation
		peak.Annotation = annotation
	}

	return peak, nil
}
