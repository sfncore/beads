package convex

// Schema defines the SQLite schema for Convex-style document storage.
// This uses 3 tables instead of beads' traditional 11+ table schema.
const Schema = `
-- Documents table: stores all documents as JSON with temporal versioning
-- Every write creates a new row; previous versions are linked via prev_ts
CREATE TABLE IF NOT EXISTS documents (
    -- Internal document ID (blob for efficiency, but we use TEXT for simplicity)
    id TEXT NOT NULL,
    
    -- Timestamp in nanoseconds (provides total ordering)
    ts INTEGER NOT NULL,
    
    -- Which logical table this document belongs to
    -- Examples: "issues", "comments", "dependencies", "labels", "events"
    table_id TEXT NOT NULL,
    
    -- The document content as JSON (NULL for tombstones)
    json_value TEXT,
    
    -- 1 if this is a deletion tombstone, 0 otherwise
    deleted INTEGER NOT NULL DEFAULT 0,
    
    -- Timestamp of the previous version (NULL for first version)
    prev_ts INTEGER,
    
    -- Primary key ensures uniqueness per (ts, table_id, id)
    -- This allows multiple versions of the same document
    PRIMARY KEY (ts, table_id, id)
);

-- Index for efficient latest-version queries per document
CREATE INDEX IF NOT EXISTS idx_documents_by_id ON documents(table_id, id, ts DESC);

-- Index for time-range queries (e.g., change feeds)
CREATE INDEX IF NOT EXISTS idx_documents_by_ts ON documents(ts);

-- Index for table scans
CREATE INDEX IF NOT EXISTS idx_documents_by_table ON documents(table_id, ts DESC);

-- Indexes table: stores secondary indexes for efficient queries
-- Managed automatically by the persistence layer
CREATE TABLE IF NOT EXISTS indexes (
    -- Which index this entry belongs to
    -- Examples: "issues_by_status", "issues_by_priority"
    index_id TEXT NOT NULL,
    
    -- Timestamp when this index entry was created
    ts INTEGER NOT NULL,
    
    -- Serialized index key (format depends on index type)
    key BLOB NOT NULL,
    
    -- 1 if this entry has been deleted, 0 otherwise
    deleted INTEGER NOT NULL DEFAULT 0,
    
    -- Table ID of the referenced document (for validation)
    table_id TEXT,
    
    -- Document ID being indexed
    document_id TEXT,
    
    -- Primary key ensures uniqueness
    PRIMARY KEY (index_id, key, ts)
);

-- Index for efficient key lookups
CREATE INDEX IF NOT EXISTS idx_indexes_by_key ON indexes(index_id, key, ts DESC);

-- Index for document reference lookups (e.g., when updating indexes)
CREATE INDEX IF NOT EXISTS idx_indexes_by_doc ON indexes(table_id, document_id, ts DESC);

-- Persistence globals: key-value store for metadata
CREATE TABLE IF NOT EXISTS persistence_globals (
    key TEXT PRIMARY KEY,
    json_value TEXT NOT NULL
);
`

// SchemaVersion is the current schema version.
// Increment this when making schema changes.
const SchemaVersion = 1

// LatestDocumentQuery returns the SQL to get the latest non-deleted version of a document.
const LatestDocumentQuery = `
SELECT id, ts, table_id, json_value, deleted, prev_ts
FROM documents
WHERE table_id = ? AND id = ? AND deleted = 0
ORDER BY ts DESC
LIMIT 1
`

// LatestDocumentAtTSQuery returns the SQL to get the latest version at or before a timestamp.
const LatestDocumentAtTSQuery = `
SELECT id, ts, table_id, json_value, deleted, prev_ts
FROM documents
WHERE table_id = ? AND id = ? AND ts <= ?
ORDER BY ts DESC
LIMIT 1
`

// DocumentsByTableQuery returns the SQL to get all documents in a table within a timestamp range.
const DocumentsByTableQuery = `
SELECT id, ts, table_id, json_value, deleted, prev_ts
FROM documents
WHERE table_id = ? AND ts >= ? AND ts <= ?
ORDER BY ts %s
`

// InsertDocumentQuery is the SQL to insert a new document version.
const InsertDocumentQuery = `
INSERT INTO documents (id, ts, table_id, json_value, deleted, prev_ts)
VALUES (?, ?, ?, ?, ?, ?)
`

// InsertIndexQuery is the SQL to insert a new index entry.
const InsertIndexQuery = `
INSERT INTO indexes (index_id, ts, key, deleted, table_id, document_id)
VALUES (?, ?, ?, ?, ?, ?)
`

// GetGlobalQuery is the SQL to get a global value.
const GetGlobalQuery = `
SELECT json_value FROM persistence_globals WHERE key = ?
`

// SetGlobalQuery is the SQL to set a global value.
const SetGlobalQuery = `
INSERT OR REPLACE INTO persistence_globals (key, json_value) VALUES (?, ?)
`

// MaxTimestampQuery returns the maximum timestamp in the documents table.
const MaxTimestampQuery = `
SELECT COALESCE(MAX(ts), 0) FROM documents
`

// DocumentCountQuery returns the approximate count of non-deleted documents in a table.
const DocumentCountQuery = `
SELECT COUNT(DISTINCT id) FROM documents WHERE table_id = ? AND deleted = 0
`

// IndexScanQuery returns documents by index key range.
// Note: This is a template - the ORDER BY direction is substituted at runtime.
const IndexScanQuery = `
WITH latest_index AS (
    SELECT index_id, key, ts, deleted, table_id, document_id,
           ROW_NUMBER() OVER (PARTITION BY index_id, key ORDER BY ts DESC) as rn
    FROM indexes
    WHERE index_id = ? AND key >= ? AND (? IS NULL OR key < ?) AND ts <= ?
)
SELECT d.id, d.ts, d.table_id, d.json_value, d.deleted, d.prev_ts
FROM latest_index i
JOIN documents d ON d.table_id = i.table_id AND d.id = i.document_id
WHERE i.rn = 1 AND i.deleted = 0 AND d.deleted = 0
  AND d.ts = (
    SELECT MAX(ts) FROM documents 
    WHERE table_id = i.table_id AND id = i.document_id AND ts <= ? AND deleted = 0
  )
ORDER BY i.key %s
LIMIT ?
`

// IndexGetQuery returns a single document by exact index key.
const IndexGetQuery = `
WITH latest_index AS (
    SELECT index_id, key, ts, deleted, table_id, document_id
    FROM indexes
    WHERE index_id = ? AND key = ? AND ts <= ?
    ORDER BY ts DESC
    LIMIT 1
)
SELECT d.id, d.ts, d.table_id, d.json_value, d.deleted, d.prev_ts
FROM latest_index i
JOIN documents d ON d.table_id = i.table_id AND d.id = i.document_id
WHERE i.deleted = 0 AND d.deleted = 0
  AND d.ts = (
    SELECT MAX(ts) FROM documents 
    WHERE table_id = i.table_id AND id = i.document_id AND ts <= ? AND deleted = 0
  )
LIMIT 1
`
