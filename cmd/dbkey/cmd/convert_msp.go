package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ChrisMcGann/DBKey/pkg/core"
	"github.com/ChrisMcGann/DBKey/pkg/filter"
	"github.com/ChrisMcGann/DBKey/pkg/reader/msp"
	"github.com/ChrisMcGann/DBKey/pkg/writer/sqlite"
)

func convertMSP() error {
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

	// Create MSP reader
	reader := msp.NewReader(inFile, modDB)

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
			// Default to HCD if not specified
			spec.FragmentationMode = "HCD"
		}

		// Set mass analyzer
		if massAnalyzer != "" {
			spec.MassAnalyzer = massAnalyzer
		} else if spec.MassAnalyzer == "" {
			// Default to FT if not specified
			spec.MassAnalyzer = "FT"
		}

		// Set collision energy if specified
		if collisionEnergy > 0 {
			spec.CollisionEnergy = &collisionEnergy
		}

		// Recalculate precursor m/z from sequence and modifications
		if len(spec.Sequence) > 0 && spec.Charge > 0 {
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

func loadMassOffsetCSV(path string) (map[string]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result := make(map[string]float64)
	scanner := bufio.NewScanner(file)

	// Skip header line
	if scanner.Scan() {
		// header
	}

	lineNum := 1
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: expected 2 fields (Sequence,massOffset), got %d", lineNum, len(parts))
		}

		sequence := strings.TrimSpace(parts[0])
		offsetStr := strings.TrimSpace(parts[1])

		offset, err := strconv.ParseFloat(offsetStr, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid mass offset value '%s': %w", lineNum, offsetStr, err)
		}

		result[sequence] = offset
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	return result, nil
}

func loadCompoundClassCSV(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)

	// Skip header line
	if scanner.Scan() {
		// header
	}

	lineNum := 1
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: expected 2 fields (Sequence,CompoundClass), got %d", lineNum, len(parts))
		}

		sequence := strings.TrimSpace(parts[0])
		class := strings.TrimSpace(parts[1])

		result[sequence] = class
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	return result, nil
}
