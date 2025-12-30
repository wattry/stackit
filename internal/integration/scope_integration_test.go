package integration

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/submit"
	"stackit.dev/stackit/internal/actions/sync"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

// noopHandler is a test handler that ignores all events
type noopHandler struct{}

func (h *noopHandler) OnEvent(_ submit.Event) {}
func (h *noopHandler) Confirm(_ string, defaultYes bool) (bool, error) {
	return defaultYes, nil
}

func TestScopeSubmitSyncFlow(t *testing.T) {
	sh := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

	// 1. Create a branch
	sh.CreateBranch("feature").
		CommitChange("file", "content").
		TrackBranch("feature", "main")

	eng := sh.Engine
	branch := eng.GetBranch("feature")

	// 2. Set a scope locally
	require.NoError(t, eng.SetScope(branch, engine.NewScope("JIRA-123")))
	require.Equal(t, "JIRA-123", eng.GetScope(branch).String())

	// 3. Setup a local remote to simulate "origin"
	_, err := sh.Scene.Repo.CreateBareRemote("origin")
	require.NoError(t, err)

	// 4. Setup mocked GitHub client for submit
	config := testhelpers.NewMockGitHubServerConfig()
	rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, config)
	githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, config)
	sh.Context.GitHubClient = githubClient

	// 5. Submit the branch (this should push the metadata ref)
	opts := submit.Options{
		NoEdit: true,
		Draft:  true,
	}
	err = submit.Action(sh.Context, opts, &noopHandler{})
	require.NoError(t, err)

	// 6. Run sync to fetch remote metadata
	// This will fetch refs/stackit/metadata/*:refs/stackit/remote-metadata/*
	err = sync.Action(sh.Context, sync.Options{})
	require.NoError(t, err)

	// 7. Verify the scope is now in the remote metadata cache
	err = eng.LoadRemoteMetadataCache()
	require.NoError(t, err)

	cache := eng.GetRemoteMetadataCache()
	require.NotNil(t, cache["feature"], "Remote metadata for 'feature' should exist in cache")
	require.NotNil(t, cache["feature"].Scope, "Scope should be set in remote metadata")
	require.Equal(t, "JIRA-123", *cache["feature"].Scope, "Remote metadata scope should match JIRA-123")
}
