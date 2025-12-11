// Package core provides modification parsing and management
package core

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ModDatabase stores modification definitions
type ModDatabase struct {
	mods map[string]float64 // name -> mass shift
}

// NewModDatabase creates an empty modification database
func NewModDatabase() *ModDatabase {
	return &ModDatabase{
		mods: make(map[string]float64),
	}
}

// LoadFromCSV loads modifications from a CSV file (format: mod,massshift,aa)
func (db *ModDatabase) LoadFromCSV(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	
	// Skip header line
	if scanner.Scan() {
		// header line
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
			return fmt.Errorf("line %d: invalid format, expected at least 2 comma-separated fields", lineNum)
		}
		
		modName := strings.TrimSpace(parts[0])
		massStr := strings.TrimSpace(parts[1])
		
		mass, err := strconv.ParseFloat(massStr, 64)
		if err != nil {
			return fmt.Errorf("line %d: invalid mass value '%s': %w", lineNum, massStr, err)
		}
		
		db.mods[modName] = mass
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading CSV: %w", err)
	}
	
	return nil
}

// GetMass returns the mass shift for a modification name
func (db *ModDatabase) GetMass(name string) (float64, bool) {
	mass, ok := db.mods[name]
	return mass, ok
}

// Add adds or updates a modification
func (db *ModDatabase) Add(name string, mass float64) {
	db.mods[name] = mass
}

// ParseModString parses a modification string like "57.021464@2;15.994915@8" or "Carbamidomethyl@C2;Oxidation@M8"
// Returns a list of modifications
func (db *ModDatabase) ParseModString(modStr string, sequence string) ([]Modification, error) {
	if modStr == "" {
		return nil, nil
	}
	
	var mods []Modification
	parts := strings.Split(modStr, ";")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Split by @
		atParts := strings.Split(part, "@")
		if len(atParts) != 2 {
			return nil, fmt.Errorf("invalid modification format '%s', expected 'name@position' or 'mass@position'", part)
		}
		
		nameOrMass := strings.TrimSpace(atParts[0])
		posStr := strings.TrimSpace(atParts[1])
		
		var mass float64
		var err error
		
		// Try to parse as a number first (direct mass)
		mass, err = strconv.ParseFloat(nameOrMass, 64)
		if err != nil {
			// Not a number, try to look up as a name
			var ok bool
			mass, ok = db.GetMass(nameOrMass)
			if !ok {
				return nil, fmt.Errorf("unknown modification '%s'", nameOrMass)
			}
		}
		
		// Parse position - may include amino acid letter
		position, err := parsePosition(posStr, sequence)
		if err != nil {
			return nil, fmt.Errorf("invalid position '%s': %w", posStr, err)
		}
		
		mods = append(mods, Modification{
			Mass:     mass,
			Position: position,
			Name:     nameOrMass,
		})
	}
	
	return mods, nil
}

// parsePosition parses a position string that may be just a number or include an amino acid
// Examples: "2", "C2", "R-1" (C-terminal), "A0" (N-terminal)
func parsePosition(posStr string, sequence string) (int, error) {
	posStr = strings.TrimSpace(posStr)
	
	// Handle special cases
	if posStr == "-1" || strings.HasSuffix(posStr, "-1") {
		return -1, nil // N-terminal
	}
	
	// Remove leading amino acid letter if present
	posStr = strings.TrimLeft(posStr, "ACDEFGHIKLMNPQRSTVWY")
	
	pos, err := strconv.Atoi(posStr)
	if err != nil {
		return 0, fmt.Errorf("invalid position number: %w", err)
	}
	
	// Convert to 0-based indexing if it's 1-based
	if pos > 0 {
		pos = pos - 1
	}
	
	return pos, nil
}

// DefaultModDatabase returns a ModDatabase pre-loaded with common modifications
func DefaultModDatabase() *ModDatabase {
	db := NewModDatabase()
	
	// Common modifications from unimod
	db.Add("Acetyl", 42.010565)
	db.Add("Amidated", -0.984016)
	db.Add("Biotin", 226.077598)
	db.Add("Carbamidomethyl", 57.021464)
	db.Add("Carbamyl", 43.005814)
	db.Add("Carboxymethyl", 58.005479)
	db.Add("Deamidated", 0.984016)
	db.Add("Met->Hse", -29.992806)
	db.Add("Met->Hsl", -48.003371)
	db.Add("NIPCAM", 99.068414)
	db.Add("Phospho", 79.966331)
	db.Add("Dehydrated", -18.010565)
	db.Add("Propionamide", 71.037114)
	db.Add("Pyro-carbamidomethyl", 39.994915)
	db.Add("Glu->pyro-Glu", -18.010565)
	db.Add("Gln->pyro-Glu", -17.026549)
	db.Add("Cation:Na", 21.981943)
	db.Add("Methyl", 14.01565)
	db.Add("Oxidation", 15.994915)
	db.Add("Dimethyl", 28.0313)
	db.Add("Trimethyl", 42.04695)
	db.Add("Methylthio", 45.987721)
	db.Add("Sulfo", 79.956815)
	db.Add("Hex", 162.052824)
	db.Add("Lipoyl", 188.032956)
	db.Add("HexNAc", 203.079373)
	db.Add("Farnesyl", 204.187801)
	db.Add("Myristoyl", 210.198366)
	db.Add("PyridoxalPhosphate", 229.014009)
	db.Add("Palmitoyl", 238.229666)
	db.Add("GeranylGeranyl", 272.250401)
	db.Add("Phosphopantetheine", 340.085794)
	db.Add("FAD", 783.141486)
	db.Add("Guanidinyl", 42.021798)
	db.Add("HNE", 156.11503)
	db.Add("Glucuronyl", 176.032088)
	db.Add("Glutathione", 305.068156)
	db.Add("Propionyl", 56.026215)
	db.Add("TMT", 229.162932)
	db.Add("TMTPro", 304.207146)
	db.Add("TMT6plex", 229.162932)
	db.Add("TMT10plex", 229.162932)
	db.Add("TMT11plex", 229.162932)
	db.Add("TMT16plex", 304.207146)
	db.Add("iTRAQ4plex", 144.102063)
	db.Add("iTRAQ8plex", 304.205360)
	
	return db
}
