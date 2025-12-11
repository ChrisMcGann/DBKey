package core

import (
	"math"
	"testing"
)

func TestSpectrumValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    *Spectrum
		wantErr bool
	}{
		{
			name: "valid spectrum",
			spec: &Spectrum{
				Sequence:          "PEPTIDE",
				Charge:            2,
				PrecursorMZ:       400.5,
				FragmentationMode: "HCD",
				MassAnalyzer:      "FT",
				Peaks: []Peak{
					{MZ: 100.0, Intensity: 1000.0},
					{MZ: 200.0, Intensity: 2000.0},
				},
			},
			wantErr: false,
		},
		{
			name: "missing sequence",
			spec: &Spectrum{
				Charge:            2,
				PrecursorMZ:       400.5,
				FragmentationMode: "HCD",
				MassAnalyzer:      "FT",
				Peaks: []Peak{
					{MZ: 100.0, Intensity: 1000.0},
				},
			},
			wantErr: true,
		},
		{
			name: "negative charge",
			spec: &Spectrum{
				Sequence:          "PEPTIDE",
				Charge:            0,
				PrecursorMZ:       400.5,
				FragmentationMode: "HCD",
				MassAnalyzer:      "FT",
				Peaks: []Peak{
					{MZ: 100.0, Intensity: 1000.0},
				},
			},
			wantErr: true,
		},
		{
			name: "no peaks",
			spec: &Spectrum{
				Sequence:          "PEPTIDE",
				Charge:            2,
				PrecursorMZ:       400.5,
				FragmentationMode: "HCD",
				MassAnalyzer:      "FT",
				Peaks:             []Peak{},
			},
			wantErr: true,
		},
		{
			name: "unsorted peaks",
			spec: &Spectrum{
				Sequence:          "PEPTIDE",
				Charge:            2,
				PrecursorMZ:       400.5,
				FragmentationMode: "HCD",
				MassAnalyzer:      "FT",
				Peaks: []Peak{
					{MZ: 200.0, Intensity: 2000.0},
					{MZ: 100.0, Intensity: 1000.0},
				},
			},
			wantErr: true,
		},
		{
			name: "NaN m/z",
			spec: &Spectrum{
				Sequence:          "PEPTIDE",
				Charge:            2,
				PrecursorMZ:       400.5,
				FragmentationMode: "HCD",
				MassAnalyzer:      "FT",
				Peaks: []Peak{
					{MZ: math.NaN(), Intensity: 1000.0},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSortPeaks(t *testing.T) {
	spec := &Spectrum{
		Peaks: []Peak{
			{MZ: 300.0, Intensity: 100.0},
			{MZ: 100.0, Intensity: 200.0},
			{MZ: 200.0, Intensity: 150.0},
		},
	}

	spec.SortPeaks()

	if len(spec.Peaks) != 3 {
		t.Fatalf("Expected 3 peaks, got %d", len(spec.Peaks))
	}

	expected := []float64{100.0, 200.0, 300.0}
	for i, peak := range spec.Peaks {
		if peak.MZ != expected[i] {
			t.Errorf("Peak %d: expected m/z %.1f, got %.1f", i, expected[i], peak.MZ)
		}
	}
}

func TestTotalModMass(t *testing.T) {
	spec := &Spectrum{
		Modifications: []Modification{
			{Mass: 57.021464, Position: 3},
			{Mass: 15.994915, Position: 7},
		},
	}

	total := spec.TotalModMass()
	expected := 57.021464 + 15.994915

	if math.Abs(total-expected) > 0.000001 {
		t.Errorf("Expected total mod mass %.6f, got %.6f", expected, total)
	}
}

func TestModString(t *testing.T) {
	spec := &Spectrum{
		Modifications: []Modification{
			{Mass: 57.021464, Position: 3},
			{Mass: 15.994915, Position: 7},
		},
	}

	modStr := spec.ModString()
	// Should contain both modifications
	if modStr == "" {
		t.Error("Expected non-empty mod string")
	}
}

func TestSpectrumName(t *testing.T) {
	spec := &Spectrum{
		Sequence: "PEPTIDE",
		Charge:   2,
	}

	name := spec.Name()
	expected := "PEPTIDE/2"

	if name != expected {
		t.Errorf("Expected name %s, got %s", expected, name)
	}
}
