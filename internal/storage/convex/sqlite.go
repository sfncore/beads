package convex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	// Import the WASM-based SQLite driver (same as beads uses)
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// SQLitePersistence implements the Persistence interface using SQLite.
// It stores documents as JSON with full temporal versioning.
type SQLitePersistence struct {
	db     *sql.DB
	dbPath string
	fresh  bool
	mu     sync.RWMutex
}

// NewSQLitePersistence creates a new SQLite-backed persistence store.
// If the database doesn't exist, it will be created with the schema.
func NewSQLitePersistence(ctx context.Context, dbPath string) (*SQLitePersistence, error) {
	// Check if this is a fresh database
	fresh := false
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fresh = true
		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("creating database directory: %w", err)
		}
	}

	// Build connection string with pragmas for optimal performance
	connStr := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)

	// Initialize schema if needed
	if fresh {
		if _, err := db.ExecContext(ctx, Schema); err != nil {
			db.Close()
			return nil, fmt.Errorf("initializing schema: %w", err)
		}

		// Set initial schema version
		versionJSON, _ := json.Marshal(SchemaVersion)
		if _, err := db.ExecContext(ctx, SetGlobalQuery, GlobalSchemaVersion, string(versionJSON)); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting schema version: %w", err)
		}
	}

	return &SQLitePersistence{
		db:     db,
		dbPath: dbPath,
		fresh:  fresh,
	}, nil
}

// IsFresh returns true if this is a newly created database.
func (p *SQLitePersistence) IsFresh() bool {
	return p.fresh
}

// Reader returns a PersistenceReader for query operations.
func (p *SQLitePersistence) Reader() PersistenceReader {
	return &sqliteReader{p: p}
}

// Write atomically writes documents and index entries.
func (p *SQLitePersistence) Write(ctx context.Context, documents []DocumentLogEntry, indexes []IndexEntry) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert documents
	docStmt, err := tx.PrepareContext(ctx, InsertDocumentQuery)
	if err != nil {
		return fmt.Errorf("preparing document insert: %w", err)
	}
	defer docStmt.Close()

	for _, doc := range documents {
		var jsonValue interface{}
		if doc.Value != nil {
			jsonValue = string(doc.Value)
		}

		var deletedInt int
		if doc.Deleted {
			deletedInt = 1
		}

		var prevTS interface{}
		if doc.PrevTS != nil {
			prevTS = int64(*doc.PrevTS)
		}

		if _, err := docStmt.ExecContext(ctx, doc.ID, int64(doc.TS), doc.TableID, jsonValue, deletedInt, prevTS); err != nil {
			return fmt.Errorf("inserting document %s: %w", doc.ID, err)
		}
	}

	// Insert indexes
	if len(indexes) > 0 {
		idxStmt, err := tx.PrepareContext(ctx, InsertIndexQuery)
		if err != nil {
			return fmt.Errorf("preparing index insert: %w", err)
		}
		defer idxStmt.Close()

		for _, idx := range indexes {
			var deletedInt int
			if idx.Deleted {
				deletedInt = 1
			}

			if _, err := idxStmt.ExecContext(ctx, idx.IndexID, int64(idx.TS), idx.Key, deletedInt, idx.TableID, idx.DocumentID); err != nil {
				return fmt.Errorf("inserting index entry: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// WriteGlobal writes a global key-value pair.
func (p *SQLitePersistence) WriteGlobal(ctx context.Context, key GlobalKey, value json.RawMessage) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := p.db.ExecContext(ctx, SetGlobalQuery, string(key), string(value))
	if err != nil {
		return fmt.Errorf("writing global %s: %w", key, err)
	}
	return nil
}

// GetGlobal retrieves a global value by key.
func (p *SQLitePersistence) GetGlobal(ctx context.Context, key GlobalKey) (json.RawMessage, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var value string
	err := p.db.QueryRowContext(ctx, GetGlobalQuery, string(key)).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading global %s: %w", key, err)
	}
	return json.RawMessage(value), nil
}

// Close closes the database connection.
func (p *SQLitePersistence) Close() error {
	return p.db.Close()
}

// Path returns the database file path.
func (p *SQLitePersistence) Path() string {
	return p.dbPath
}

// sqliteReader implements PersistenceReader for SQLite.
type sqliteReader struct {
	p *SQLitePersistence
}

// LoadDocuments returns documents from a table within the timestamp range.
func (r *sqliteReader) LoadDocuments(ctx context.Context, tableID string, tsRange TimestampRange, order Order) ([]DocumentLogEntry, error) {
	r.p.mu.RLock()
	defer r.p.mu.RUnlock()

	query := fmt.Sprintf(DocumentsByTableQuery, order.String())
	rows, err := r.p.db.QueryContext(ctx, query, tableID, int64(tsRange.Start), int64(tsRange.End))
	if err != nil {
		return nil, fmt.Errorf("querying documents: %w", err)
	}
	defer rows.Close()

	var docs []DocumentLogEntry
	for rows.Next() {
		var doc DocumentLogEntry
		var ts, deletedInt int64
		var jsonValue sql.NullString
		var prevTS sql.NullInt64

		if err := rows.Scan(&doc.ID, &ts, &doc.TableID, &jsonValue, &deletedInt, &prevTS); err != nil {
			return nil, fmt.Errorf("scanning document: %w", err)
		}

		doc.TS = Timestamp(ts)
		doc.Deleted = deletedInt == 1
		if jsonValue.Valid {
			doc.Value = json.RawMessage(jsonValue.String)
		}
		if prevTS.Valid {
			prev := Timestamp(prevTS.Int64)
			doc.PrevTS = &prev
		}

		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating documents: %w", err)
	}

	return docs, nil
}

// GetDocument returns the latest non-deleted version of a document.
func (r *sqliteReader) GetDocument(ctx context.Context, tableID string, docID string, atTS *Timestamp) (*DocumentLogEntry, error) {
	r.p.mu.RLock()
	defer r.p.mu.RUnlock()

	var doc DocumentLogEntry
	var ts, deletedInt int64
	var jsonValue sql.NullString
	var prevTS sql.NullInt64

	var err error
	if atTS != nil {
		err = r.p.db.QueryRowContext(ctx, LatestDocumentAtTSQuery, tableID, docID, int64(*atTS)).Scan(
			&doc.ID, &ts, &doc.TableID, &jsonValue, &deletedInt, &prevTS,
		)
	} else {
		err = r.p.db.QueryRowContext(ctx, LatestDocumentQuery, tableID, docID).Scan(
			&doc.ID, &ts, &doc.TableID, &jsonValue, &deletedInt, &prevTS,
		)
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying document %s/%s: %w", tableID, docID, err)
	}

	doc.TS = Timestamp(ts)
	doc.Deleted = deletedInt == 1
	if jsonValue.Valid {
		doc.Value = json.RawMessage(jsonValue.String)
	}
	if prevTS.Valid {
		prev := Timestamp(prevTS.Int64)
		doc.PrevTS = &prev
	}

	// If deleted, return nil (tombstone)
	if doc.Deleted {
		return nil, nil
	}

	return &doc, nil
}

// GetDocuments returns the latest non-deleted version of multiple documents.
func (r *sqliteReader) GetDocuments(ctx context.Context, tableID string, docIDs []string, atTS *Timestamp) (map[string]*DocumentLogEntry, error) {
	result := make(map[string]*DocumentLogEntry, len(docIDs))
	for _, id := range docIDs {
		doc, err := r.GetDocument(ctx, tableID, id, atTS)
		if err != nil {
			return nil, err
		}
		if doc != nil {
			result[id] = doc
		}
	}
	return result, nil
}

// IndexScan scans an index within the given key interval.
func (r *sqliteReader) IndexScan(ctx context.Context, indexID string, interval Interval, readTS Timestamp, order Order, limit int) ([]IndexResult, error) {
	r.p.mu.RLock()
	defer r.p.mu.RUnlock()

	if readTS == 0 {
		readTS = Now()
	}

	query := fmt.Sprintf(IndexScanQuery, order.String())

	var endKey interface{}
	if interval.End != nil {
		endKey = interval.End
	}

	startKey := interval.Start
	if startKey == nil {
		startKey = []byte{}
	}

	rows, err := r.p.db.QueryContext(ctx, query, indexID, startKey, endKey, endKey, int64(readTS), int64(readTS), limit)
	if err != nil {
		return nil, fmt.Errorf("scanning index %s: %w", indexID, err)
	}
	defer rows.Close()

	var results []IndexResult
	for rows.Next() {
		var doc DocumentLogEntry
		var ts, deletedInt int64
		var jsonValue sql.NullString
		var prevTS sql.NullInt64

		if err := rows.Scan(&doc.ID, &ts, &doc.TableID, &jsonValue, &deletedInt, &prevTS); err != nil {
			return nil, fmt.Errorf("scanning index result: %w", err)
		}

		doc.TS = Timestamp(ts)
		doc.Deleted = deletedInt == 1
		if jsonValue.Valid {
			doc.Value = json.RawMessage(jsonValue.String)
		}
		if prevTS.Valid {
			prev := Timestamp(prevTS.Int64)
			doc.PrevTS = &prev
		}

		results = append(results, IndexResult{Document: &doc})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating index results: %w", err)
	}

	return results, nil
}

// IndexGet performs a point lookup on an index.
func (r *sqliteReader) IndexGet(ctx context.Context, indexID string, key []byte, readTS Timestamp) (*DocumentLogEntry, error) {
	r.p.mu.RLock()
	defer r.p.mu.RUnlock()

	if readTS == 0 {
		readTS = Now()
	}

	var doc DocumentLogEntry
	var ts, deletedInt int64
	var jsonValue sql.NullString
	var prevTS sql.NullInt64

	err := r.p.db.QueryRowContext(ctx, IndexGetQuery, indexID, key, int64(readTS), int64(readTS)).Scan(
		&doc.ID, &ts, &doc.TableID, &jsonValue, &deletedInt, &prevTS,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("index get %s: %w", indexID, err)
	}

	doc.TS = Timestamp(ts)
	doc.Deleted = deletedInt == 1
	if jsonValue.Valid {
		doc.Value = json.RawMessage(jsonValue.String)
	}
	if prevTS.Valid {
		prev := Timestamp(prevTS.Int64)
		doc.PrevTS = &prev
	}

	return &doc, nil
}

// MaxTimestamp returns the maximum timestamp written to the store.
func (r *sqliteReader) MaxTimestamp(ctx context.Context) (Timestamp, error) {
	r.p.mu.RLock()
	defer r.p.mu.RUnlock()

	var ts int64
	err := r.p.db.QueryRowContext(ctx, MaxTimestampQuery).Scan(&ts)
	if err != nil {
		return 0, fmt.Errorf("querying max timestamp: %w", err)
	}
	return Timestamp(ts), nil
}

// DocumentCount returns the approximate count of non-deleted documents in a table.
func (r *sqliteReader) DocumentCount(ctx context.Context, tableID string) (int64, error) {
	r.p.mu.RLock()
	defer r.p.mu.RUnlock()

	var count int64
	err := r.p.db.QueryRowContext(ctx, DocumentCountQuery, tableID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting documents in %s: %w", tableID, err)
	}
	return count, nil
}

// Compile-time check that SQLitePersistence implements Persistence
var _ Persistence = (*SQLitePersistence)(nil)

// Compile-time check that sqliteReader implements PersistenceReader
var _ PersistenceReader = (*sqliteReader)(nil)
