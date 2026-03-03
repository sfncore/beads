package dolt

import (
	"context"
	"testing"

	"github.com/steveyegge/beads/internal/types"
)

func TestGetDependenciesForIssues_Interface(t *testing.T) {
	// This test just verifies that the method exists and can be called
	// without compilation errors. Actual functionality is tested in integration tests.
	var _ interface {
		GetDependenciesForIssues(ctx context.Context, issueIDs []string) (map[string][]*types.Issue, error)
		GetDependentsForIssues(ctx context.Context, issueIDs []string) (map[string][]*types.Issue, error)
		GetDependenciesWithMetadataForIssues(ctx context.Context, issueIDs []string) (map[string][]*types.IssueWithDependencyMetadata, error)
		GetDependentsWithMetadataForIssues(ctx context.Context, issueIDs []string) (map[string][]*types.IssueWithDependencyMetadata, error)
	} = (*DoltStore)(nil)
}