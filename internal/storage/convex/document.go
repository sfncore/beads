// Package convex provides a Convex-style document persistence layer for beads.
//
// This package implements temporal document storage with full history,
// compatible with both local SQLite and Convex Cloud backends.
package convex

import (
	"encoding/json"
	"time"
)

// Timestamp represents a nanosecond-precision timestamp for document versioning.
// Using int64 nanoseconds provides total ordering and high precision.
type Timestamp int64

// Now returns the current timestamp.
func Now() Timestamp {
	return Timestamp(time.Now().UnixNano())
}

// Time converts the timestamp to a time.Time.
func (t Timestamp) Time() time.Time {
	return time.Unix(0, int64(t))
}

// DocumentLogEntry represents a single version of a document in the log.
// Every write creates a new entry rather than updating in place,
// enabling full temporal history and time-travel queries.
type DocumentLogEntry struct {
	// TS is the timestamp when this version was written.
	// Used as the primary ordering key for version history.
	TS Timestamp `json:"ts"`

	// ID is the document's unique identifier within its table.
	// For beads issues, this is the issue ID (e.g., "bd-123").
	ID string `json:"id"`

	// TableID identifies which logical table this document belongs to.
	// Examples: "issues", "comments", "dependencies", "labels"
	TableID string `json:"table_id"`

	// Value is the document content as JSON.
	// Nil value indicates a deletion (tombstone).
	Value json.RawMessage `json:"value,omitempty"`

	// Deleted indicates whether this entry is a tombstone.
	Deleted bool `json:"deleted"`

	// PrevTS points to the previous version's timestamp, if any.
	// Nil for the first version of a document.
	PrevTS *Timestamp `json:"prev_ts,omitempty"`
}

// IsDeleted returns true if this entry represents a deletion.
func (d *DocumentLogEntry) IsDeleted() bool {
	return d.Deleted || d.Value == nil
}

// IndexEntry represents an entry in a secondary index.
// Indexes are stored separately from documents and maintained automatically.
type IndexEntry struct {
	// IndexID identifies which index this entry belongs to.
	// Examples: "issues_by_status", "issues_by_priority"
	IndexID string `json:"index_id"`

	// TS is the timestamp when this index entry was created.
	TS Timestamp `json:"ts"`

	// Key is the serialized index key bytes.
	// The format depends on the index type.
	Key []byte `json:"key"`

	// Deleted indicates whether this entry has been removed.
	Deleted bool `json:"deleted"`

	// TableID of the referenced document (optional, for validation).
	TableID string `json:"table_id,omitempty"`

	// DocumentID of the referenced document.
	DocumentID string `json:"document_id,omitempty"`
}

// TimestampRange represents a range of timestamps for queries.
type TimestampRange struct {
	// Start is the inclusive lower bound (0 for unbounded).
	Start Timestamp

	// End is the inclusive upper bound (MaxTimestamp for unbounded).
	End Timestamp
}

// MaxTimestamp is the maximum possible timestamp value.
const MaxTimestamp = Timestamp(1<<63 - 1)

// AllTime returns a range covering all timestamps.
func AllTime() TimestampRange {
	return TimestampRange{Start: 0, End: MaxTimestamp}
}

// AtOrBefore returns a range from the beginning up to and including ts.
func AtOrBefore(ts Timestamp) TimestampRange {
	return TimestampRange{Start: 0, End: ts}
}

// After returns a range starting after ts.
func After(ts Timestamp) TimestampRange {
	return TimestampRange{Start: ts + 1, End: MaxTimestamp}
}

// Contains returns true if ts is within the range.
func (r TimestampRange) Contains(ts Timestamp) bool {
	return ts >= r.Start && ts <= r.End
}

// Interval represents a key range for index scans.
type Interval struct {
	// Start is the inclusive lower bound of the key range.
	// Nil means unbounded (start from beginning).
	Start []byte

	// End is the exclusive upper bound of the key range.
	// Nil means unbounded (scan to end).
	End []byte
}

// All returns an interval covering all keys.
func All() Interval {
	return Interval{}
}

// Prefix returns an interval matching all keys with the given prefix.
func Prefix(prefix []byte) Interval {
	if len(prefix) == 0 {
		return All()
	}
	// End is prefix with last byte incremented (handles overflow)
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xFF {
			end[i]++
			return Interval{Start: prefix, End: end[:i+1]}
		}
	}
	// All bytes were 0xFF, so there's no upper bound
	return Interval{Start: prefix}
}

// Order specifies sort order for query results.
type Order int

const (
	// Asc sorts results in ascending order.
	Asc Order = iota
	// Desc sorts results in descending order.
	Desc
)

// String returns the string representation of the order.
func (o Order) String() string {
	if o == Desc {
		return "DESC"
	}
	return "ASC"
}

// IndexResult represents a result from an index scan.
type IndexResult struct {
	// Key is the index key that matched.
	Key []byte

	// Document is the referenced document (latest non-deleted version).
	Document *DocumentLogEntry
}

// GlobalKey identifies a global persistence value.
type GlobalKey string

const (
	// GlobalMaxRepeatableTS is the maximum timestamp that can be safely read.
	GlobalMaxRepeatableTS GlobalKey = "max_repeatable_ts"

	// GlobalSchemaVersion tracks the schema version for migrations.
	GlobalSchemaVersion GlobalKey = "schema_version"
)
