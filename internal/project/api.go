// MODIFIED IN THIS FORK: 2025-08-25 Aleksei Berezkin 8aa1bd88535b085be08c607ccbecbe4e6fbfeeea Fixed fork after refactoring in upstream
package project

import (
	"context"

	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func (s *Session) APIOpenProject(ctx context.Context, configFileName string, apiFileChanges FileChangeSummary) (*Project, *Snapshot, error) {
	s.snapshotUpdateMu.Lock()
	defer s.snapshotUpdateMu.Unlock()

	fileChanges, overlays, ataChanges, _ := s.flushChanges(ctx)
	mergeFileChangeSummary(&fileChanges, apiFileChanges)
	newSnapshot := s.UpdateSnapshot(ctx, overlays, SnapshotChange{
		fileChanges: fileChanges,
		ataChanges:  ataChanges,
		apiRequest: &APISnapshotRequest{
			OpenProjects: collections.NewSetFromItems(configFileName),
		},
	})

	if newSnapshot.apiError != nil {
		return nil, newSnapshot, newSnapshot.apiError
	}

	project := newSnapshot.ProjectCollection.ConfiguredProject(s.toPath(configFileName))
	if project == nil {
		panic("OpenProject request returned no error but project not present in snapshot")
	}

	return project, newSnapshot, nil
}

// APIUpdateWithFileChanges creates a new snapshot incorporating the given file changes.
func (s *Session) APIUpdateWithFileChanges(ctx context.Context, apiFileChanges FileChangeSummary) *Snapshot {
	s.snapshotUpdateMu.Lock()
	defer s.snapshotUpdateMu.Unlock()

	fileChanges, overlays, ataChanges, _ := s.flushChanges(ctx)
	mergeFileChangeSummary(&fileChanges, apiFileChanges)

	newSnapshot := s.UpdateSnapshot(ctx, overlays, SnapshotChange{
		apiRequest:  &APISnapshotRequest{},
		fileChanges: fileChanges,
		ataChanges:  ataChanges,
	})

	return newSnapshot
}

// CloseProject - for JB fork, because flushChanges is private
func (s *Session) CloseProject(ctx context.Context, configFileName string) error {
	fileChanges, overlays, ataChanges, _ := s.flushChanges(ctx)
	newSnapshot := s.UpdateSnapshot(ctx, overlays, SnapshotChange{
		fileChanges: fileChanges,
		ataChanges:  ataChanges,
		apiRequest: &APISnapshotRequest{
			CloseProjects: collections.NewSetFromItems[tspath.Path](s.toPath(configFileName)),
		},
	})

	return newSnapshot.apiError
}
