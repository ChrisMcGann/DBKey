// Package cmd provides CLI command implementations
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Flags for convert command
	inputFile        string
	inputFormat      string
	outputFile       string
	fragmentation    string
	collisionEnergy  float64
	massAnalyzer     string
	topN             int
	cutoffPercent    float64
	ionTypes         string
	massOffsetCSV    string
	compoundClassCSV string
	oldModMass       float64
	newModMass       float64
	threads          int
	chunkSize        int
)

var rootCmd = &cobra.Command{
	Use:   "dbkey",
	Short: "DBKey - Spectral library conversion tool",
	Long: `DBKey converts spectral libraries (MSP, SPTXT, BLIB) to SQLite databases
compatible with RTLS/mzVault workflows.

Fast, memory-efficient, and cross-platform conversion with support for:
- Peak filtering (top-N, intensity cutoff)
- Ion type filtering
- Fragment mass adjustments
- Mass offset and compound class mapping`,
	Version: "2.0.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(summarizeCmd)

	// Convert command flags
	convertCmd.Flags().StringVarP(&inputFile, "in", "i", "", "Input file path (required)")
	convertCmd.Flags().StringVarP(&inputFormat, "from", "f", "", "Input format: msp, sptxt, blib (auto-detect if not specified)")
	convertCmd.Flags().StringVarP(&outputFile, "out", "o", "", "Output database file (required)")
	convertCmd.Flags().StringVar(&fragmentation, "fragmentation", "HCD", "Fragmentation mode: HCD, CID, or 'read' to read from file")
	convertCmd.Flags().Float64Var(&collisionEnergy, "collision-energy", 0, "Collision energy (0 = read from file)")
	convertCmd.Flags().StringVar(&massAnalyzer, "mass-analyzer", "FT", "Mass analyzer: FT or IT")
	convertCmd.Flags().IntVar(&topN, "top-n", 0, "Keep only top N most intense peaks (0 = no limit)")
	convertCmd.Flags().Float64Var(&cutoffPercent, "cutoff", 0, "Intensity cutoff as % of base peak (0 = no cutoff)")
	convertCmd.Flags().StringVar(&ionTypes, "ion-types", "", "Comma-separated ion types to keep (e.g., 'b,y')")
	convertCmd.Flags().StringVar(&massOffsetCSV, "mass-offset", "", "Path to mass offset CSV file")
	convertCmd.Flags().StringVar(&compoundClassCSV, "compound-class", "", "Path to compound class CSV file")
	convertCmd.Flags().Float64Var(&oldModMass, "adjust-fragments-old", 0, "Old modification mass for fragment adjustment")
	convertCmd.Flags().Float64Var(&newModMass, "adjust-fragments-new", 0, "New modification mass for fragment adjustment")
	convertCmd.Flags().IntVar(&threads, "threads", 1, "Number of worker threads (currently not implemented)")
	convertCmd.Flags().IntVar(&chunkSize, "chunk-size", 10000, "Chunk size for batch processing")

	convertCmd.MarkFlagRequired("in")
	convertCmd.MarkFlagRequired("out")
}

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert spectral library to SQLite database",
	Long: `Convert spectral libraries in MSP, SPTXT, or BLIB format to SQLite databases
compatible with RTLS and mzVault workflows.

Examples:
  # Convert MSP file with default settings
  dbkey convert --in library.msp --out library.db

  # Convert with filtering and mass analyzer specification
  dbkey convert --in library.msp --out library.db --top-n 150 --cutoff 1 --mass-analyzer FT

  # Convert with ion type filtering and fragment adjustment
  dbkey convert --in library.msp --out library.db --ion-types b,y --adjust-fragments-old 229.16 --adjust-fragments-new 304.21`,
	RunE: runConvert,
}

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate input file format and contents",
	Long:  `Validate that an input file is properly formatted and contains valid spectral data.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement validation
		fmt.Fprintf(os.Stderr, "Validation not yet implemented\n")
		return nil
	},
}

var summarizeCmd = &cobra.Command{
	Use:   "summarize [file]",
	Short: "Summarize spectral library contents",
	Long:  `Print summary statistics about a spectral library including spectrum count, m/z ranges, and metadata coverage.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement summarization
		fmt.Fprintf(os.Stderr, "Summarization not yet implemented\n")
		return nil
	},
}

func runConvert(cmd *cobra.Command, args []string) error {
	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Auto-detect format if not specified
	if inputFormat == "" {
		ext := strings.ToLower(filepath.Ext(inputFile))
		switch ext {
		case ".msp":
			inputFormat = "msp"
		case ".sptxt":
			inputFormat = "sptxt"
		case ".blib":
			inputFormat = "blib"
		default:
			return fmt.Errorf("cannot auto-detect format from extension '%s', please specify --from", ext)
		}
	}

	// Validate format
	inputFormat = strings.ToLower(inputFormat)
	if inputFormat != "msp" && inputFormat != "sptxt" && inputFormat != "blib" {
		return fmt.Errorf("invalid input format '%s', must be msp, sptxt, or blib", inputFormat)
	}

	// BLIB is not yet implemented
	if inputFormat == "blib" {
		return fmt.Errorf("format 'blib' is not yet implemented")
	}

	fmt.Printf("Converting %s to %s...\n", inputFile, outputFile)
	fmt.Printf("Format: %s\n", inputFormat)
	fmt.Printf("Fragmentation: %s\n", fragmentation)
	fmt.Printf("Mass Analyzer: %s\n", massAnalyzer)
	
	if topN > 0 {
		fmt.Printf("Top N filter: %d\n", topN)
	}
	if cutoffPercent > 0 {
		fmt.Printf("Intensity cutoff: %.1f%%\n", cutoffPercent)
	}
	if ionTypes != "" {
		fmt.Printf("Ion types: %s\n", ionTypes)
	}

	// Call the appropriate conversion logic
	switch inputFormat {
	case "msp":
		return convertMSP()
	case "sptxt":
		return convertSPTXT()
	default:
		return fmt.Errorf("unsupported format: %s", inputFormat)
	}
}
