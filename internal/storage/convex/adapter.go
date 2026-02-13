// Package convex provides a Convex-style document persistence layer for beads.
package convex

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/steveyegge/beads/internal/types"
)

// ConvexStorageAdapter implements storage.Storage interface using convex persistence.
// It bridges beads' relational model with Convex's document model.
type ConvexStorageAdapter struct {
	persistence Persistence
	clock       func() Timestamp
	idxGen      *IndexGenerator
}

// NewConvexStorageAdapter creates a new adapter.
func NewConvexStorageAdapter(p Persistence) *ConvexStorageAdapter {
	return &ConvexStorageAdapter{
		persistence: p,
		clock:       Now,
		idxGen:      NewIndexGenerator(),
	}
}

// withClock sets a custom clock for deterministic tests.
func (a *ConvexStorageAdapter) withClock(clock func() Timestamp) *ConvexStorageAdapter {
	a.clock = clock
	return a
}

// CreateIssue creates a new issue document and necessary indexes.
func (a *ConvexStorageAdapter) CreateIssue(ctx context.Context, issue *types.Issue, actor string) error {
	// Serialize issue to JSON
	jsonValue, err := json.Marshal(issue)
	if err != nil {
		return fmt.Errorf("serializing issue %s: %w", issue.ID, err)
	}

	// Create document entry
	ts := a.clock()
	doc := DocumentLogEntry{
		TS:      ts,
		ID:      issue.ID,
		TableID: "issues",
		Value:   json.RawMessage(jsonValue),
		Deleted: false,
	}

	// Generate index entries
	indexes := a.idxGen.IndexIssue(issue, ts)

	// Write atomically
	return a.persistence.Write(ctx, []DocumentLogEntry{doc}, indexes)
}

// CreateIssues creates multiple issues in a single transaction.
func (a *ConvexStorageAdapter) CreateIssues(ctx context.Context, issues []*types.Issue, actor string) error {
	if len(issues) == 0 {
		return nil
	}

	var batch WriteBatch
	ts := a.clock()

	for _, issue := range issues {
		// Serialize issue to JSON
		jsonValue, err := json.Marshal(issue)
		if err != nil {
			return fmt.Errorf("serializing issue %s: %w", issue.ID, err)
		}

		// Create document entry
		doc := DocumentLogEntry{
			TS:      ts,
			ID:      issue.ID,
			TableID: "issues",
			Value:   json.RawMessage(jsonValue),
			Deleted: false,
		}
		batch.AddDocument(doc)

		// Add indexes
		indexes := a.idxGen.IndexIssue(issue, ts)
		for _, idx := range indexes {
			batch.AddIndex(idx)
		}
	}

	return a.persistence.Write(ctx, batch.Documents, batch.Indexes)
}

// GetIssue retrieves a single issue by ID.
func (a *ConvexStorageAdapter) GetIssue(ctx context.Context, id string) (*types.Issue, error) {
	doc, err := a.persistence.Reader().GetDocument(ctx, "issues", id, nil)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("issue %s not found", id)
	}

	// Deserialize JSON
	var issue types.Issue
	if err := json.Unmarshal(doc.Value, &issue); err != nil {
		return nil, fmt.Errorf("deserializing issue %s: %w", id, err)
	}

	return &issue, nil
}

// GetIssueByExternalRef finds issue by external reference.
func (a *ConvexStorageAdapter) GetIssueByExternalRef(ctx context.Context, externalRef string) (*types.Issue, error) {
	// Query all issues and filter by external_ref
	docs, err := a.persistence.Reader().LoadDocuments(ctx, "issues", AllTime(), Asc)
	if err != nil {
		return nil, err
	}

	for _, doc := range docs {
		if doc.Deleted || doc.Value == nil {
			continue
		}

		var issue types.Issue
		if err := json.Unmarshal(doc.Value, &issue); err != nil {
			continue
		}

		if issue.ExternalRef != nil && *issue.ExternalRef == externalRef {
			return &issue, nil
		}
	}

	return nil, fmt.Errorf("issue with external_ref %s not found", externalRef)
}

// UpdateIssue modifies an existing issue.
func (a *ConvexStorageAdapter) UpdateIssue(ctx context.Context, id string, updates map[string]interface{}, actor string) error {
	// Get current issue to build new version
	current, err := a.GetIssue(ctx, id)
	if err != nil {
		return err
	}

	// Apply updates to current issue
	if err := applyUpdates(current, updates); err != nil {
		return fmt.Errorf("applying updates to issue %s: %w", id, err)
	}

	// Update timestamp
	current.UpdatedAt = time.Now()

	// Serialize updated issue
	jsonValue, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("serializing updated issue %s: %w", id, err)
	}

	// Create new document version
	ts := a.clock()
	doc := DocumentLogEntry{
		TS:      ts,
		ID:      id,
		TableID: "issues",
		Value:   json.RawMessage(jsonValue),
		Deleted: false,
		PrevTS:  &Timestamp(current.CreatedAt.UnixNano()),
	}

	// Update indexes
	oldTS := Timestamp(current.CreatedAt.UnixNano())
	newTS := ts
	newIndexKeys := a.idxGen.IndexIssue(current, oldTS)
	var indexUpdates []IndexEntry
	for _, newKey := range newIndexKeys {
		indexUpdates = append(indexUpdates, newKey)
	}
	var indexUpdates []IndexEntry
	for _, newKey := range newIndexKeys {
		indexUpdates = append(indexUpdates, newKey)
	}

	// Write atomically
	return a.persistence.Write(ctx, []DocumentLogEntry{doc}, indexUpdates)
}

// CloseIssue marks an issue as closed.
func (a *ConvexStorageAdapter) CloseIssue(ctx context.Context, id string, reason string, actor string, session string) error {
	// Get current issue
	current, err := a.GetIssue(ctx, id)
	if err != nil {
		return err
	}

	// Update fields for closing
	current.Status = types.StatusClosed
	now := time.Now()
	current.ClosedAt = &now
	current.CloseReason = reason
	current.ClosedBySession = session

	// Serialize updated issue
	jsonValue, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("serializing closed issue %s: %w", id, err)
	}

	// Create new document version
	ts := a.clock()
	doc := DocumentLogEntry{
		TS:      ts,
		ID:      id,
		TableID: "issues",
		Value:   json.RawMessage(jsonValue),
		Deleted: false,
		PrevTS:  &Timestamp(current.CreatedAt.UnixNano()),
	}

	// Update status index
	indexUpdates := []IndexEntry{
		{
			IndexID:    "issues_by_status",
			TS:         ts,
			Key:        a.idxGen.StatusIndexKey(types.StatusClosed),
			TableID:    "issues",
			DocumentID: id,
		},
	}

	return a.persistence.Write(ctx, []DocumentLogEntry{doc}, indexUpdates)
}

// DeleteIssue marks an issue as deleted (tombstone).
func (a *ConvexStorageAdapter) DeleteIssue(ctx context.Context, id string) error {
	// Create tombstone entry
	ts := a.clock()
	doc := DocumentLogEntry{
		TS:      ts,
		ID:      id,
		TableID: "issues",
		Value:   nil,
		Deleted: true,
	}

	// Indexes are tombstoned automatically by schema queries
	return a.persistence.Write(ctx, []DocumentLogEntry{doc}, nil)
}

// SearchIssues searches issues with filters.
func (a *ConvexStorageAdapter) SearchIssues(ctx context.Context, query string, filter types.IssueFilter) ([]*types.Issue, error) {
	// Build index scan based on filters
	var indexID string
	var interval Interval
	var limit int = 100 // Default limit

	if filter.Status != nil {
		indexID = "issues_by_status"
		interval = Prefix([]byte(string(*filter.Status) + "\x00"))
	}

	if filter.Priority != nil {
		if indexID != "" {
			// TODO: Implement more efficient multi-index queries
		}
		indexID = "issues_by_priority"
		interval = Prefix(a.idxGen.PriorityIndexKey(*filter.Priority))
	}

	if filter.Labels != nil {
		// TODO: Handle multiple labels - for now, scan all and filter in memory
	}

	if filter.ParentID != nil {
		indexID = "issues_by_parent"
		interval = Prefix([]byte(*filter.ParentID + "\x00"))
	}

	if filter.Assignee != nil {
		indexID = "issues_by_assignee"
		interval = Prefix([]byte(*filter.Assignee + "\x00"))
	}

	if filter.NoAssignee {
		indexID = "issues_unassigned"
		interval = Prefix([]byte("unassigned\x00"))
	}

	// If no specific index, scan all and filter in memory
	if indexID == "" {
		docs, err := a.persistence.Reader().LoadDocuments(ctx, "issues", AllTime(), Asc)
		if err != nil {
			return nil, err
		}
		return a.filterIssues(docs, filter)
	}

	// Use index scan
	results, err := a.persistence.Reader().IndexScan(ctx, indexID, interval, 0, Desc, limit)
	if err != nil {
		return nil, err
	}

	var issues []*types.Issue
	for _, result := range results {
		var issue types.Issue
		if err := json.Unmarshal(result.Document.Value, &issue); err != nil {
			continue
		}
		issues = append(issues, &issue)
	}

	return a.filterIssues(issues, filter), nil
}

// AddDependency adds a dependency relationship.
func (a *ConvexStorageAdapter) AddDependency(ctx context.Context, dep *types.Dependency, actor string) error {
	// Serialize dependency
	jsonValue, err := json.Marshal(dep)
	if err != nil {
		return fmt.Errorf("serializing dependency: %w", err)
	}

	// Create document
	ts := a.clock()
	doc := DocumentLogEntry{
		TS:      ts,
		ID:      dep.ID,
		TableID: "dependencies",
		Value:   json.RawMessage(jsonValue),
		Deleted: false,
	}

	// Create indexes
	indexes := []IndexEntry{
		{
			IndexID:    "dependencies_by_issue",
			TS:         ts,
			Key:        Prefix([]byte(dep.IssueID + "\x00")),
			TableID:    "dependencies",
			DocumentID: dep.ID,
		},
		{
			IndexID:    "dependencies_by_depends_on",
			TS:         ts,
			Key:        Prefix([]byte(dep.DependsOnID + "\x00")),
			TableID:    "dependencies",
			DocumentID: dep.ID,
		},
	}

	return a.persistence.Write(ctx, []DocumentLogEntry{doc}, indexes)
}

// RemoveDependency removes a dependency (tombstone).
func (a *ConvexStorageAdapter) RemoveDependency(ctx context.Context, issueID, dependsOnID string, actor string) error {
	// Find dependency document by scanning
	interval := Prefix([]byte(dependsOnID + "\x00"))
	results, err := a.persistence.Reader().IndexScan(ctx, "dependencies_by_depends_on", interval, 0, Desc, 1)
	if err != nil {
		return err
	}

	for _, result := range results {
		var dep types.Dependency
		if err := json.Unmarshal(result.Document.Value, &dep); err != nil {
			continue
		}

		if dep.IssueID == issueID && dep.DependsOnID == dependsOnID {
			// Create tombstone
			ts := a.clock()
			doc := DocumentLogEntry{
				TS:      ts,
				ID:      dep.ID,
				TableID: "dependencies",
				Value:   nil,
				Deleted:  true,
			}
			return a.persistence.Write(ctx, []DocumentLogEntry{doc}, nil)
		}
	}

	}

return 
}

// GetDependencies returns all dependencies for an issue.
// GetDependencies returns all dependencies for an issue.
func (a *ConvexStorageAdapter) GetDependencies(ctx context.Context, issueID string) ([]*types.Issue, error) {
	// Simple placeholder for now
	return []*types.Issue{}, nil
}
}

// AddLabel adds a label to an issue.
func (a *ConvexStorageAdapter) AddLabel(ctx context.Context, issueID, label, actor string) error {
	// Get current issue to add label to it
	issue, err := a.GetIssue(ctx, issueID)
	if err != nil {
		return err
	}

	// Add label to array (avoid duplicates)
	if issue.Labels == nil {
		issue.Labels = []string{label}
	} else {
		for _, l := range issue.Labels {
			if l == label {
				return nil // Already has label
			}
		}
		issue.Labels = append(issue.Labels, label)
	}

	// Update issue (this will also update the labels index)
	return a.UpdateIssue(ctx, issueID, map[string]interface{}{
		"labels": issue.Labels,
	}, actor)
}

// RemoveLabel removes a label from an issue.
func (a *ConvexStorageAdapter) RemoveLabel(ctx context.Context, issueID, label, actor string) error {
	issue, err := a.GetIssue(ctx, issueID)
	if err != nil {
		return err
	}

	// Remove label from array
	if issue.Labels != nil {
		var newLabels []string
		for _, l := range issue.Labels {
			if l != label {
				newLabels = append(newLabels, l)
			}
		}
		issue.Labels = newLabels
	}

	// Update issue
	return a.UpdateIssue(ctx, issueID, map[string]interface{}{
		"labels": newLabels,
	}, actor)
}

// GetLabels returns all labels for an issue.
func (a *ConvexStorageAdapter) GetLabels(ctx context.Context, issueID string) ([]string, error) {
	issue, err := a.GetIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue.Labels == nil {
		return []string{}, nil
	}
	return issue.Labels, nil
}

// AddComment adds a comment to an issue.
func (a *ConvexStorageAdapter) AddComment(ctx context.Context, issueID, actor, comment string) error {
	// Create comment document
	ts := a.clock()
	doc := DocumentLogEntry{
		TS:      ts,
		ID:      generateCommentID(issueID, ts),
		TableID: "comments",
		Value:   json.RawMessage(fmt.Sprintf(`{"issue_id":"%s","author":"%s","text":"%s","created_at":"%s"}`, issueID, actor, comment, ts.Time().Format(time.RFC3339Nano))),
		Deleted: false,
	}

	// Add indexes
	indexes := []IndexEntry{
		{
			IndexID:    "comments_by_issue",
			TS:         ts,
			Key:        Prefix([]byte(issueID + "\x00")),
			TableID:    "comments",
			DocumentID: doc.ID,
		},
	}

	return a.persistence.Write(ctx, []DocumentLogEntry{doc}, indexes)
}

// GetEvents returns event history for an issue.
func (a *ConvexStorageAdapter) GetEvents(ctx context.Context, issueID string, limit int) ([]*types.Event, error) {
	interval := Prefix([]byte(issueID + "\x00"))
	results, err := a.persistence.Reader().IndexScan(ctx, "events_by_issue", interval, 0, Desc, limit)
	if err != nil {
		return nil, err
	}

	var events []*types.Event
	for _, result := range results {
		var event types.Event
		if err := json.Unmarshal(result.Document.Value, &event); err != nil {
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}

// GetStatistics returns database statistics.
func (a *ConvexStorageAdapter) GetStatistics(ctx context.Context) (*types.Statistics, error) {
	// Count issues by status
	statusCounts := make(map[types.Status]int64)
	for _, status := range []types.Status{types.StatusOpen, types.StatusInProgress, types.StatusClosed} {
		interval := Prefix([]byte(string(status) + "\x00"))
		results, err := a.persistence.Reader().IndexScan(ctx, "issues_by_status", interval, 0, Desc, 1000)
		if err == nil {
			statusCounts[status] = int64(len(results))
		}
	}

	// Count total documents
	totalCount, err := a.persistence.Reader().DocumentCount(ctx, "issues")
	if err != nil {
		return nil, fmt.Errorf("getting document count: %w", err)
	}

	// For now, return a simplified statistics
	return &types.Statistics{
		TotalIssues: totalCount,
		OpenIssues:   statusCounts[types.StatusOpen],
	}, nil
}

// RunInTransaction executes operations within a persistence transaction.
func (a *ConvexStorageAdapter) RunInTransaction(ctx context.Context, fn func(storage.Transaction) error) error {
	// For simplicity, batch all operations
	// In a real implementation, we'd need transaction support in persistence
	var batch WriteBatch

	// Execute callback to collect operations
	err := fn(&convexTransaction{adapter: a, batch: &batch})
	if err != nil {
		return err
	}

	// Commit the batch
	if len(batch.Documents) > 0 || len(batch.Indexes) > 0 {
		return a.persistence.Write(ctx, batch.Documents, batch.Indexes)
	}

	return nil
}

// Close closes the adapter and underlying persistence.
func (a *ConvexStorageAdapter) Close() error {
	return a.persistence.Close()
}

// Path returns the persistence store path.
func (a *ConvexStorageAdapter) Path() string {
	return a.persistence.Path()
}

// UnderlyingDB returns the underlying database connection for extensions.
func (a *ConvexStorageAdapter) UnderlyingDB() interface{} {
	// Try type assertion to access underlying DB
	if dbGetter, ok := a.persistence.(interface{ UnderlyingDB() interface{} }); ok {
		return dbGetter.UnderlyingDB()
	}
	return nil
}

// UnderlyingConn returns a database connection.
func (a *ConvexStorageAdapter) UnderlyingConn(ctx context.Context) (interface{}, error) {
	if connGetter, ok := a.persistence.(interface {
		UnderlyingConn(context.Context) (interface{}, error)
	}); ok {
		return connGetter.UnderlyingConn(ctx)
	}
	return nil, fmt.Errorf("UnderlyingConn not supported by this persistence implementation")
}

// Helper functions

// applyUpdates applies map updates to an issue struct
func applyUpdates(issue *types.Issue, updates map[string]interface{}) error {
	for key, value := range updates {
		switch key {
		case "title":
			if v, ok := value.(string); ok {
				issue.Title = v
			}
		case "description":
			if v, ok := value.(string); ok {
				issue.Description = v
			}
		case "status":
			if v, ok := value.(string); ok {
				issue.Status = types.Status(v)
			}
		case "priority":
			if v, ok := value.(float64); ok {
				issue.Priority = int(v)
			}
		case "labels":
			if v, ok := value.([]string); ok {
				issue.Labels = v
			}
		case "updated_at":
			if v, ok := value.(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					issue.UpdatedAt = t
				}
			}
		case "closed_at":
			if v, ok := value.(string); ok {
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					issue.ClosedAt = &t
				}
			}
		case "close_reason":
			if v, ok := value.(string); ok {
				issue.CloseReason = v
			}
		case "closed_by_session":
			if v, ok := value.(string); ok {
				issue.ClosedBySession = v
			}
		case "assignee":
			if v, ok := value.(string); ok {
				issue.Assignee = v
			}
		}
	}
	return nil
}

// filterIssues applies filters to issue slice
func (a *ConvexStorageAdapter) filterIssues(issues []*types.Issue, filter types.IssueFilter) []*types.Issue {
	var result []*types.Issue

	for _, issue := range issues {
		if a.matchesFilter(issue, filter) {
			result = append(result, issue)
		}
	}

	return result
}

// matchesFilter checks if an issue matches the given filter
func (a *ConvexStorageAdapter) matchesFilter(issue *types.Issue, filter types.IssueFilter) bool {
	if filter.Status != nil && issue.Status != *filter.Status {
		return false
	}

	if filter.Priority != nil && issue.Priority != *filter.Priority {
		return false
	}

	if filter.Labels != nil {
		found := false
		for _, label := range issue.Labels {
			for _, filterLabel := range filter.Labels {
				if label == filterLabel {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	if filter.ParentID != nil && issue.Parent != *filter.ParentID {
		return false
	}

	if filter.Assignee != nil && issue.Assignee != *filter.Assignee {
		return false
	}

	if filter.NoAssignee && (issue.Assignee != nil && *issue.Assignee != "") {
		return false
	}

	return true
}

// generateCommentID creates a unique comment ID.
func generateCommentID(issueID string, ts Timestamp) string {
	return fmt.Sprintf("%s-%d", issueID, int64(ts))
}

// Placeholder implementations for remaining Storage interface methods
// TODO: Implement full compatibility in Phase 2

func (a *ConvexStorageAdapter) GetDependents(ctx context.Context, issueID string) ([]*types.Issue, error) {
	return nil, fmt.Errorf("GetDependents not implemented yet")
}

func (a *ConvexStorageAdapter) GetReadyWork(ctx context.Context, filter types.WorkFilter) ([]*types.Issue, error) {
	return nil, fmt.Errorf("GetReadyWork not implemented yet")
}

func (a *ConvexStorageAdapter) AddIssueComment(ctx context.Context, issueID, author, text string) (*types.Comment, error) {
	return nil, fmt.Errorf("AddIssueComment not implemented yet")
}

func (a *ConvexStorageAdapter) GetBlockedIssues(ctx context.Context, filter types.WorkFilter) ([]*types.BlockedIssue, error) {
	return nil, fmt.Errorf("GetBlockedIssues not implemented yet")
}

func (a *ConvexStorageAdapter) IsBlocked(ctx context.Context, issueID string) (bool, []string, error) {
	return false, nil, fmt.Errorf("IsBlocked not implemented yet")
}

// convexTransaction implements storage.Transaction for convex backend.
type convexTransaction struct {
	adapter *ConvexStorageAdapter
	batch   *WriteBatch
}

func (t *convexTransaction) CreateIssue(ctx context.Context, issue *types.Issue, actor string) error {
	return t.adapter.CreateIssue(ctx, issue, actor)
}

func (t *convexTransaction) UpdateIssue(ctx context.Context, id string, updates map[string]interface{}, actor string) error {
	return t.adapter.UpdateIssue(ctx, id, updates, actor)
}

func (t *convexTransaction) CloseIssue(ctx context.Context, id string, reason string, actor string, session string) error {
	return t.adapter.CloseIssue(ctx, id, reason, actor, session)
}

func (t *convexTransaction) DeleteIssue(ctx context.Context, id string) error {
	return t.adapter.DeleteIssue(ctx, id)
}

func (t *convexTransaction) AddDependency(ctx context.Context, dep *types.Dependency, actor string) error {
	return t.adapter.AddDependency(ctx, dep, actor)
}

func (t *convexTransaction) RemoveDependency(ctx context.Context, issueID, dependsOnID string, actor string) error {
	return t.adapter.RemoveDependency(ctx, issueID, dependsOnID, actor)
}

func (t *convexTransaction) AddLabel(ctx context.Context, issueID, label, actor string) error {
	return t.adapter.AddLabel(ctx, issueID, label, actor)
}

func (t *convexTransaction) RemoveLabel(ctx context.Context, issueID, label, actor string) error {
	return t.adapter.RemoveLabel(ctx, issueID, label, actor)
}

func (t *convexTransaction) SetConfig(ctx context.Context, key, value string) error {
	return t.adapter.SetConfig(ctx, key, value)
}

func (t *convexTransaction) GetConfig(ctx context.Context, key string) (string, error) {
	return t.adapter.GetConfig(ctx, key)
}

func (t *convexTransaction) SetMetadata(ctx context.Context, key, value string) error {
	return t.adapter.SetMetadata(ctx, key, value)
}

func (t *convexTransaction) GetMetadata(ctx context.Context, key string) (string, error) {
	return t.adapter.GetMetadata(ctx, key)
}

func (t *convexTransaction) AddComment(ctx context.Context, issueID, actor, comment string) error {
	return t.adapter.AddComment(ctx, issueID, actor, comment)
}

// Additional methods for full storage.Storage compatibility
func (a *ConvexStorageAdapter) GetDependenciesWithMetadata(ctx context.Context, issueID string) ([]*types.IssueWithDependencyMetadata, error) {
	return nil, fmt.Errorf("GetDependenciesWithMetadata not implemented yet")
}

func (a *ConvexStorageAdapter) GetDependentsWithMetadata(ctx context.Context, issueID string) ([]*types.IssueWithDependencyMetadata, error) {
	return nil, fmt.Errorf("GetDependentsWithMetadata not implemented yet")
}

func (a *ConvexStorageAdapter) GetDependencyRecords(ctx context.Context, issueID string) ([]*types.Dependency, error) {
	return nil, fmt.Errorf("GetDependencyRecords not implemented yet")
}

func (a *ConvexStorageAdapter) GetAllDependencyRecords(ctx context.Context) (map[string][]*types.Dependency, error) {
	return nil, fmt.Errorf("GetAllDependencyRecords not implemented yet")
}

func (a *ConvexStorageAdapter) GetDependencyCounts(ctx context.Context, issueIDs []string) (map[string]*types.DependencyCounts, error) {
	return nil, fmt.Errorf("GetDependencyCounts not implemented yet")
}

func (a *ConvexStorageAdapter) GetDependencyTree(ctx context.Context, issueID string, maxDepth int, showAllPaths bool, reverse bool) ([]*types.TreeNode, error) {
	return nil, fmt.Errorf("GetDependencyTree not implemented yet")
}

func (a *ConvexStorageAdapter) DetectCycles(ctx context.Context) ([][]*types.Issue, error) {
	return nil, fmt.Errorf("DetectCycles not implemented yet")
}

func (a *ConvexStorageAdapter) GetIssuesByLabel(ctx context.Context, label string) ([]*types.Issue, error) {
	return nil, fmt.Errorf("GetIssuesByLabel not implemented yet")
}

func (a *ConvexStorageAdapter) GetLabelsForIssues(ctx context.Context, issueIDs []string) (map[string][]string, error) {
	return nil, fmt.Errorf("GetLabelsForIssues not implemented yet")
}

func (a *ConvexStorageAdapter) GetEpicsEligibleForClosure(ctx context.Context) ([]*types.EpicStatus, error) {
	return nil, fmt.Errorf("GetEpicsEligibleForClosure not implemented yet")
}

func (a *ConvexStorageAdapter) GetStaleIssues(ctx context.Context, filter types.StaleFilter) ([]*types.Issue, error) {
	return nil, fmt.Errorf("GetStaleIssues not implemented yet")
}

func (a *ConvexStorageAdapter) GetNewlyUnblockedByClose(ctx context.Context, closedIssueID string) ([]*types.Issue, error) {
	return nil, fmt.Errorf("GetNewlyUnblockedByClose not implemented yet")
}

func (a *ConvexStorageAdapter) GetIssueComments(ctx context.Context, issueID string) ([]*types.Comment, error) {
	return nil, fmt.Errorf("GetIssueComments not implemented yet")
}

func (a *ConvexStorageAdapter) GetCommentsForIssues(ctx context.Context, issueIDs []string) (map[string][]*types.Comment, error) {
	return nil, fmt.Errorf("GetCommentsForIssues not implemented yet")
}

func (a *ConvexStorageAdapter) GetMoleculeProgress(ctx context.Context, moleculeID string) (*types.MoleculeProgressStats, error) {
	return nil, fmt.Errorf("GetMoleculeProgress not implemented yet")
}

func (a *ConvexStorageAdapter) GetDirtyIssues(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("GetDirtyIssues not implemented yet")
}

func (a *ConvexStorageAdapter) GetDirtyIssueHash(ctx context.Context, issueID string) (string, error) {
	return "", fmt.Errorf("GetDirtyIssueHash not implemented yet")
}

func (a *ConvexStorageAdapter) ClearDirtyIssuesByID(ctx context.Context, issueIDs []string) error {
	return fmt.Errorf("ClearDirtyIssuesByID not implemented yet")
}

func (a *ConvexStorageAdapter) GetExportHash(ctx context.Context, issueID string) (string, error) {
	return "", fmt.Errorf("GetExportHash not implemented yet")
}

func (a *ConvexStorageAdapter) SetExportHash(ctx context.Context, issueID, contentHash string) error {
	return fmt.Errorf("SetExportHash not implemented yet")
}

func (a *ConvexStorageAdapter) ClearAllExportHashes(ctx context.Context) error {
	return fmt.Errorf("ClearAllExportHashes not implemented yet")
}

func (a *ConvexStorageAdapter) GetJSONLFileHash(ctx context.Context) (string, error) {
	return "", fmt.Errorf("GetJSONLFileHash not implemented yet")
}

func (a *ConvexStorageAdapter) SetJSONLFileHash(ctx context.Context, fileHash string) error {
	return fmt.Errorf("SetJSONLFileHash not implemented yet")
}

func (a *ConvexStorageAdapter) GetNextChildID(ctx context.Context, parentID string) (string, error) {
	return "", fmt.Errorf("GetNextChildID not implemented yet")
}

func (a *ConvexStorageAdapter) GetAllConfig(ctx context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("GetAllConfig not implemented yet")
}

func (a *ConvexStorageAdapter) DeleteConfig(ctx context.Context, key string) error {
	return fmt.Errorf("DeleteConfig not implemented yet")
}

func (a *ConvexStorageAdapter) GetCustomStatuses(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("GetCustomStatuses not implemented yet")
}

func (a *ConvexStorageAdapter) GetCustomTypes(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("GetCustomTypes not implemented yet")
}

func (a *ConvexStorageAdapter) UpdateIssueID(ctx context.Context, oldID, newID string, issue *types.Issue, actor string) error {
	return fmt.Errorf("UpdateIssueID not implemented yet")
}

func (a *ConvexStorageAdapter) RenameDependencyPrefix(ctx context.Context, oldPrefix, newPrefix string) error {
	return fmt.Errorf("RenameDependencyPrefix not implemented yet")
}

func (a *ConvexStorageAdapter) RenameCounterPrefix(ctx context.Context, oldPrefix, newPrefix string) error {
	return fmt.Errorf("RenameCounterPrefix not implemented yet")
}

func (a *ConvexStorageAdapter) SetConfig(ctx context.Context, key, value string) error {
	valueJSON := json.RawMessage(`"` + value + `"`)
	return a.persistence.WriteGlobal(ctx, GlobalKey(key), valueJSON)
}

func (a *ConvexStorageAdapter) GetConfig(ctx context.Context, key string) (string, error) {
	value, err := a.persistence.GetGlobal(ctx, GlobalKey(key))
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", fmt.Errorf("config key %s not found", key)
	}
	var result string
	if err := json.Unmarshal(value, &result); err != nil {
		return "", fmt.Errorf("unmarshaling config %s: %w", key, err)
	}
	return result, nil
}

func (a *ConvexStorageAdapter) SetMetadata(ctx context.Context, key, value string) error {
	valueJSON := json.RawMessage(`"` + value + `"`)
	return a.persistence.WriteGlobal(ctx, GlobalKey("metadata_"+key), valueJSON)
}

func (a *ConvexStorageAdapter) GetMetadata(ctx context.Context, key string) (string, error) {
	value, err := a.persistence.GetGlobal(ctx, GlobalKey("metadata_"+key))
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", fmt.Errorf("metadata key %s not found", key)
	}
	var result string
	if err := json.Unmarshal(value, &result); err != nil {
		return "", fmt.Errorf("unmarshaling metadata %s: %w", key, err)
	}
	return result, nil
}
