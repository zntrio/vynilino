## ADDED Requirements

### Requirement: Cover art upload
The system SHALL allow authenticated users to upload a cover art image for a record via a dedicated HTTP endpoint `POST /media/cover-art` with `multipart/form-data`.

#### Scenario: Successful upload
- **WHEN** an authenticated user submits a valid image file (JPEG, PNG, WebP ≤ 5MB) with a `recordId`
- **THEN** the system SHALL store the file on the configured media path, associate it with the record, and return the cover art URL

#### Scenario: Unsupported file type rejected
- **WHEN** a file with a non-image MIME type or a disallowed extension is uploaded
- **THEN** the system SHALL return `415 Unsupported Media Type` and not store the file

#### Scenario: File too large rejected
- **WHEN** an uploaded file exceeds 5MB
- **THEN** the system SHALL return `413 Request Entity Too Large`

#### Scenario: File type validated by content, not extension
- **WHEN** a file is uploaded with an image extension but non-image content (magic bytes mismatch)
- **THEN** the system SHALL return `415 Unsupported Media Type`

### Requirement: Cover art serving
The system SHALL serve cover art images via an authenticated `GET /media/cover-art/:id` endpoint.

#### Scenario: Serve existing cover art
- **WHEN** an authenticated user requests `GET /media/cover-art/:id` for a record they own
- **THEN** the system SHALL respond with the image file and appropriate `Content-Type` and cache headers (`Cache-Control: private, max-age=86400`)

#### Scenario: Cover art not found
- **WHEN** the requested cover art ID does not exist or belongs to another user
- **THEN** the system SHALL return `404 Not Found`

#### Scenario: Unauthenticated access rejected
- **WHEN** cover art is requested without a valid Bearer token
- **THEN** the system SHALL return `401 Unauthorized`

### Requirement: Cover art deletion on record removal
The system SHALL automatically delete the associated cover art file when a record is deleted.

#### Scenario: Cover art cleaned up
- **WHEN** a record with cover art is deleted via `deleteRecord`
- **THEN** the system SHALL remove the corresponding file from the media directory

### Requirement: Configurable media storage path
The system SHALL read the media storage root from `VYNILINO_MEDIA_DIR` environment variable (default: `./data/media`).

#### Scenario: Custom media path respected
- **WHEN** `VYNILINO_MEDIA_DIR=/mnt/storage/vynilino` is set
- **THEN** the system SHALL store and serve all cover art files under that path
