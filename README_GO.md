# DBKey

DBKey is a fast, memory-efficient, cross-platform tool for converting spectral libraries (MSP, SPTXT, BLIB) to SQLite databases compatible with RTLS/mzVault workflows.

## Features

- **Fast streaming conversion** - Memory-efficient processing of large libraries
- **Multiple format support** - MSP (Prosit), SPTXT (SpectraST), and BLIB (Skyline) formats
- **Flexible filtering** - Top-N peaks, intensity cutoff, ion type filtering
- **Fragment adjustments** - Modify fragment masses for TMT/iTRAQ corrections
- **Cross-platform** - Native binaries for Linux, macOS, and Windows
- **Schema compatible** - Generates SQLite databases compatible with existing RTLS workflows

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases page](https://github.com/ChrisMcGann/DBKey/releases).

### Build from Source

Requirements:
- Go 1.22 or later
- gcc (for SQLite support)

```bash
git clone https://github.com/ChrisMcGann/DBKey.git
cd DBKey
go build -o dbkey ./cmd/dbkey
```

## Quick Start

### Convert MSP file
```bash
dbkey convert --in library.msp --out library.db --mass-analyzer FT --fragmentation HCD
```

### Convert SPTXT file
```bash
dbkey convert --in library.sptxt --out library.db --mass-analyzer IT --fragmentation CID
```

### Convert with filtering
```bash
dbkey convert \
  --in library.msp \
  --out library.db \
  --top-n 150 \
  --cutoff 1.0 \
  --ion-types b,y \
  --mass-analyzer FT \
  --fragmentation HCD
```

### Adjust fragment masses (e.g., TMT to TMTPro)
```bash
dbkey convert \
  --in library.msp \
  --out library.db \
  --adjust-fragments-old 229.162932 \
  --adjust-fragments-new 304.207146
```

## Command Reference

### `dbkey convert`

Convert a spectral library to SQLite database.

**Required Flags:**
- `--in, -i` - Input file path
- `--out, -o` - Output database path

**Optional Flags:**
- `--from, -f` - Input format (msp, sptxt, blib). Auto-detected from file extension if not specified.
- `--fragmentation` - Fragmentation mode: HCD, CID, or 'read' to read from file (default: HCD)
- `--collision-energy` - Collision energy value (0 = read from file, default: 0)
- `--mass-analyzer` - Mass analyzer: FT or IT (default: FT)
- `--top-n` - Keep only top N most intense peaks (0 = no limit, default: 0)
- `--cutoff` - Intensity cutoff as % of base peak (0 = no cutoff, default: 0)
- `--ion-types` - Comma-separated ion types to keep (e.g., 'b,y')
- `--mass-offset` - Path to mass offset CSV file (format: Sequence,massOffset)
- `--compound-class` - Path to compound class CSV file (format: Sequence,CompoundClass)
- `--adjust-fragments-old` - Old modification mass for fragment adjustment
- `--adjust-fragments-new` - New modification mass for fragment adjustment

**Examples:**

Basic conversion:
```bash
dbkey convert --in library.msp --out library.db
```

With peak filtering:
```bash
dbkey convert \
  --in library.msp \
  --out library.db \
  --top-n 150 \
  --cutoff 1.0
```

With ion type filtering:
```bash
dbkey convert \
  --in library.msp \
  --out library.db \
  --ion-types b,y
```

TMT to TMTPro conversion:
```bash
dbkey convert \
  --in tmt_library.msp \
  --out tmtpro_library.db \
  --adjust-fragments-old 229.162932 \
  --adjust-fragments-new 304.207146
```

### `dbkey validate`

Validate input file format and contents (coming soon).

### `dbkey summarize`

Print summary statistics about a spectral library (coming soon).

## Database Schema

DBKey generates SQLite databases with the following tables compatible with RTLS/mzVault:

- **CompoundTable** - Peptide sequences and metadata
- **SpectrumTable** - Spectral data with binary-encoded peak arrays
- **HeaderTable** - Database metadata
- **MaintenanceTable** - Maintenance information

Peak data is stored as little-endian float64 binary blobs for efficient storage and retrieval.

## Supported Formats

### MSP (Prosit)
- Prosit-generated MSP files
- Header fields: Name, MW, Comment, Num peaks
- Inline modification parsing
- iRT and collision energy extraction

### SPTXT (SpectraST)
- SpectraST text format libraries
- Inline modification notation (e.g., `n[305]SEQUENCE[160]`)
- Multiple modification support
- Retention time extraction

### BLIB (Skyline)
Coming soon - SQLite-based Skyline libraries with packed peak arrays.

## Modification Support

DBKey includes built-in support for common modifications from Unimod:
- TMT/TMTPro variants
- iTRAQ
- Carbamidomethyl
- Oxidation
- Phosphorylation
- And many more...

Custom modifications can be defined in `unimod_custom.csv` in the working directory.

## Performance

DBKey processes spectral libraries using streaming I/O for memory efficiency:
- **11,883 MSP spectra** converted in ~10 seconds
- **5,655 SPTXT spectra** converted in ~5 seconds
- Memory usage remains constant regardless of library size

## Development

### Running Tests
```bash
go test ./...
```

### Running Tests with Coverage
```bash
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting
```bash
golangci-lint run
```

## Migration from R/Shiny Version

The Go version provides equivalent functionality to the R/Shiny implementation with several improvements:

1. **Performance** - 10-100x faster processing
2. **Memory efficiency** - Streaming I/O vs. full file loading
3. **Cross-platform** - Native binaries for all platforms
4. **CLI-first** - Scriptable and automatable
5. **Deterministic** - Consistent output ordering

The database schema and output format remain compatible with existing workflows.

## Citation

If you use DBKey in your research, please cite:

Real-time spectral library matching for sample multiplexed quantitative proteomics.
Christopher D McGann, William D Barshop, Jesse Canterbury, Chuwei Lin, Wassim Gabriel, Mathias Wilhelm, Graeme McAlister, Devin K Schweppe
bioRxiv 2023.02.08.527705; doi: https://doi.org/10.1101/2023.02.08.527705

## License

[Add license information]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions:
- GitHub Issues: https://github.com/ChrisMcGann/DBKey/issues
- Original R implementation: https://github.com/SchweppeLab/DBKey
