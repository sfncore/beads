// Package convex provides a Convex-style document persistence layer for beads.
package convex

import (
	"context"
	"encoding/json"
)

// Persistence defines the interface for temporal document storage.
// Both SQLite (local) and Convex Cloud backends implement this interface.
//
// The persistence layer stores documents as JSON with full temporal history:
// - Every write creates a new version (no in-place updates)
// - Previous versions are linked via PrevTS for time-travel queries
// - Deletions are recorded as tombstones (Deleted=true)
//
// This design enables:
// - Point-in-time queries (read data as it existed at any timestamp)
// - Audit trails (full history of all changes)
// - Conflict detection (compare timestamps during sync)
// - Efficient change feeds (query by timestamp range)
type Persistence interface {
	// IsFresh returns true if this is a newly created persistence store.
	// Used to detect first-run scenarios for initialization.
	IsFresh() bool

	// Reader returns a PersistenceReader for query operations.
	// The reader may be used concurrently from multiple goroutines.
	Reader() PersistenceReader

	// Write atomically writes documents and index entries.
	// All entries in a single Write call are committed together.
	//
	// For updates, the caller should set PrevTS to link to the previous version.
	// For deletions, set Deleted=true and Value=nil.
	Write(ctx context.Context, documents []DocumentLogEntry, indexes []IndexEntry) error

	// WriteGlobal writes a global key-value pair.
	// Globals are used for persistence-wide metadata (schema version, etc).
	WriteGlobal(ctx context.Context, key GlobalKey, value json.RawMessage) error

	// GetGlobal retrieves a global value by key.
	// Returns nil if the key doesn't exist.
	GetGlobal(ctx context.Context, key GlobalKey) (json.RawMessage, error)

	// Close releases resources held by the persistence store.
	Close() error

	// Path returns the path to the persistence store (for SQLite, the db file).
	Path() string
}

// PersistenceReader provides read operations on the persistence store.
// All read operations are point-in-time consistent within a single call.
type PersistenceReader interface {
	// LoadDocuments returns documents from a table within the timestamp range.
	// Results are ordered by timestamp according to the specified order.
	//
	// The returned documents include the full version history within the range.
	// To get only the latest version, use GetDocument instead.
	LoadDocuments(ctx context.Context, tableID string, tsRange TimestampRange, order Order) ([]DocumentLogEntry, error)

	// GetDocument returns the latest non-deleted version of a document.
	// Returns nil if the document doesn't exist or has been deleted.
	//
	// If atTS is non-nil, returns the latest version at or before that timestamp
	// (time-travel query). If atTS is nil, returns the current latest version.
	GetDocument(ctx context.Context, tableID string, docID string, atTS *Timestamp) (*DocumentLogEntry, error)

	// GetDocuments returns the latest non-deleted version of multiple documents.
	// Missing or deleted documents are not included in the result map.
	GetDocuments(ctx context.Context, tableID string, docIDs []string, atTS *Timestamp) (map[string]*DocumentLogEntry, error)

	// IndexScan scans an index within the given key interval.
	// Results are ordered by index key according to the specified order.
	//
	// The returned documents are the latest non-deleted versions at readTS.
	// If readTS is 0, uses the current timestamp.
	IndexScan(ctx context.Context, indexID string, interval Interval, readTS Timestamp, order Order, limit int) ([]IndexResult, error)

	// IndexGet performs a point lookup on an index.
	// Returns the document if found, nil otherwise.
	IndexGet(ctx context.Context, indexID string, key []byte, readTS Timestamp) (*DocumentLogEntry, error)

	// MaxTimestamp returns the maximum timestamp written to the store.
	// Returns 0 if the store is empty.
	MaxTimestamp(ctx context.Context) (Timestamp, error)

	// DocumentCount returns the count of non-deleted documents in a table.
	// This is an approximate count for efficiency (may include recently deleted).
	DocumentCount(ctx context.Context, tableID string) (int64, error)
}

// ConflictStrategy specifies how to handle write conflicts.
type ConflictStrategy int

const (
	// ConflictError returns an error if a document with the same (ts, table_id, id) exists.
	ConflictError ConflictStrategy = iota

	// ConflictOverwrite overwrites existing documents with the same key.
	ConflictOverwrite
)

// WriteBatch accumulates writes for atomic commit.
type WriteBatch struct {
	Documents []DocumentLogEntry
	Indexes   []IndexEntry
}

// AddDocument adds a document to the batch.
func (b *WriteBatch) AddDocument(doc DocumentLogEntry) {
	b.Documents = append(b.Documents, doc)
}

// AddIndex adds an index entry to the batch.
func (b *WriteBatch) AddIndex(idx IndexEntry) {
	b.Indexes = append(b.Indexes, idx)
}

// Clear resets the batch.
func (b *WriteBatch) Clear() {
	b.Documents = b.Documents[:0]
	b.Indexes = b.Indexes[:0]
}

// Len returns the total number of entries in the batch.
func (b *WriteBatch) Len() int {
	return len(b.Documents) + len(b.Indexes)
}
