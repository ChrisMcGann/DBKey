// Package core provides chemistry calculations for peptide mass calculations
package core

import "math"

// Atomic masses (monoisotopic)
const (
	MassH  = 1.0078250321
	MassC  = 12.0000000000
	MassN  = 14.0030740052
	MassO  = 15.9949146221
	MassS  = 31.9720706900
	MassP  = 30.9737615100
	
	// Proton mass for charge calculations
	ProtonMass = 1.00727646688
)

// AminoAcidComposition stores elemental composition
type AminoAcidComposition struct {
	C, H, N, O, S int
}

// AminoAcidMasses maps amino acid one-letter codes to elemental composition
var AminoAcidMasses = map[rune]AminoAcidComposition{
	'A': {C: 3, H: 5, N: 1, O: 1, S: 0},
	'R': {C: 6, H: 12, N: 4, O: 1, S: 0},
	'N': {C: 4, H: 6, N: 2, O: 2, S: 0},
	'D': {C: 4, H: 5, N: 1, O: 3, S: 0},
	'C': {C: 3, H: 5, N: 1, O: 1, S: 1},
	'E': {C: 5, H: 7, N: 1, O: 3, S: 0},
	'Q': {C: 5, H: 8, N: 2, O: 2, S: 0},
	'G': {C: 2, H: 3, N: 1, O: 1, S: 0},
	'H': {C: 6, H: 7, N: 3, O: 1, S: 0},
	'I': {C: 6, H: 11, N: 1, O: 1, S: 0},
	'L': {C: 6, H: 11, N: 1, O: 1, S: 0},
	'K': {C: 6, H: 12, N: 2, O: 1, S: 0},
	'M': {C: 5, H: 9, N: 1, O: 1, S: 1},
	'F': {C: 9, H: 9, N: 1, O: 1, S: 0},
	'P': {C: 5, H: 7, N: 1, O: 1, S: 0},
	'S': {C: 3, H: 5, N: 1, O: 2, S: 0},
	'T': {C: 4, H: 7, N: 1, O: 2, S: 0},
	'W': {C: 11, H: 10, N: 2, O: 1, S: 0},
	'Y': {C: 9, H: 9, N: 1, O: 2, S: 0},
	'V': {C: 5, H: 9, N: 1, O: 1, S: 0},
}

// CalculatePeptideMass computes monoisotopic mass of a peptide sequence
// including modifications, then returns the m/z for a given charge state.
func CalculatePeptideMass(sequence string, charge int, modifications []Modification) float64 {
	comp := AminoAcidComposition{C: 0, H: 2, N: 0, O: 1, S: 0} // Add water
	
	for _, aa := range sequence {
		if aaComp, ok := AminoAcidMasses[aa]; ok {
			comp.C += aaComp.C
			comp.H += aaComp.H
			comp.N += aaComp.N
			comp.O += aaComp.O
			comp.S += aaComp.S
		}
	}
	
	mass := float64(comp.C)*MassC +
		float64(comp.H)*MassH +
		float64(comp.N)*MassN +
		float64(comp.O)*MassO +
		float64(comp.S)*MassS
	
	// Add modification masses
	for _, mod := range modifications {
		mass += mod.Mass
	}
	
	// Calculate m/z: (mass + charge * proton) / charge
	mz := (mass + float64(charge)*ProtonMass) / float64(charge)
	
	return mz
}

// CalculateNeutralMass computes the neutral monoisotopic mass of a peptide
func CalculateNeutralMass(sequence string, modifications []Modification) float64 {
	comp := AminoAcidComposition{C: 0, H: 2, N: 0, O: 1, S: 0} // Add water
	
	for _, aa := range sequence {
		if aaComp, ok := AminoAcidMasses[aa]; ok {
			comp.C += aaComp.C
			comp.H += aaComp.H
			comp.N += aaComp.N
			comp.O += aaComp.O
			comp.S += aaComp.S
		}
	}
	
	mass := float64(comp.C)*MassC +
		float64(comp.H)*MassH +
		float64(comp.N)*MassN +
		float64(comp.O)*MassO +
		float64(comp.S)*MassS
	
	// Add modification masses
	for _, mod := range modifications {
		mass += mod.Mass
	}
	
	return mass
}

// RoundFloat rounds a float to n decimal places
func RoundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
