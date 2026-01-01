package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"stackit.dev/stackit/internal/git"
)

func TestMetaSerialization(t *testing.T) {
	parent := "main"
	scope := "feat/xyz"
	now := time.Now().UTC().Truncate(time.Second) // JSON unmarshaling might lose sub-second precision

	meta := &git.Meta{
		ParentBranchName: &parent,
		Scope:            &scope,
		LockReason:       "manual",
		BranchType:       git.BranchTypeUser,
		LastModifiedBy: &git.ModifiedBy{
			GitName:  "John Doe",
			GitEmail: "john@example.com",
		},
		LastModifiedAt: &now,
	}

	// Marshal
	data, err := json.Marshal(meta)
	assert.NoError(t, err)

	// Unmarshal
	var meta2 git.Meta
	err = json.Unmarshal(data, &meta2)
	assert.NoError(t, err)

	assert.Equal(t, meta.ParentBranchName, meta2.ParentBranchName)
	assert.Equal(t, meta.Scope, meta2.Scope)
	assert.Equal(t, meta.LockReason, meta2.LockReason)
	assert.Equal(t, meta.BranchType, meta2.BranchType)
	assert.Equal(t, meta.LastModifiedBy.GitName, meta2.LastModifiedBy.GitName)
	assert.Equal(t, meta.LastModifiedBy.GitEmail, meta2.LastModifiedBy.GitEmail)
	assert.True(t, meta.LastModifiedAt.Equal(*meta2.LastModifiedAt))
}

func TestLocalMetaSerialization(t *testing.T) {
	meta := &git.LocalMeta{
		Frozen: true,
	}

	// Marshal
	data, err := json.Marshal(meta)
	assert.NoError(t, err)

	// Unmarshal
	var meta2 git.LocalMeta
	err = json.Unmarshal(data, &meta2)
	assert.NoError(t, err)

	assert.Equal(t, meta.Frozen, meta2.Frozen)
}

func TestMetaBackwardCompatibility(t *testing.T) {
	// Old metadata format (no new fields)
	jsonData := `{"parentBranchName":"main","lockReason":"locked"}`

	var meta git.Meta
	err := json.Unmarshal([]byte(jsonData), &meta)
	assert.NoError(t, err)

	assert.Equal(t, "main", *meta.ParentBranchName)
	assert.Equal(t, "locked", meta.LockReason)
	assert.Equal(t, git.BranchType(""), meta.BranchType) // Should be empty/default
	assert.Nil(t, meta.LastModifiedBy)
	assert.Nil(t, meta.LastModifiedAt)
}
