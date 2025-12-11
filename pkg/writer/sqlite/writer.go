// Package sqlite provides SQLite database writing for spectral libraries
package sqlite

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/ChrisMcGann/DBKey/pkg/core"
	_ "github.com/mattn/go-sqlite3"
)

const (
	// Date format for HeaderTable (ISO 8601)
	headerDateFormat = "2006-01-02"
	// Date format for MaintenanceTable (space-separated, matches R implementation)
	maintenanceDateFormat = "2006 01 02"
)

// Writer handles writing spectra to SQLite database files
type Writer struct {
	db           *sql.DB
	outputPath   string
	compoundStmt *sql.Stmt
	spectrumStmt *sql.Stmt
	compoundID   int
}

// NewWriter creates a new SQLite writer
func NewWriter(outputPath string) (*Writer, error) {
	db, err := sql.Open("sqlite3", outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	w := &Writer{
		db:         db,
		outputPath: outputPath,
		compoundID: 1,
	}

	if err := w.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	if err := w.prepareStatements(); err != nil {
		db.Close()
		return nil, err
	}

	return w, nil
}

// createTables creates the required database schema
func (w *Writer) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS CompoundTable (
		CompoundId INTEGER PRIMARY KEY,
		Formula TEXT,
		Name TEXT,
		Synonyms BLOB_TEXT,
		Tag TEXT,
		Sequence TEXT,
		CASId TEXT,
		ChemSpiderId TEXT,
		HMDBId TEXT,
		KEGGId TEXT,
		PubChemId TEXT,
		Structure BLOB_TEXT,
		mzCloudId INTEGER,
		CompoundClass TEXT,
		SmilesDescription TEXT,
		InChiKey TEXT
	);

	CREATE TABLE IF NOT EXISTS SpectrumTable (
		SpectrumId INTEGER PRIMARY KEY,
		CompoundId INTEGER REFERENCES CompoundTable(CompoundId),
		mzCloudURL TEXT,
		ScanFilter TEXT,
		RetentionTime DOUBLE,
		ScanNumber INTEGER,
		PrecursorMass DOUBLE,
		NeutralMass DOUBLE,
		CollisionEnergy DOUBLE,
		Polarity TEXT,
		FragmentationMode TEXT,
		IonizationMode TEXT,
		MassAnalyzer TEXT,
		InstrumentName TEXT,
		InstrumentOperator TEXT,
		RawFileURL TEXT,
		blobMass BLOB,
		blobIntensity BLOB,
		blobAccuracy BLOB,
		blobResolution BLOB,
		blobNoises BLOB,
		blobFlags BLOB,
		blobTopPeaks BLOB,
		Version INTEGER,
		CreationDate TEXT,
		Curator TEXT,
		CurationType TEXT,
		PrecursorIonType TEXT,
		Accession TEXT
	);

	CREATE TABLE IF NOT EXISTS HeaderTable (
		version INTEGER NOT NULL DEFAULT 0,
		CreationDate TEXT,
		LastModifiedDate TEXT,
		Description TEXT,
		Company TEXT,
		ReadOnly BOOL,
		UserAccess TEXT,
		PartialEdits BOOL
	);

	CREATE TABLE IF NOT EXISTS MaintenanceTable (
		CreationDate TEXT,
		NoofCompoundsModified INTEGER,
		Description TEXT
	);
	`

	_, err := w.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// prepareStatements prepares SQL statements for batch insertion
func (w *Writer) prepareStatements() error {
	var err error

	w.compoundStmt, err = w.db.Prepare(`
		INSERT INTO CompoundTable (
			CompoundId, Formula, Name, Synonyms, Tag, Sequence,
			CASId, ChemSpiderId, HMDBId, KEGGId, PubChemId,
			Structure, mzCloudId, CompoundClass, SmilesDescription, InChiKey
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare compound statement: %w", err)
	}

	w.spectrumStmt, err = w.db.Prepare(`
		INSERT INTO SpectrumTable (
			SpectrumId, CompoundId, mzCloudURL, ScanFilter, RetentionTime,
			ScanNumber, PrecursorMass, NeutralMass, CollisionEnergy, Polarity,
			FragmentationMode, IonizationMode, MassAnalyzer, InstrumentName,
			InstrumentOperator, RawFileURL, blobMass, blobIntensity,
			blobAccuracy, blobResolution, blobNoises, blobFlags,
			blobTopPeaks, Version, CreationDate, Curator, CurationType,
			PrecursorIonType, Accession
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare spectrum statement: %w", err)
	}

	return nil
}

// WriteSpectrum writes a single spectrum to the database
func (w *Writer) WriteSpectrum(spec *core.Spectrum) error {
	// Ensure peaks are sorted
	if !spec.ArePeaksSorted() {
		spec.SortPeaks()
	}

	// Build tag with modifications and mass offset
	tag := fmt.Sprintf("mods:%s", spec.ModString())
	if spec.MassOffset != 0 {
		tag = fmt.Sprintf("%s massOffset:%.6f", tag, spec.MassOffset)
	}

	// Insert into CompoundTable
	_, err := w.compoundStmt.Exec(
		w.compoundID,       // CompoundId
		spec.ModString(),   // Formula (reuse for mods)
		spec.Name(),        // Name
		"",                 // Synonyms
		tag,                // Tag
		spec.Sequence,      // Sequence
		"",                 // CASId
		"",                 // ChemSpiderId
		"",                 // HMDBId
		"",                 // KEGGId
		"",                 // PubChemId
		"",                 // Structure
		nil,                // mzCloudId
		spec.CompoundClass, // CompoundClass
		"",                 // SmilesDescription
		"",                 // InChiKey
	)
	if err != nil {
		return fmt.Errorf("failed to insert compound: %w", err)
	}

	// Encode peaks as binary blobs (little-endian float64)
	mzBlob := encodePeaksFloat64(spec.Peaks, true)   // m/z values
	intBlob := encodePeaksFloat64(spec.Peaks, false) // intensity values

	// Calculate neutral mass
	neutralMass := core.CalculateNeutralMass(spec.Sequence, spec.Modifications)

	// Handle optional retention time
	var rt interface{} = nil
	if spec.RetentionTime != nil {
		rt = *spec.RetentionTime
	}

	// Handle optional collision energy
	var ce interface{} = nil
	if spec.CollisionEnergy != nil {
		ce = *spec.CollisionEnergy
	}

	// Insert into SpectrumTable
	_, err = w.spectrumStmt.Exec(
		w.compoundID,           // SpectrumId (same as CompoundId for 1:1 mapping)
		w.compoundID,           // CompoundId
		"",                     // mzCloudURL
		"",                     // ScanFilter
		rt,                     // RetentionTime
		0,                      // ScanNumber
		spec.PrecursorMZ,       // PrecursorMass
		neutralMass,            // NeutralMass
		ce,                     // CollisionEnergy
		"+",                    // Polarity
		spec.FragmentationMode, // FragmentationMode
		"ESI",                  // IonizationMode
		spec.MassAnalyzer,      // MassAnalyzer
		spec.Instrument,        // InstrumentName
		"",                     // InstrumentOperator
		"",                     // RawFileURL
		mzBlob,                 // blobMass
		intBlob,                // blobIntensity
		nil,                    // blobAccuracy
		nil,                    // blobResolution
		nil,                    // blobNoises
		nil,                    // blobFlags
		nil,                    // blobTopPeaks
		nil,                    // Version
		nil,                    // CreationDate
		"",                     // Curator
		"",                     // CurationType
		"",                     // PrecursorIonType
		"",                     // Accession
	)
	if err != nil {
		return fmt.Errorf("failed to insert spectrum: %w", err)
	}

	w.compoundID++
	return nil
}

// encodePeaksFloat64 encodes peak data as little-endian float64 blob
func encodePeaksFloat64(peaks []core.Peak, useMZ bool) []byte {
	buf := make([]byte, len(peaks)*8)
	for i, peak := range peaks {
		var value float64
		if useMZ {
			value = peak.MZ
		} else {
			value = peak.Intensity
		}
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(value))
	}
	return buf
}

// Finalize writes the header and maintenance tables and closes the database
func (w *Writer) Finalize() error {
	// Write HeaderTable
	_, err := w.db.Exec(`
		INSERT INTO HeaderTable (version, CreationDate, LastModifiedDate, Description, Company, ReadOnly, UserAccess, PartialEdits)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, 5, time.Now().Format(headerDateFormat), time.Now().Format(headerDateFormat), "", "", false, "", false)
	if err != nil {
		return fmt.Errorf("failed to insert header: %w", err)
	}

	// Write MaintenanceTable
	_, err = w.db.Exec(`
		INSERT INTO MaintenanceTable (CreationDate, NoofCompoundsModified, Description)
		VALUES (?, ?, ?)
	`, time.Now().Format(maintenanceDateFormat), nil, "")
	if err != nil {
		return fmt.Errorf("failed to insert maintenance: %w", err)
	}

	// Close prepared statements
	if w.compoundStmt != nil {
		w.compoundStmt.Close()
	}
	if w.spectrumStmt != nil {
		w.spectrumStmt.Close()
	}

	// Close database
	if err := w.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}

// Close closes the database connection (alias for Finalize)
func (w *Writer) Close() error {
	return w.Finalize()
}
