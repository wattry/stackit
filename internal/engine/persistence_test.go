package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetaSerialization(t *testing.T) {
	parent := "main"
	scope := "feat/xyz"
	now := time.Now().UTC().Truncate(time.Second) // JSON unmarshaling might lose sub-second precision

	meta := &Meta{
		ParentBranchName: &parent,
		Scope:            &scope,
		Locked:           true,
		BranchType:       BranchTypeUser,
		LastModifiedBy: &ModifiedBy{
			GitName:  "John Doe",
			GitEmail: "john@example.com",
		},
		LastModifiedAt: &now,
	}

	// Marshal
	data, err := json.Marshal(meta)
	assert.NoError(t, err)

	// Unmarshal
	var meta2 Meta
	err = json.Unmarshal(data, &meta2)
	assert.NoError(t, err)

	assert.Equal(t, meta.ParentBranchName, meta2.ParentBranchName)
	assert.Equal(t, meta.Scope, meta2.Scope)
	assert.Equal(t, meta.Locked, meta2.Locked)
	assert.Equal(t, meta.BranchType, meta2.BranchType)
	assert.Equal(t, meta.LastModifiedBy.GitName, meta2.LastModifiedBy.GitName)
	assert.Equal(t, meta.LastModifiedBy.GitEmail, meta2.LastModifiedBy.GitEmail)
	assert.True(t, meta.LastModifiedAt.Equal(*meta2.LastModifiedAt))
}

func TestMetaBackwardCompatibility(t *testing.T) {
	// Old metadata format (no new fields)
	jsonData := `{"parentBranchName":"main","locked":true}`

	var meta Meta
	err := json.Unmarshal([]byte(jsonData), &meta)
	assert.NoError(t, err)

	assert.Equal(t, "main", *meta.ParentBranchName)
	assert.True(t, meta.Locked)
	assert.Equal(t, BranchType(""), meta.BranchType) // Should be empty/default
	assert.Nil(t, meta.LastModifiedBy)
	assert.Nil(t, meta.LastModifiedAt)
}
