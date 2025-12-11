package core

import (
	"math"
	"testing"
)

func TestCalculatePeptideMass(t *testing.T) {
	tests := []struct {
		name          string
		sequence      string
		charge        int
		modifications []Modification
		wantMZ        float64
		tolerance     float64
	}{
		{
			name:          "simple peptide charge 1",
			sequence:      "AAA",
			charge:        1,
			modifications: nil,
			wantMZ:        232.129, // Approximate
			tolerance:     0.1,
		},
		{
			name:          "simple peptide charge 2",
			sequence:      "AAA",
			charge:        2,
			modifications: nil,
			wantMZ:        116.569, // Approximate
			tolerance:     0.1,
		},
		{
			name:     "peptide with modification",
			sequence: "PEPTIDE",
			charge:   2,
			modifications: []Modification{
				{Mass: 57.021464, Position: 0}, // Carbamidomethyl on first residue
			},
			wantMZ:    429.2, // Approximate
			tolerance: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePeptideMass(tt.sequence, tt.charge, tt.modifications)
			if math.Abs(got-tt.wantMZ) > tt.tolerance {
				t.Errorf("CalculatePeptideMass() = %.3f, want %.3f (within %.3f)", got, tt.wantMZ, tt.tolerance)
			}
		})
	}
}

func TestCalculateNeutralMass(t *testing.T) {
	tests := []struct {
		name          string
		sequence      string
		modifications []Modification
		wantMass      float64
		tolerance     float64
	}{
		{
			name:          "simple tripeptide",
			sequence:      "AAA",
			modifications: nil,
			wantMass:      231.121, // Approximate neutral mass
			tolerance:     0.1,
		},
		{
			name:     "with modification",
			sequence: "AAA",
			modifications: []Modification{
				{Mass: 57.021464, Position: 0},
			},
			wantMass:  288.143, // Approximate
			tolerance: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateNeutralMass(tt.sequence, tt.modifications)
			if math.Abs(got-tt.wantMass) > tt.tolerance {
				t.Errorf("CalculateNeutralMass() = %.3f, want %.3f (within %.3f)", got, tt.wantMass, tt.tolerance)
			}
		})
	}
}

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		name      string
		val       float64
		precision int
		want      float64
	}{
		{"round to 2 decimals", 3.14159, 2, 3.14},
		{"round to 4 decimals", 3.14159, 4, 3.1416},
		{"round to 0 decimals", 3.6, 0, 4.0},
		{"round negative", -3.14159, 2, -3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.val, tt.precision)
			if got != tt.want {
				t.Errorf("RoundFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}
