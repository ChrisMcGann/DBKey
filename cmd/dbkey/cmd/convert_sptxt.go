// Package cmd provides SPTXT conversion implementation
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ChrisMcGann/DBKey/pkg/core"
	"github.com/ChrisMcGann/DBKey/pkg/filter"
	"github.com/ChrisMcGann/DBKey/pkg/reader/sptxt"
	"github.com/ChrisMcGann/DBKey/pkg/writer/sqlite"
)

func convertSPTXT() error {
	// Open input file
	inFile, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inFile.Close()

	// Load modification database
	modDB := core.DefaultModDatabase()
	
	// Load custom modifications from unimod_custom.csv if it exists
	if _, err := os.Stat("unimod_custom.csv"); err == nil {
		f, err := os.Open("unimod_custom.csv")
		if err == nil {
			if err := modDB.LoadFromCSV(f); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to load unimod_custom.csv: %v\n", err)
			}
			f.Close()
		}
	}

	// Create SPTXT reader
	reader := sptxt.NewReader(inFile, modDB)

	// Create SQLite writer
	writer, err := sqlite.NewWriter(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output database: %w", err)
	}
	defer writer.Close()

	// Set up filter config
	filterConfig := &filter.Config{
		TopN:            topN,
		IntensityCutoff: cutoffPercent,
		OldModMass:      oldModMass,
		NewModMass:      newModMass,
	}

	// Parse ion types
	if ionTypes != "" {
		filterConfig.IonTypes = strings.Split(ionTypes, ",")
		for i := range filterConfig.IonTypes {
			filterConfig.IonTypes[i] = strings.TrimSpace(filterConfig.IonTypes[i])
		}
	}

	// Load mass offset mapping if provided
	massOffsetMap := make(map[string]float64)
	if massOffsetCSV != "" {
		var err error
		massOffsetMap, err = loadMassOffsetCSV(massOffsetCSV)
		if err != nil {
			return fmt.Errorf("failed to load mass offset CSV: %w", err)
		}
		fmt.Printf("Loaded %d mass offset mappings\n", len(massOffsetMap))
	}

	// Load compound class mapping if provided
	compoundClassMap := make(map[string]string)
	if compoundClassCSV != "" {
		var err error
		compoundClassMap, err = loadCompoundClassCSV(compoundClassCSV)
		if err != nil {
			return fmt.Errorf("failed to load compound class CSV: %w", err)
		}
		fmt.Printf("Loaded %d compound class mappings\n", len(compoundClassMap))
	}

	// Process spectra
	count := 0
	skipped := 0
	
	for reader.Next() {
		spec := reader.Spectrum()
		
		// Apply mass offset if configured
		if offset, ok := massOffsetMap[spec.Sequence]; ok {
			spec.MassOffset = offset
		}

		// Apply compound class if configured
		if class, ok := compoundClassMap[spec.Sequence]; ok {
			spec.CompoundClass = class
		}

		// Set fragmentation mode if specified
		if fragmentation != "" && fragmentation != "read" {
			spec.FragmentationMode = fragmentation
		} else if spec.FragmentationMode == "" {
			// Default to CID for SpectraST files
			spec.FragmentationMode = "CID"
		}

		// Set mass analyzer
		if massAnalyzer != "" {
			spec.MassAnalyzer = massAnalyzer
		} else if spec.MassAnalyzer == "" {
			// Default to IT for SpectraST files
			spec.MassAnalyzer = "IT"
		}

		// Set collision energy if specified
		if collisionEnergy > 0 {
			spec.CollisionEnergy = &collisionEnergy
		}

		// Recalculate precursor m/z from sequence and modifications if not set
		if spec.PrecursorMZ == 0 && len(spec.Sequence) > 0 && spec.Charge > 0 {
			calculatedMZ := core.CalculatePeptideMass(spec.Sequence, spec.Charge, spec.Modifications)
			// Add mass offset to precursor if configured
			if spec.MassOffset != 0 {
				calculatedMZ += spec.MassOffset / float64(spec.Charge)
			}
			spec.PrecursorMZ = calculatedMZ
		}

		// Remove zero intensity peaks
		filter.RemoveZeroIntensityPeaks(spec)

		// Apply filters
		if err := filterConfig.Apply(spec); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to filter spectrum %s: %v\n", spec.Name(), err)
			skipped++
			continue
		}

		// Validate spectrum
		if err := spec.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid spectrum %s: %v\n", spec.Name(), err)
			skipped++
			continue
		}

		// Write to database
		if err := writer.WriteSpectrum(spec); err != nil {
			return fmt.Errorf("failed to write spectrum %s: %w", spec.Name(), err)
		}

		count++
		if count%1000 == 0 {
			fmt.Printf("Processed %d spectra...\n", count)
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	// Finalize database
	if err := writer.Finalize(); err != nil {
		return fmt.Errorf("failed to finalize database: %w", err)
	}

	fmt.Printf("\nConversion complete!\n")
	fmt.Printf("Processed: %d spectra\n", count)
	if skipped > 0 {
		fmt.Printf("Skipped: %d spectra (validation errors)\n", skipped)
	}
	fmt.Printf("Output: %s\n", outputFile)

	return nil
}
