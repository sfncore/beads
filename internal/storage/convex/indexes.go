package convex

import (
	"encoding/binary"

	"github.com/steveyegge/beads/internal/types"
)

// IndexGenerator manages secondary indexes for efficient queries.
type IndexGenerator struct {
	// Could maintain index state to optimize updates
	// For now, simple key generation is sufficient
}

// NewIndexGenerator creates a new index generator.
func NewIndexGenerator() *IndexGenerator {
	return &IndexGenerator{}
}

// IndexIssue creates index entries for an issue document.
func (g *IndexGenerator) IndexIssue(issue *types.Issue, ts Timestamp) []IndexEntry {
	var indexes []IndexEntry

	// Status index
	if issue.Status != "" {
		indexes = append(indexes, IndexEntry{
			IndexID:    "issues_by_status",
			TS:         ts,
			Key:        g.StatusIndexKey(types.Status(issue.Status)),
			TableID:    "issues",
			DocumentID: issue.ID,
		})
	}

	// Priority index
	indexes = append(indexes, IndexEntry{
		IndexID:    "issues_by_priority",
		TS:         ts,
		Key:        g.PriorityIndexKey(issue.Priority),
		TableID:    "issues",
		DocumentID: issue.ID,
	})

	// Issue type index
	if issue.IssueType != "" {
		indexes = append(indexes, IndexEntry{
			IndexID:    "issues_by_type",
			TS:         ts,
			Key:        g.TypeIndexKey(types.IssueType(issue.IssueType)),
			TableID:    "issues",
			DocumentID: issue.ID,
		})
	}

	// Parent index
	if issue.Parent != "" {
		indexes = append(indexes, IndexEntry{
			IndexID:    "issues_by_parent",
			TS:         ts,
			Key:        g.ParentIndexKey(issue.Parent),
			TableID:    "issues",
			DocumentID: issue.ID,
		})
	}

	// Assignee index
	if issue.Assignee != "" {
		indexes = append(indexes, IndexEntry{
			IndexID:    "issues_by_assignee",
			TS:         ts,
			Key:        g.AssigneeIndexKey(issue.Assignee),
			TableID:    "issues",
			DocumentID: issue.ID,
		})
	}

	// Label indexes
	if issue.Labels != nil {
		for _, label := range issue.Labels {
			indexes = append(indexes, IndexEntry{
				IndexID:    "issues_by_label",
				TS:         ts,
				Key:        g.LabelIndexKey(label),
				TableID:    "issues",
				DocumentID: issue.ID,
			})
		}
	}

	return indexes
}

// StatusIndexKey creates an index key for status queries.
func (g *IndexGenerator) StatusIndexKey(status types.Status) []byte {
	return []byte(string(status) + "\x00")
}

// PriorityIndexKey creates an index key for priority queries.
func (g *IndexGenerator) PriorityIndexKey(priority int) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(priority))
	return append(buf, '\x00')
}

// TypeIndexKey creates an index key for issue type queries.
func (g *IndexGenerator) TypeIndexKey(issueType types.IssueType) []byte {
	return []byte(string(issueType) + "\x00")
}

// ParentIndexKey creates an index key for parent queries.
func (g *IndexGenerator) ParentIndexKey(parent string) []byte {
	return []byte(parent + "\x00")
}

// AssigneeIndexKey creates an index key for assignee queries.
func (g *IndexGenerator) AssigneeIndexKey(assignee string) []byte {
	return []byte(assignee + "\x00")
}

// LabelIndexKey creates an index key for label queries.
func (g *IndexGenerator) LabelIndexKey(label string) []byte {
	return []byte(label + "\x00")
}
