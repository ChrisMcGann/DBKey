// Package msp provides streaming readers for MSP (Prosit) format spectral libraries
package msp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ChrisMcGann/DBKey/pkg/core"
)

// Reader provides streaming access to MSP format files
type Reader struct {
	scanner     *bufio.Scanner
	modDB       *core.ModDatabase
	lineNum     int
	currentSpec *core.Spectrum
	err         error
}

// NewReader creates a new MSP reader
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

// readSpectrum reads a single spectrum entry from the MSP file
func (r *Reader) readSpectrum() (*core.Spectrum, error) {
	spec := &core.Spectrum{
		SourceFormat: "msp",
		Peaks:        []core.Peak{},
	}

	var numPeaks int
	inPeaks := false
	peaksRead := 0

	for r.scanner.Scan() {
		r.lineNum++
		line := strings.TrimSpace(r.scanner.Text())

		// Skip empty lines between entries
		if line == "" && spec.Sequence == "" {
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
			} else if strings.HasPrefix(line, "Comment: ") {
				comment := strings.TrimPrefix(line, "Comment: ")
				if err := r.parseComment(spec, comment); err != nil {
					return nil, fmt.Errorf("line %d: %w", r.lineNum, err)
				}
			} else if strings.HasPrefix(line, "Num peaks: ") {
				numPeaksStr := strings.TrimPrefix(line, "Num peaks: ")
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

// parseName extracts sequence and charge from Name field (format: "SEQUENCE/CHARGE")
func (r *Reader) parseName(spec *core.Spectrum, name string) error {
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid name format '%s', expected 'SEQUENCE/CHARGE'", name)
	}

	spec.Sequence = parts[0]
	charge, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid charge in name '%s': %w", name, err)
	}
	spec.Charge = charge

	return nil
}

// parseComment extracts metadata from Comment field
func (r *Reader) parseComment(spec *core.Spectrum, comment string) error {
	// Comment format: key=value key=value...
	// Example: Parent=414.71 Collision_energy=35 Mods=1/-1,R,TMT_Pro ModString=SEQUENCE//TMT_Pro@R-1/4 iRT=61.01

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

		case "Collision_energy", "CollisionEnergy":
			ce, err := strconv.ParseFloat(value, 64)
			if err == nil {
				spec.CollisionEnergy = &ce
			}

		case "iRT", "RetentionTime":
			rt, err := strconv.ParseFloat(value, 64)
			if err == nil {
				spec.RetentionTime = &rt
			}

		case "Mods":
			// Mods format can vary; try to parse
			// Example: 1/-1,R,TMT_Pro or just modification info
			if err := r.parseMods(spec, value); err != nil {
				// Non-fatal, continue
			}

		case "ModString":
			// ModString format: SEQUENCE//ModName@Pos/Charge
			// Example: EIESAGDITFNR//TMT_Pro@R-1/4
			if err := r.parseModString(spec, value); err != nil {
				// Non-fatal, continue
			}
		}
	}

	return nil
}

// parseMods parses modification information from Mods field
func (r *Reader) parseMods(spec *core.Spectrum, modsStr string) error {
	// Format can be: "1/-1,R,TMT_Pro" or similar
	// Try to extract modification name and position
	parts := strings.Split(modsStr, ",")
	if len(parts) >= 3 {
		// Format: count/position,AA,ModName
		modName := parts[2]
		posStr := parts[0]

		// Extract position
		posParts := strings.Split(posStr, "/")
		if len(posParts) == 2 {
			pos, err := strconv.Atoi(posParts[1])
			if err == nil {
				mass, ok := r.modDB.GetMass(modName)
				if ok {
					spec.Modifications = append(spec.Modifications, core.Modification{
						Mass:     mass,
						Position: pos,
						Name:     modName,
					})
				}
			}
		}
	}
	return nil
}

// parseModString parses modification information from ModString field
func (r *Reader) parseModString(spec *core.Spectrum, modString string) error {
	// Format: SEQUENCE//Mod@Pos/Charge or SEQUENCE//Mod@Pos
	// Example: EIESAGDITFNR//TMT_Pro@R-1/4

	parts := strings.Split(modString, "//")
	if len(parts) < 2 {
		return nil
	}

	modPart := parts[1]
	// Remove trailing charge info if present
	modPart = strings.Split(modPart, "/")[0]

	// Parse modifications (format: ModName@Pos;ModName@Pos...)
	modSpecs := strings.Split(modPart, ";")
	for _, modSpec := range modSpecs {
		if modSpec == "" {
			continue
		}

		modSpec = strings.TrimSpace(modSpec)
		atParts := strings.Split(modSpec, "@")
		if len(atParts) != 2 {
			continue
		}

		modName := atParts[0]
		posStr := atParts[1]

		// Remove amino acid letter from position if present
		posStr = strings.TrimLeft(posStr, "ACDEFGHIKLMNPQRSTVWY")

		pos, err := strconv.Atoi(posStr)
		if err != nil {
			continue
		}

		mass, ok := r.modDB.GetMass(modName)
		if ok {
			spec.Modifications = append(spec.Modifications, core.Modification{
				Mass:     mass,
				Position: pos,
				Name:     modName,
			})
		}
	}

	return nil
}

// parsePeak parses a single peak line (format: "mz\tintensity\t\"annotation\"")
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

	// Parse annotation if present (third field, may be quoted)
	if len(fields) >= 3 {
		annotation := fields[2]
		// Remove quotes
		annotation = strings.Trim(annotation, "\"")
		// Extract ion type and number (remove ppm error info)
		if idx := strings.Index(annotation, "/"); idx > 0 {
			annotation = annotation[:idx]
		}
		peak.Annotation = annotation
	}

	return peak, nil
}
