## ADDED Requirements

### Requirement: Export collection as JSON
The system SHALL allow authenticated users to export their entire collection as a JSON file via `GET /export/json`.

#### Scenario: Successful JSON export
- **WHEN** an authenticated user requests `GET /export/json`
- **THEN** the system SHALL return a downloadable JSON file containing all records with all fields, with `Content-Disposition: attachment; filename="vynilino-export-<date>.json"`

#### Scenario: Empty collection export
- **WHEN** the user's collection is empty
- **THEN** the system SHALL return a valid JSON file with an empty records array

### Requirement: Export collection as CSV
The system SHALL allow authenticated users to export their collection as a CSV file via `GET /export/csv`.

#### Scenario: Successful CSV export
- **WHEN** an authenticated user requests `GET /export/csv`
- **THEN** the system SHALL return a UTF-8 CSV file with headers matching record fields and one row per record

#### Scenario: CSV handles special characters
- **WHEN** record fields contain commas, quotes, or newlines
- **THEN** the system SHALL produce valid RFC 4180 CSV with proper quoting

### Requirement: Import from CSV
The system SHALL allow authenticated users to import records from a CSV file via `POST /import/csv` with `multipart/form-data`.

#### Scenario: Successful import
- **WHEN** an authenticated user uploads a valid CSV file with required headers (title, artist)
- **THEN** the system SHALL create records for each valid row and return a summary (imported count, skipped count, errors)

#### Scenario: Partial import on row errors
- **WHEN** a CSV file contains some invalid rows (missing required fields)
- **THEN** the system SHALL import valid rows and report skipped rows with reasons, without failing the entire import

#### Scenario: Invalid file format rejected
- **WHEN** a non-CSV file is uploaded to the import endpoint
- **THEN** the system SHALL return `415 Unsupported Media Type`

#### Scenario: Import file size limit
- **WHEN** an uploaded CSV exceeds 10MB
- **THEN** the system SHALL return `413 Request Entity Too Large`

### Requirement: Import from Discogs CSV export
The system SHALL support importing from the standard Discogs collection export CSV format (column mapping applied automatically).

#### Scenario: Discogs CSV detected and mapped
- **WHEN** an uploaded CSV contains Discogs-standard headers (e.g., "Catalog#", "Artist", "Title", "Label", "Format", "Rating", "Released", "release_id")
- **THEN** the system SHALL map Discogs columns to vynilino fields automatically and import records

#### Scenario: Unknown columns ignored
- **WHEN** a Discogs CSV contains columns with no vynilino mapping
- **THEN** the system SHALL silently ignore unmapped columns and proceed with known fields
