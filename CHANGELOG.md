# Changelog

All notable changes to DBKey will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-12-11

### Added - Complete Go Rewrite

This is a complete rewrite of DBKey in Go, replacing the R/Shiny implementation.

#### Core Features
- **CLI-first design** with `convert`, `validate`, and `summarize` commands
- **Streaming I/O** for memory-efficient processing of large libraries
- **Cross-platform binaries** for Linux, macOS, and Windows
- **Intermediate representation (IR)** with validation and error handling
- **SQLite writer** with schema compatibility for RTLS/mzVault workflows

#### Format Support
- **MSP (Prosit)** reader with streaming, modification parsing, and metadata extraction
- **SPTXT (SpectraST)** reader with inline modification support
- **BLIB (Skyline)** reader (planned)

#### Processing Features
- **Peak filtering**: top-N most intense peaks
- **Intensity cutoff**: filter by percentage of base peak
- **Ion type filtering**: keep only specified ion types (b, y, a, c, z)
- **Fragment mass adjustment**: modify fragment masses for TMT/iTRAQ corrections
- **Precursor mass recalculation**: accurate m/z from sequence and modifications

#### Additional Features
- **Modification database**: built-in Unimod support with 40+ common modifications
- **Custom modifications**: support for `unimod_custom.csv`
- **Mass offset CSV**: per-sequence mass adjustments
- **Compound class CSV**: per-sequence classification
- **Chemistry calculations**: monoisotopic mass calculations for peptides

#### Testing & CI/CD
- Unit tests for core functionality
- Integration tests with example files
- GitHub Actions workflows for linting, testing, and building
- Cross-platform CI (Linux, macOS, Windows)
- Automated release builds for all platforms

#### Performance
- **11,883 MSP spectra** converted in ~10 seconds
- **5,655 SPTXT spectra** converted in ~5 seconds
- Constant memory usage regardless of library size

#### Database Schema
Four tables compatible with existing workflows:
- `CompoundTable`: peptide sequences and metadata
- `SpectrumTable`: spectral data with binary-encoded peaks (little-endian float64)
- `HeaderTable`: database metadata
- `MaintenanceTable`: maintenance information

### Changed from v1.x (R/Shiny)
- CLI replaces web UI as primary interface (web UI planned as optional wrapper)
- Streaming processing replaces full-file loading
- Go replaces R for core processing
- Native binaries replace Docker as primary distribution
- 10-100x performance improvement

### Migration Notes
- Output database format is fully compatible with v1.x
- Command-line flags map to previous Shiny UI options
- `unimod_custom.csv` format unchanged
- Mass offset and compound class CSV formats unchanged

## [1.x] - Previous R/Shiny Implementation

Historical R/Shiny implementation available at https://github.com/SchweppeLab/DBKey

### Features (R version)
- Shiny web interface
- Docker-based deployment
- MSP, SPTXT, and BLIB format support
- Peak filtering and modification handling
- SQLite database output
